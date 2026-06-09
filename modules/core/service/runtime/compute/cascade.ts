// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../../orm/metadata/storage';
import { ComputeEngine } from './engine';
import type { ModelMetadata, CollectionPathDep } from '../../orm/metadata/model';
import type BaseModel from '../../orm/model/model';
import type { ParentComputeTrigger } from './types';
import type { BaseQueryCondition, FieldSelection } from '../../api';
import { buildComputeGraph } from './graph';
import { getRuntimeRepository } from '../runtime_repository_facade';
import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';

type EntityRecord = ObjectRecord;

function getEntityField(entity: EntityRecord | undefined, key: string): unknown {
  return entity?.[key];
}

function getEntityId(entity?: EntityRecord): string {
  const raw = getEntityField(entity, 'Id');
  return raw == null ? '' : String(raw).trim();
}

/**
 * Context for multi-level trigger propagation.
 */
interface CascadeContext {
  depth: number; // Current recursion depth.
  maxDepth: number; // Maximum recursion depth, defaulting to 5.
  visited: Set<string>; // Processed record keys in the form `${modelKey}#${id}`.
  path: string[]; // Trigger path used for diagnostics.
}

export interface UpstreamChangeEvent {
  childCtor: typeof BaseModel;
  operation: 'create' | 'update' | 'delete';
  changedFields: string[];
  beforeEntity?: EntityRecord;
  afterEntity?: EntityRecord;
}

export interface UpstreamObservabilityStats {
  upstreamEventCount: number;
  recursiveTriggerCount: number;
  dedupParentIdCount: number;
  dedupTriggerCount: number;
  parentBatchQueryCount: number;
  collectionBatchQueryCount: number;
}

/**
 * Multi-level compute cascade engine.
 *
 * Responsibilities:
 * - Recompute parent-model compute fields when child rows change, are created, or are deleted.
 * - Recompute child-model compute fields when parent scalar fields change.
 * - Support multi-level propagation such as grandchild -> parent -> grandparent with cycle and depth guards.
 * - Group work, prefetch dependencies, detect changes, and isolate failures.
 */
export class ComputeCascadeEngine {
  private static warmedModelCount = -1;

  private static upstreamStats: UpstreamObservabilityStats = {
    upstreamEventCount: 0,
    recursiveTriggerCount: 0,
    dedupParentIdCount: 0,
    dedupTriggerCount: 0,
    parentBatchQueryCount: 0,
    collectionBatchQueryCount: 0,
  };

  static resetUpstreamStats() {
    this.upstreamStats = {
      upstreamEventCount: 0,
      recursiveTriggerCount: 0,
      dedupParentIdCount: 0,
      dedupTriggerCount: 0,
      parentBatchQueryCount: 0,
      collectionBatchQueryCount: 0,
    };
  }

  static getUpstreamStats(): UpstreamObservabilityStats {
    return { ...this.upstreamStats };
  }

  private static createCascadeContext(): CascadeContext {
    return {
      depth: 0,
      maxDepth: 5,
      visited: new Set(),
      path: [],
    };
  }

  private static ensureAllComputeGraphsBuilt() {
    const storage = asObjectRecord(MetadataStorage.instance);
    const models = storage?.models;
    if (!(models instanceof Map)) return;
    const size = models.size;
    if (size >= 0 && size === this.warmedModelCount) return;

    for (const [ctor] of models.entries() as IterableIterator<[typeof BaseModel, ModelMetadata]>) {
      const m = MetadataStorage.instance.getModelMetadata(ctor);
      if (!m.computeGraph) {
        m.computeGraph = buildComputeGraph(m);
      }
    }

    if (size >= 0) this.warmedModelCount = size;
  }

  static collectUpstreamInverseFields(childCtor: typeof BaseModel): string[] {
    // Warm the global computeGraph surface so child reverse indexes do not only see already-built parent graphs.
    this.ensureAllComputeGraphsBuilt();

    const childMeta = MetadataStorage.instance.getModelMetadata(childCtor);
    childMeta.computeGraph = buildComputeGraph(childMeta);
    const reverseIndex = childMeta.computeGraph?.reverseComputeIndex;
    if (!reverseIndex) return [];

    const out = new Set<string>();
    reverseIndex.forEach(triggers => {
      for (const t of triggers) {
        if (t?.inverseField) out.add(String(t.inverseField));
      }
    });
    return [...out];
  }

  static async triggerUpstream(event: UpstreamChangeEvent): Promise<void> {
    const { childCtor, operation, beforeEntity, afterEntity } = event;
    this.upstreamStats.upstreamEventCount += 1;
    const changedFields = [...new Set((event.changedFields || []).filter(Boolean))];
    const inverseFields = this.collectUpstreamInverseFields(childCtor);

    const toIdSet = (entity?: EntityRecord) => {
      const map = new Map<string, Set<string>>();
      if (!entity) return map;
      for (const f of inverseFields) {
        const raw = entity[f];
        const id = raw == null ? '' : String(raw).trim();
        if (!id) continue;
        if (!map.has(f)) map.set(f, new Set<string>());
        map.get(f)!.add(id);
      }
      return map;
    };

    const mergeMaps = (a: Map<string, Set<string>>, b: Map<string, Set<string>>) => {
      const out = new Map<string, Set<string>>();
      let duplicateCount = 0;
      const add = (k: string, v: Set<string>) => {
        if (!out.has(k)) out.set(k, new Set<string>());
        v.forEach(x => {
          const bucket = out.get(k)!;
          if (bucket.has(x)) {
            duplicateCount += 1;
          } else {
            bucket.add(x);
          }
        });
      };
      a.forEach((v, k) => add(k, v));
      b.forEach((v, k) => add(k, v));
      if (duplicateCount > 0) {
        this.upstreamStats.dedupParentIdCount += duplicateCount;
      }
      return out;
    };

    const isSameSet = (a?: Set<string>, b?: Set<string>) => {
      const sa = a || new Set<string>();
      const sb = b || new Set<string>();
      if (sa.size !== sb.size) return false;
      for (const x of sa) if (!sb.has(x)) return false;
      return true;
    };

    const beforeMap = toIdSet(beforeEntity);
    const afterMap = toIdSet(afterEntity);

    // 1) Propagate field changes upstream for update operations.
    if (operation === 'update' && changedFields.length) {
      const recordId = getEntityId(afterEntity) || getEntityId(beforeEntity);
      if (recordId) {
        await this.triggerRecursive(childCtor, changedFields, recordId, afterEntity || beforeEntity, 'field-change', this.createCascadeContext());
      }
    }

    // 2) Propagate lifecycle events upstream for create and delete.
    if (operation === 'create' || operation === 'delete') {
      const targetEntity = operation === 'create' ? afterEntity : beforeEntity;
      const recordId = getEntityId(targetEntity);
      const entity = operation === 'create' ? afterEntity : beforeEntity;
      const parentMap = operation === 'create' ? afterMap : beforeMap;
      if (recordId && entity) {
        await this.triggerRecursive(childCtor, ['__lifecycle'], recordId, entity, 'lifecycle', this.createCascadeContext(), parentMap);
      }
    }

    // 3) Propagate membership changes upstream for create, delete, and re-parent operations.
    const membershipChangedFields: string[] = [];
    if (operation === 'create') {
      for (const f of inverseFields) {
        if ((afterMap.get(f)?.size || 0) > 0) membershipChangedFields.push(f);
      }
    } else if (operation === 'delete') {
      for (const f of inverseFields) {
        if ((beforeMap.get(f)?.size || 0) > 0) membershipChangedFields.push(f);
      }
    } else {
      for (const f of inverseFields) {
        if (!isSameSet(beforeMap.get(f), afterMap.get(f))) membershipChangedFields.push(f);
      }
    }

    if (membershipChangedFields.length) {
      const recordId = getEntityId(afterEntity) || getEntityId(beforeEntity);
      if (recordId) {
        const merged = mergeMaps(beforeMap, afterMap);
        await this.triggerRecursive(
          childCtor,
          membershipChangedFields,
          recordId,
          afterEntity || beforeEntity,
          'membership-change',
          this.createCascadeContext(),
          merged
        );
      }
    }
  }

  static async triggerUpstreamCreateBatch(childCtor: typeof BaseModel, afterEntities: EntityRecord[]): Promise<void> {
    const rows = (afterEntities || []).filter(Boolean);
    if (!rows.length) return;
    this.upstreamStats.upstreamEventCount += 1;

    const inverseFields = this.collectUpstreamInverseFields(childCtor);
    if (!inverseFields.length) return;

    const parentMap = new Map<string, Set<string>>();
    for (const row of rows) {
      for (const f of inverseFields) {
        const raw = row[f];
        const id = raw == null ? '' : String(raw).trim();
        if (!id) continue;
        if (!parentMap.has(f)) parentMap.set(f, new Set<string>());
        parentMap.get(f)!.add(id);
      }
    }

    const hasAnyParent = Array.from(parentMap.values()).some(set => set.size > 0);
    if (!hasAnyParent) return;

    const sample = rows[0];
    const sampleId = getEntityId(sample) || '__batch_create__';

    await this.triggerRecursive(childCtor, ['__lifecycle'], sampleId, sample, 'lifecycle', this.createCascadeContext(), parentMap);

    const membershipFields = inverseFields.filter(f => (parentMap.get(f)?.size || 0) > 0);
    if (membershipFields.length) {
      await this.triggerRecursive(childCtor, membershipFields, sampleId, sample, 'membership-change', this.createCascadeContext(), parentMap);
    }
  }

  /**
   * Propagate child changes to parent computes, including multi-level propagation.
   *
   * @param childCtor Child model constructor.
   * @param changedFields Changed field list, or ['__lifecycle'] for create and delete.
   * @param childId Child record Id.
   * @param childEntity Child entity when it is already loaded.
   * @param mode Trigger mode.
   */
  static async trigger(
    childCtor: typeof BaseModel,
    changedFields: string[],
    childId: string,
    childEntity?: EntityRecord,
    mode: 'field-change' | 'lifecycle' = 'field-change'
  ): Promise<void> {
    if (mode === 'lifecycle') {
      await this.triggerUpstream({
        childCtor,
        operation: 'delete',
        changedFields: [],
        beforeEntity: childEntity || { Id: childId },
      });
      return;
    }

    await this.triggerUpstream({
      childCtor,
      operation: 'update',
      changedFields,
      afterEntity: childEntity || { Id: childId },
    });
  }

  /**
   * Propagate parent scalar changes to child-model compute fields and persist the recomputed values.
   *
   * Use this after the parent model finishes its first write pass in static Update or UpdateById.
   *
   * @param parentCtor Parent model constructor, for example Order.
   * @param changedFields Parent scalar fields changed in the current write, for example ['DiscountRate'].
   * @param parentId Parent record Id.
   */
  static async triggerDownstream(parentCtor: typeof BaseModel, changedFields: string[], parentId: string): Promise<void> {
    if (!changedFields?.length || !parentId) return;

    const storage = asObjectRecord(MetadataStorage.instance);
    const changed = new Set<string>(changedFields.filter(Boolean).map(String));
    const storageModels = storage?.models;
    if (!(storageModels instanceof Map)) return;

    const models = storageModels.entries() as IterableIterator<[typeof BaseModel, ModelMetadata]>;

    for (const [childCtor /*, childMetaRaw*/] of models as IterableIterator<[typeof BaseModel, ModelMetadata]>) {
      // 1) Refresh metadata and ensure computeGraph is built.
      const childMeta = MetadataStorage.instance.getModelMetadata(childCtor);
      if (!childMeta.computeGraph) {
        try {
          childMeta.computeGraph = buildComputeGraph(childMeta);
        } catch (e) {
          console.warn('[CascadeDown] failed to build child model computeGraph:', childMeta.fullModelName || childCtor.name, e);
          continue;
        }
      }
      const g = childMeta.computeGraph;
      if (!g) continue;

      // 2) Collect ManyToOne roots that point to the parent model.
      const m2oRoots: string[] = [];
      childMeta.fields.forEach((f, name) => {
        if (f?.type === 'ManyToOne' && f.relation?.targetModel?.() === parentCtor) {
          m2oRoots.push(name);
        }
      });
      if (!m2oRoots.length) continue;

      // 3) Filter affected compute fields from computePathDeps with a root match and a changed first chain segment.
      const affectedByRoot = new Map<string, Set<string>>();
      for (const cf of g.computePathDeps?.keys?.() || []) {
        const deps = g.computePathDeps!.get(cf) || [];
        for (const d of deps) {
          const root = d.root;
          const chain: string[] = Array.isArray(d.chain) ? d.chain : [];
          if (!root || !m2oRoots.includes(root) || chain.length === 0) continue;
          if (changed.has(chain[0])) {
            if (!affectedByRoot.has(root)) affectedByRoot.set(root, new Set<string>());
            affectedByRoot.get(root)!.add(cf);
          }
        }
      }
      if (!affectedByRoot.size) continue;

      const childRepo = getRuntimeRepository(childCtor);

      // 4) Scan and recompute one root at a time.
      for (const [root, computeSet] of affectedByRoot.entries()) {
        const needed = new Set<string>(['Id', root]);

        // Scalar dependencies.
        computeSet.forEach(cf => g.computeScalarDeps?.get(cf)?.forEach(dep => needed.add(dep)));

        // Include compute results themselves for diffing.
        computeSet.forEach(cf => needed.add(cf));

        // Include scaleField for decimal compute fields on the child model.
        computeSet.forEach(cf => {
          const cfMeta = childMeta.fields.get(cf);
          if (cfMeta?.type === 'decimal') {
            const spec = asObjectRecord(cfMeta.column) || asObjectRecord(cfMeta.select) || {};
            const s = spec.scaleField;
            if (typeof s === 'string' && s) needed.add(s);
          }
        });

        // Filter child rows by inverseField, which is the root here.
        const condition: BaseQueryCondition = [root, '=', parentId];
        const rows = await childRepo.search(condition, {
          fields: Array.from(needed) as FieldSelection<EntityRecord>,
        });
        if (!rows?.length) continue;

        for (const row of rows) {
          const entityObj: EntityRecord = { ...row };
          const oldVals = new Map<string, unknown>();
          computeSet.forEach(cf => oldVals.set(cf, entityObj[cf]));

          // Use root as the trigger source so ComputeEngine can prefetch the parent-side ManyToOne path.
          const seed = new Set<string>([root]);

          try {
            await ComputeEngine.recompute(childMeta, entityObj, seed, 'persist');
          } catch (e) {
            console.warn('[ComputeCascade] downstream recompute failed; row skipped:', e);
            continue;
          }

          // Write back precisely by persisting only changed fields.
          const updates: EntityRecord = {};
          let changedAny = false;
          for (const cf of computeSet) {
            if (cf in entityObj && entityObj[cf] !== oldVals.get(cf)) {
              updates[cf] = entityObj[cf];
              changedAny = true;
            }
          }

          if (changedAny) {
            // Include the matching scaleField for changed decimal fields.
            for (const cf of computeSet) {
              const cfMeta = childMeta.fields.get(cf);
              if (cfMeta?.type !== 'decimal') continue;
              const spec = asObjectRecord(cfMeta.column) || asObjectRecord(cfMeta.select) || {};
              const sField = typeof spec.scaleField === 'string' ? spec.scaleField : undefined;
              if (sField && entityObj[sField] !== undefined && !(sField in updates)) {
                updates[sField] = entityObj[sField];
              }
            }

            updates.UpdatedAt = new Date();
            await childRepo.update(updates, ['Id', '=', String(entityObj.Id ?? '')]);
          }
        }
      }
    }
  }

  /**
   * Recursively propagate child changes to parent computes using depth-first traversal.
   */
  private static async triggerRecursive(
    modelCtor: typeof BaseModel,
    changedFields: string[],
    recordId: string,
    entity: EntityRecord | undefined,
    mode: 'field-change' | 'lifecycle' | 'membership-change',
    ctx: CascadeContext,
    parentIdOverrideByInverse?: Map<string, Set<string>>
  ): Promise<void> {
    this.upstreamStats.recursiveTriggerCount += 1;

    // 1. Cycle guard.
    const meta = MetadataStorage.instance.getModelMetadata(modelCtor);
    const modelKey = this.getModelKey(meta);
    const visitKey = `${modelKey}#${recordId}`;

    if (ctx.depth >= ctx.maxDepth) {
      console.warn(`[ComputeCascade] reached max depth ${ctx.maxDepth}; stopping propagation. Path: ${ctx.path.join(' -> ')} -> ${visitKey}`);
      return;
    }

    if (ctx.visited.has(visitKey)) {
      console.warn(`[ComputeCascade] detected cyclic trigger: ${visitKey}; skipping. Path: ${ctx.path.join(' -> ')}`);
      return;
    }
    ctx.visited.add(visitKey);

    // 2. Look up the reverse index from computeGraph.reverseComputeIndex.
    const graph = meta.computeGraph;
    if (!graph?.reverseComputeIndex) return;

    const allTriggers: ParentComputeTrigger[] = [];
    if (mode === 'lifecycle') {
      // Lifecycle triggers for create and delete.
      allTriggers.push(...(graph.reverseComputeIndex.get('__lifecycle') || []).filter(t => t.triggerMode === 'lifecycle'));
    } else {
      // Field-change triggers.
      for (const field of changedFields) {
        allTriggers.push(...(graph.reverseComputeIndex.get(field) || []).filter(t => t.triggerMode === mode));
      }
    }

    if (!allTriggers.length) return;

    const distinctTriggerKeys = new Set<string>();
    for (const t of allTriggers) {
      const tParentKey = this.getModelKey(MetadataStorage.instance.getModelMetadata(t.parentModelCtor));
      distinctTriggerKeys.add(`${tParentKey}#${t.inverseField}#${t.parentComputeField}#${t.triggerMode}`);
    }
    const dedupTrigger = allTriggers.length - distinctTriggerKeys.size;
    if (dedupTrigger > 0) {
      this.upstreamStats.dedupTriggerCount += dedupTrigger;
    }

    // 3. Group by parent model to avoid duplicate parent-record work.
    const parentGroups = new Map<
      string,
      {
        parentCtor: typeof BaseModel;
        inverseField: string;
        computeFields: Set<string>;
      }
    >();

    for (const t of allTriggers) {
      const parentKey2 = this.getModelKey(MetadataStorage.instance.getModelMetadata(t.parentModelCtor));
      const groupKey = `${parentKey2}#${t.inverseField}`;
      if (!parentGroups.has(groupKey)) {
        parentGroups.set(groupKey, {
          parentCtor: t.parentModelCtor,
          inverseField: t.inverseField,
          computeFields: new Set(),
        });
      }
      parentGroups.get(groupKey)!.computeFields.add(t.parentComputeField);
    }

    // 4. Process each parent model.
    for (const [parentKey2, group] of parentGroups.entries()) {
      try {
        // 4.1 Gather parent Ids, preferring overrides to support re-parent flows.
        const parentIdSet = new Set<string>();
        const overridden = parentIdOverrideByInverse?.get(group.inverseField);
        if (overridden && overridden.size) {
          overridden.forEach(x => {
            const v = String(x || '').trim();
            if (v) parentIdSet.add(v);
          });
        } else {
          if (!entity) {
            // If the entity is missing, fetch the child record first.
            const repo = getRuntimeRepository(modelCtor);
            const rows = await repo.search(['Id', '=', recordId], {
              fields: ['Id', group.inverseField] as FieldSelection<EntityRecord>,
            });
            if (!rows.length) continue;
            entity = rows[0] as EntityRecord;
          }
          const parentIdRaw = entity[group.inverseField];
          const parentId = String(parentIdRaw || '').trim();
          if (parentId) parentIdSet.add(parentId);
        }

        if (!parentIdSet.size) continue;

        const parentIds = Array.from(parentIdSet);

        // 4.2 Gather collection dependencies and affected collections from the compute fields.
        const parentMeta = MetadataStorage.instance.getModelMetadata(group.parentCtor);
        const parentGraph = parentMeta.computeGraph;
        if (!parentGraph) continue;

        // 4.2.0 Aggregate collection dependencies as collection -> chains.
        const collDepsMap = new Map<string, Set<string>>();
        for (const cf of group.computeFields) {
          const deps: CollectionPathDep[] = parentGraph.computeCollectionPathDeps?.get(cf) || [];
          for (const d of deps) {
            if (!collDepsMap.has(d.collection)) collDepsMap.set(d.collection, new Set<string>());
            collDepsMap.get(d.collection)!.add(d.chain && d.chain.length ? d.chain.join('.') : 'Id');
          }
        }

        // 4.2.1 Run BFS from collection roots to get affected compute fields for query and diff.
        const affectedCompute = new Set<string>();
        {
          const baseSeedTmp = new Set<string>([...collDepsMap.keys()]);
          const queue: string[] = [];
          const seen = new Set<string>();
          baseSeedTmp.forEach(src => {
            const arr = parentGraph.fastReverseDeps.get(src);
            if (arr)
              arr.forEach(cf => {
                if (!seen.has(cf)) {
                  seen.add(cf);
                  queue.push(cf);
                }
              });
          });
          for (let i = 0; i < queue.length; i++) {
            const cf = queue[i];
            affectedCompute.add(cf);
            const next = parentGraph.fastReverseDeps.get(cf);
            if (next)
              next.forEach(n => {
                if (!seen.has(n)) {
                  seen.add(n);
                  queue.push(n);
                }
              });
          }
        }

        // 4.2.2 Build the parent fields to query: Id plus affected compute fields and their scalar dependencies.
        const neededFields = new Set<string>(['Id']);
        affectedCompute.forEach(cf => {
          neededFields.add(cf);
          parentGraph.computeScalarDeps?.get(cf)?.forEach(f => neededFields.add(f));
        });

        const parentRepo = getRuntimeRepository(group.parentCtor);
        const parentCondition: BaseQueryCondition = parentIds.length === 1 ? ['Id', '=', parentIds[0]] : ['Id', 'in', parentIds];
        if (parentIds.length > 1) {
          this.upstreamStats.parentBatchQueryCount += 1;
        }
        const parentRows = await parentRepo.search(parentCondition, {
          fields: Array.from(neededFields) as FieldSelection<EntityRecord>,
        });

        if (!parentRows.length) continue;
        const parentEntityById = new Map<string, EntityRecord>();
        for (const row of parentRows) {
          const rowEntity = row as EntityRecord;
          const id = getEntityId(rowEntity);
          if (id) parentEntityById.set(id, rowEntity);
        }
        if (!parentEntityById.size) continue;

        // 4.2.3 Prefetch collection dependencies in batches and attach them to the parent entity to avoid fallback when collections are missing.
        try {
          for (const [coll, chains] of collDepsMap.entries()) {
            const fieldMeta = asObjectRecord(parentMeta.fields.get(coll));
            const rel = asObjectRecord(fieldMeta?.relation);
            const targetModelFn = rel?.targetModel;
            const childCtor: typeof BaseModel | undefined = typeof targetModelFn === 'function' ? targetModelFn() : undefined;
            const inverseFieldRaw = rel?.inverseField || rel?.fkField || rel?.foreignKey || rel?.refField;
            const inverseField: string | undefined = typeof inverseFieldRaw === 'string' ? inverseFieldRaw : undefined;

            if (!childCtor || !inverseField) continue;

            const childRepo = getRuntimeRepository(childCtor);
            const select = new Set<string>(['Id', inverseField]);
            chains.forEach(p => p && select.add(p));

            const collCondition: BaseQueryCondition = parentIds.length === 1 ? [inverseField, '=', parentIds[0]] : [inverseField, 'in', parentIds];
            if (parentIds.length > 1) {
              this.upstreamStats.collectionBatchQueryCount += 1;
            }
            const rows = await childRepo.search(collCondition, {
              fields: Array.from(select) as FieldSelection<EntityRecord>,
            });

            const grouped = new Map<string, EntityRecord[]>();
            for (const item of rows || []) {
              const rowEntity = item as EntityRecord;
              const ownerId = String(rowEntity[inverseField] || '').trim();
              if (!ownerId) continue;
              if (!grouped.has(ownerId)) grouped.set(ownerId, []);
              grouped.get(ownerId)!.push(rowEntity);
            }

            for (const [id, parentEntity] of parentEntityById.entries()) {
              parentEntity[coll] = grouped.get(id) || [];
            }
          }
        } catch (e) {
          console.warn('[ComputeCascade] collection dependency prefetch failed; falling back to scalar dependencies only:', e);
        }

        // 4.3 Recompute parent compute fields with the collection root as the true trigger source.
        const baseSeed = new Set<string>([...collDepsMap.keys()]);

        for (const [parentId, parentEntity] of parentEntityById.entries()) {
          // Preserve old values for change detection, limited to affected compute fields.
          const oldValues = new Map<string, unknown>();
          affectedCompute.forEach(f => {
            oldValues.set(f, parentEntity[f]);
          });

          // Recompute in persist mode.
          await ComputeEngine.recompute(parentMeta, parentEntity, baseSeed, 'persist');

          // 4.4 Detect changes and persist only the affected compute fields.
          const updates: EntityRecord = {};
          let hasChanges = false;
          for (const f of affectedCompute) {
            if (f in parentEntity && parentEntity[f] !== oldValues.get(f)) {
              updates[f] = parentEntity[f];
              hasChanges = true;
            }
          }

          if (hasChanges) {
            updates.UpdatedAt = new Date();
            await parentRepo.update(updates, ['Id', '=', parentId]);

            // 4.5 Continue recursively because parent changes may affect grandparents.
            const nextChangedFields = Object.keys(updates).filter(k => k !== 'UpdatedAt');
            if (nextChangedFields.length) {
              await this.triggerRecursive(group.parentCtor, nextChangedFields, String(parentId), parentEntity, 'field-change', {
                depth: ctx.depth + 1,
                maxDepth: ctx.maxDepth,
                visited: ctx.visited,
                path: [...ctx.path, visitKey],
              });
            }
          }
        }
      } catch (error) {
        // Error isolation: parent recompute failures must not block child-record updates.
        console.error(`[ComputeCascade] failed to process parent record: ${parentKey2}`, error);
      }
    }
  }

  /**
   * Get the model key, preferring fullModelName.
   */
  private static getModelKey(meta: ModelMetadata): string {
    return (meta.fullModelName || meta.modelName || meta.className || 'Unknown') as string;
  }
}
