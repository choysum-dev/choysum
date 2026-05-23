// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata, ModelComputeGraph, ParsedDep, PathDep, CollectionPathDep } from '../../orm/metadata/model';
import { MetadataStorage } from '../../orm/metadata/storage';
import type BaseModel from '../../orm/model/model';
import type { ParentComputeTrigger } from './types';
import { parseDeps } from './parser';
import { asObjectRecord } from '@/core/utils/object';

function getRegisteredModelCtors(storage: MetadataStorage): Array<typeof BaseModel> {
  const models = (storage as unknown as { models?: unknown }).models;
  if (!(models instanceof Map)) return [];

  return Array.from(models.keys()).filter((ctor): ctor is typeof BaseModel => typeof ctor === 'function');
}

/**
 * Build the compute graph for a model.
 *  - Enumerate fields that define column.compute.
 *  - Parse dependencies.
 *  - Build reverseDeps from trigger source to affected compute fields.
 *  - Build collectionTouchIndex for computes triggered by collection fields.
 *  - Topologically sort compute fields and detect cycles.
 */
export function buildComputeGraph(meta: ModelMetadata): ModelComputeGraph | undefined {
  if (!meta.fields || !meta.fields.size) return;
  const modelLabel = String(meta.fullModelName || meta.modelName || meta.className || 'Unknown');

  const computeFields: string[] = [];
  const persistedComputeFields = new Set<string>();
  const virtualComputeFields = new Set<string>();
  meta.fields.forEach((f, name) => {
    if (!f.column?.compute) return;
    computeFields.push(name);
    if (f.column.compute.store === false) {
      virtualComputeFields.add(name);
    } else {
      persistedComputeFields.add(name);
    }
  });

  /**
   * Build the reverse trigger index for this model when it acts as a child model.
   *
   * Iterate through all registered models, find parent compute fields whose parsed dependencies
   * reference the current child model, and group them by trigger field.
   *
   * @param childCtor Current child model constructor.
   * @param childMeta Current child model metadata.
   * @returns Reverse index keyed by field name or '__lifecycle'.
   */
  function buildReverseComputeIndex(childCtor: typeof BaseModel, childMeta: ModelMetadata): Map<string, ParentComputeTrigger[]> {
    const index = new Map<string, ParentComputeTrigger[]>();
    const storage = MetadataStorage.instance;

    // Scan all registered models for parent computes that depend on the current child model.
    for (const parentCtor of getRegisteredModelCtors(storage)) {
      const parentMeta = MetadataStorage.instance.getModelMetadata(parentCtor as typeof BaseModel);
      const graph = parentMeta.computeGraph;
      if (!graph?.parsedDeps) continue;

      // Walk every compute field on the parent model.
      for (const [computeField, deps] of graph.parsedDeps.entries()) {
        for (const d of deps) {
          // Only collection-style dependencies participate in the reverse index.
          if (d.kind === 'collectionPath' || d.kind === 'collection') {
            const collectionFieldName = d.collection;
            const collectionMeta = parentMeta.fields.get(collectionFieldName);

            if (!collectionMeta?.relation?.targetModel) {
              continue;
            }

            const targetCtor = collectionMeta.relation.targetModel();
            if (targetCtor !== childCtor) {
              continue;
            }

            // Read the inverse field used to link the collection rows back to the parent.
            const inverseField = collectionMeta.relation.inverseField as string;
            if (!inverseField) {
              if (typeof console !== 'undefined') {
                console.warn(
                  `[buildReverseComputeIndex] collection field ${
                    parentMeta.fullModelName || parentMeta.modelName
                  }.${collectionFieldName} is missing inverseField; skipping reverse-index registration`
                );
              }
              continue;
            }

            // Register trigger rules.
            if (d.kind === 'collectionPath') {
              // Field-change trigger, for example collectionPath: Lines.Quantity.
              for (const seg of d.chain) {
                if (!index.has(seg)) index.set(seg, []);
                addTriggerToIndex(index.get(seg)!, {
                  parentModelCtor: parentCtor,
                  parentComputeField: computeField,
                  inverseField,
                  collectionField: collectionFieldName,
                  triggerMode: 'field-change',
                });
              }

              // Lifecycle trigger for create and delete.
              const lifecycleKey = '__lifecycle';
              if (!index.has(lifecycleKey)) index.set(lifecycleKey, []);
              addTriggerToIndex(index.get(lifecycleKey)!, {
                parentModelCtor: parentCtor,
                parentComputeField: computeField,
                inverseField,
                collectionField: collectionFieldName,
                triggerMode: 'lifecycle',
              });

              // Membership-change trigger for re-parent scenarios.
              if (!index.has(inverseField)) index.set(inverseField, []);
              addTriggerToIndex(index.get(inverseField)!, {
                parentModelCtor: parentCtor,
                parentComputeField: computeField,
                inverseField,
                collectionField: collectionFieldName,
                triggerMode: 'membership-change',
              });
            } else {
              // Lifecycle trigger for collection roots such as Lines.
              const lifecycleKey = '__lifecycle';
              if (!index.has(lifecycleKey)) index.set(lifecycleKey, []);
              addTriggerToIndex(index.get(lifecycleKey)!, {
                parentModelCtor: parentCtor,
                parentComputeField: computeField,
                inverseField,
                collectionField: collectionFieldName,
                triggerMode: 'lifecycle',
              });

              // Membership-change trigger for create, delete, or re-parent.
              if (!index.has(inverseField)) index.set(inverseField, []);
              addTriggerToIndex(index.get(inverseField)!, {
                parentModelCtor: parentCtor,
                parentComputeField: computeField,
                inverseField,
                collectionField: collectionFieldName,
                triggerMode: 'membership-change',
              });
            }
          }
        }
      }
    }

    return index;
  }

  /**
   * Add a trigger rule to an index list with deduplication.
   *
   * Two rules are considered the same when they share the same parent model,
   * parent compute field, inverse field, collection field, and trigger mode.
   */
  function addTriggerToIndex(list: ParentComputeTrigger[], trigger: ParentComputeTrigger): void {
    const exists = list.some(
      t =>
        t.parentModelCtor === trigger.parentModelCtor &&
        t.parentComputeField === trigger.parentComputeField &&
        t.inverseField === trigger.inverseField &&
        t.collectionField === trigger.collectionField &&
        t.triggerMode === trigger.triggerMode
    );
    if (!exists) list.push(trigger);
  }

  const computeFieldSet = new Set<string>(computeFields);
  const parsedDeps = new Map<string, ParsedDep[]>();
  const reverseDeps = new Map<string, Set<string>>();
  const collectionTouchIndex = new Map<string, Set<string>>();
  const computePathDeps = new Map<string, PathDep[]>();
  const computeCollectionPathDeps = new Map<string, CollectionPathDep[]>();

  // Parse dependencies and build the first-pass indexes.
  for (const cf of computeFields) {
    const spec = meta.fields.get(cf)!.column!.compute!;
    const baseDeps = parseDeps(meta, cf, spec.deps || []);

    // If a decimal compute field declares scaleField, add that scale carrier implicitly as a dependency.
    try {
      const fm = meta.fields.get(cf);
      const isDecimal = fm?.type === 'decimal';
      const scaleCarrier = asObjectRecord(fm?.column) ?? asObjectRecord(fm?.select);
      const scaleField = isDecimal ? scaleCarrier?.scaleField : undefined;

      if (isDecimal && typeof scaleField === 'string' && scaleField) {
        const already = baseDeps.some(d => d.kind === 'scalar' && d.field === scaleField);
        if (!already) {
          baseDeps.push({ kind: 'scalar', field: scaleField });
        }
      }
    } catch {
      // ignore
    }

    const deps = baseDeps;
    parsedDeps.set(cf, deps);

    const paths: PathDep[] = [];
    const colPaths: CollectionPathDep[] = [];
    for (const d of deps) {
      if (d.kind === 'path') {
        paths.push({ root: d.root, chain: d.chain });
      } else if (d.kind === 'collection') {
        colPaths.push({ collection: d.collection, chain: [] });
      } else if (d.kind === 'collectionPath') {
        colPaths.push({ collection: d.collection, chain: d.chain });
      }
    }
    computePathDeps.set(cf, paths);
    computeCollectionPathDeps.set(cf, colPaths);

    for (const d of deps) {
      const triggerKey = d.kind === 'scalar' ? d.field : d.kind === 'path' ? d.root : d.collection;
      if (!reverseDeps.has(triggerKey)) reverseDeps.set(triggerKey, new Set());
      reverseDeps.get(triggerKey)!.add(cf);

      if (d.kind === 'collection' || d.kind === 'collectionPath') {
        if (!collectionTouchIndex.has(d.collection)) {
          collectionTouchIndex.set(d.collection, new Set());
        }
        collectionTouchIndex.get(d.collection)!.add(cf);
      }
    }
  }

  // Build the dependency graph between compute fields for topological sorting.
  // Edge direction: depCompute -> field.
  const inDegree = new Map<string, number>();
  const adj = new Map<string, Set<string>>();
  for (const f of computeFields) {
    inDegree.set(f, 0);
    adj.set(f, new Set());
  }

  for (const f of computeFields) {
    const deps = parsedDeps.get(f)!;
    for (const d of deps) {
      if (d.kind === 'scalar' && computeFieldSet.has(d.field)) {
        if (persistedComputeFields.has(f) && virtualComputeFields.has(d.field)) {
          throw new Error(`invalid compute dependency (model=${modelLabel}): persisted field ${f} cannot depend on virtual field ${d.field}`);
        }
        // d.field -> f
        adj.get(d.field)!.add(f);
        inDegree.set(f, (inDegree.get(f) || 0) + 1);
      } else if (d.kind === 'path' && computeFieldSet.has(d.root)) {
        if (persistedComputeFields.has(f) && virtualComputeFields.has(d.root)) {
          throw new Error(`invalid compute dependency (model=${modelLabel}): persisted field ${f} cannot depend on virtual field ${d.root}`);
        }
        // d.root -> f
        adj.get(d.root)!.add(f);
        inDegree.set(f, (inDegree.get(f) || 0) + 1);
      }
      // collection and collectionPath cannot themselves be compute fields.
    }
  }

  // Kahn topological sort.
  const queue: string[] = [];
  inDegree.forEach((deg, k) => {
    if (deg === 0) queue.push(k);
  });

  const order: string[] = [];
  while (queue.length) {
    const n = queue.shift()!;
    order.push(n);
    for (const nxt of adj.get(n)!) {
      inDegree.set(nxt, inDegree.get(nxt)! - 1);
      if (inDegree.get(nxt) === 0) queue.push(nxt);
    }
  }

  if (order.length !== computeFields.length) {
    const remain = [...computeFields].filter(f => !order.includes(f));
    throw new Error(`compute fields contain a dependency cycle (model=${modelLabel}): ${remain.join(', ')}`);
  }

  // Build orderIndex and fastReverseDeps for runtime traversal.
  const orderIndex = new Map<string, number>();
  order.forEach((f, i) => orderIndex.set(f, i));

  const fastReverseDeps = new Map<string, string[]>();
  reverseDeps.forEach((set, key) => fastReverseDeps.set(key, [...set]));

  const fastPersistReverseDeps = new Map<string, string[]>();
  reverseDeps.forEach((set, key) => {
    const persistedTargets = [...set].filter(field => persistedComputeFields.has(field));
    if (persistedTargets.length) {
      fastPersistReverseDeps.set(key, persistedTargets);
    }
  });

  // Compute the primary-table scalar dependency set for each compute field.
  const computeScalarDeps = new Map<string, Set<string>>();
  for (const cf of computeFields) {
    const deps = parsedDeps.get(cf)!;
    const set = new Set<string>();
    for (const d of deps) {
      if (d.kind === 'scalar') {
        set.add(d.field);
      } else if (d.kind === 'path') {
        set.add(d.root);
      }
    }
    computeScalarDeps.set(cf, set);
  }

  // Build the reverse trigger index.
  let reverseComputeIndex: Map<string, ParentComputeTrigger[]> | undefined;
  try {
    reverseComputeIndex = buildReverseComputeIndex(meta.type as typeof BaseModel, meta);
  } catch (error) {
    if (typeof console !== 'undefined') {
      console.warn(`[buildComputeGraph] failed to build reverse index: ${modelLabel}`, error);
    }
    reverseComputeIndex = undefined;
  }

  return {
    order,
    reverseDeps,
    parsedDeps,
    collectionTouchIndex,
    computeFields: computeFieldSet,
    persistedComputeFields,
    virtualComputeFields,
    orderIndex,
    fastReverseDeps,
    fastPersistReverseDeps,
    computeScalarDeps,
    computePathDeps,
    computeCollectionPathDeps,
    reverseComputeIndex,
  };
}
