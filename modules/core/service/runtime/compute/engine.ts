// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata, ModelComputeGraph } from '../../orm/metadata/model';
import type { ModelCtor } from '../../orm/metadata/field';
import { MetadataStorage } from '../../orm/metadata';
import type BaseModel from '../../orm/model/model';
import Decimal, { decimalEqual, isDecimal, normalizeDecimalByMeta } from '@/core/utils/decimal';
import { getRuntimeRepository } from '../runtime_repository_facade';
import { withComputeRunAsExecution } from './runas';
import { withBridgeFrame } from './bridge';
import { createEntityBackedModelInstance, resolveInstanceHandler } from './handler_runtime';
import { asObjectRecord, hasOwnKey } from '../../../utils/object';
import type { UnknownRecord } from '../../../utils/types';

// Notes:
// - In preview mode, path prefetching is handled by the Onchange planner, so no IO happens here.
// - In persist mode, this engine performs lightweight multi-hop ManyToOne prefetching based on compute.deps
//   paths so cross-table dependencies such as Line -> Order.DiscountRate can be resolved with only the needed leaves loaded.

/**
 * Compute execution mode.
 */
export type ComputeMode = 'persist' | 'preview';

type RuntimeComputeModelCtor = ModelCtor<BaseModel> & typeof BaseModel;

function isObject(v: unknown): v is UnknownRecord {
  return v !== null && typeof v === 'object';
}

function isPromiseLike(value: unknown): value is PromiseLike<unknown> {
  if (value == null) return false;
  const t = typeof value;
  if (t !== 'object' && t !== 'function') return false;
  return typeof (value as { then?: unknown }).then === 'function';
}

function readBigdecimalEnvelope(value: unknown): string | number | undefined {
  const record = asObjectRecord(value);
  if (!record || !hasOwnKey(record, '$bigdecimal')) return undefined;
  const wrapped = record.$bigdecimal;
  return typeof wrapped === 'string' || typeof wrapped === 'number' ? wrapped : undefined;
}

// Resolve the effective decimal scale: prefer a fixed scale, otherwise read the value from scaleField on the entity.
function resolveDecimalScale(metaField: unknown, entity: unknown): number | undefined {
  const metaRecord = asObjectRecord(metaField);
  const spec = asObjectRecord(metaRecord?.column) ?? {};
  if (typeof spec.scale === 'number') return spec.scale;
  const s = spec.scaleField;
  const entityRecord = asObjectRecord(entity);
  if (typeof s === 'string' && entityRecord && s in entityRecord) {
    const n = Number(entityRecord[s]);
    if (Number.isInteger(n) && n >= 0 && n <= 18) return n;
  }
  return undefined;
}

// Normalize or quantize a Decimal using metadata, including dynamic scaleField support.
function quantizeByMeta(metaField: unknown, entity: UnknownRecord, value: unknown): unknown {
  const metaRecord = asObjectRecord(metaField);
  const spec = asObjectRecord(metaRecord?.column) ?? {};
  const effScale = resolveDecimalScale(metaField, entity);
  const override = (effScale != null ? { column: { ...spec, scale: effScale, round: spec.round, precision: spec.precision } } : metaRecord) as
    | { column?: unknown }
    | undefined;
  try {
    return normalizeDecimalByMeta(override, value) ?? value;
  } catch {
    return value;
  }
}

function ensureSyncBridgeResult(value: unknown, label: string): unknown {
  if (isPromiseLike(value)) {
    throw new Error(`${label} returned a Promise in a sync execution path`);
  }
  return value;
}

type RuntimeComputeExecution = {
  store: boolean;
  runAs: 'user' | 'sudo';
  execute: (modelInstance: BaseModel) => unknown;
};

function resolveRuntimeComputeExecution(meta: ModelMetadata, field: string): RuntimeComputeExecution | undefined {
  const computeHandler = meta.computeHandlers?.get(field);
  if (computeHandler) {
    const method = resolveInstanceHandler(meta, field, computeHandler.method, '@Compute');
    return {
      store: computeHandler.store !== false,
      runAs: computeHandler.runAs === 'sudo' ? 'sudo' : 'user',
      execute: modelInstance => method.call(modelInstance),
    };
  }

  // Fallback: read inline compute expression from field-level column metadata.
  const fieldMeta = meta.fields?.get(field);
  const colSpec = (fieldMeta?.column ?? {}) as UnknownRecord;
  const computeSpec = colSpec.compute as UnknownRecord | undefined;
  const expr = computeSpec?.expr;
  if (typeof expr === 'function') {
    return {
      store: computeSpec?.store !== false,
      runAs: computeSpec?.runAs === 'sudo' ? 'sudo' : 'user',
      execute: modelInstance => (expr as (self: unknown) => unknown).call(modelInstance, modelInstance),
    };
  }

  return;
}

function createSqlBridgeContext(entity: UnknownRecord) {
  const resolveFieldKey = (modelOrKey: unknown, key: unknown): string => {
    if (typeof key === 'string') return key;
    return typeof modelOrKey === 'string' ? modelOrKey : '';
  };

  const throwSqlQueryBridgeUnavailable = (method: 'col' | 'selectFrom'): never => {
    throw new Error(`BRIDGE_CONTEXT_UNAVAILABLE: sql.${method} is unavailable in runtime read context`);
  };

  return {
    field(modelOrKey: unknown, key?: unknown): unknown {
      const resolvedKey = resolveFieldKey(modelOrKey, key);
      if (!resolvedKey) return undefined;
      return entity[resolvedKey];
    },
    fieldExist(modelOrKey: unknown, key?: unknown): boolean {
      const resolvedKey = resolveFieldKey(modelOrKey, key);
      if (!resolvedKey) return false;
      return resolvedKey in entity;
    },
    str: {
      concat: (...items: unknown[]) => items.map(item => String(item ?? '')).join(''),
      lower: (value: unknown) => String(value ?? '').toLowerCase(),
    },
    col() {
      return throwSqlQueryBridgeUnavailable('col');
    },
    selectFrom() {
      return throwSqlQueryBridgeUnavailable('selectFrom');
    },
  };
}

function resolveSqlComputeExecution(meta: ModelMetadata, field: string): ((modelInstance: BaseModel) => unknown) | undefined {
  const sqlHandler = meta.sqlComputeHandlers?.get(field);
  if (!sqlHandler) return;
  const method = resolveInstanceHandler(meta, field, sqlHandler.method, '@SqlCompute');
  return modelInstance => method.call(modelInstance);
}

// In preview mode, normalize decimal fields on the entity and any attached ManyToOne objects as a fallback.
// Prefer normalizeDecimalByMeta to stay aligned with repository behavior, then fall back to best-effort Decimal construction.
function normalizeDecimalFields(meta: ModelMetadata, entity: UnknownRecord) {
  if (!isObject(entity)) return;
  meta.fields.forEach((f, key) => {
    const val = entity[key];
    if (val == null) return;

    if (f.type === 'decimal') {
      // Prefer quantization by dynamic or fixed scale first.
      try {
        entity[key] = quantizeByMeta(f, entity, val);
        return;
      } catch {
        /* fallback to best-effort below */
      }

      // Fallback: if the value is not already Decimal, try {$bigdecimal}, plain strings, or numbers.
      if (!isDecimal(val)) {
        try {
          const wrapped = readBigdecimalEnvelope(val);
          if (wrapped !== undefined) {
            entity[key] = new Decimal(wrapped);
          } else if (typeof val === 'string' || typeof val === 'number') {
            entity[key] = new Decimal(val);
          }
        } catch {
          /* keep original */
        }
      }
      return;
    }

    if (f.type === 'ManyToOne' && isObject(val)) {
      const targetModel = f.relation?.targetModel;
      const ctor = typeof targetModel === 'function' ? (targetModel() as RuntimeComputeModelCtor) : undefined;
      if (ctor) {
        const m = MetadataStorage.instance.getModelMetadata(ctor);
        normalizeDecimalFields(m, val);
      }
    }
  });
}

/**
 * ComputeEngine evaluates compute fields for preview and persist flows.
 */
export class ComputeEngine {
  /**
   * Injects virtual compute-field values into a read result.
   */
  static injectVirtualForRead(meta: ModelMetadata, entity: UnknownRecord, requestedFields?: Iterable<string>): void {
    const g = meta.computeGraph;
    const virtualFields = g?.virtualComputeFields;
    if (!g || !virtualFields || virtualFields.size === 0) return;

    try {
      normalizeDecimalFields(meta, entity);
    } catch {}

    const requested =
      requestedFields == null
        ? undefined
        : new Set(
            Array.from(requestedFields)
              .map(field => String(field || '').trim())
              .filter(Boolean)
          );

    const targets = new Set<string>();
    if (!requested) {
      virtualFields.forEach(field => targets.add(field));
    } else {
      requested.forEach(field => {
        if (virtualFields.has(field)) targets.add(field);
      });
    }
    if (!targets.size) return;

    const required = new Set<string>();
    const includeWithVirtualDeps = (field: string) => {
      if (required.has(field)) return;
      required.add(field);
      const deps = g.parsedDeps.get(field) || [];
      for (const dep of deps) {
        const depCompute = dep.kind === 'scalar' ? dep.field : dep.kind === 'path' ? dep.root : undefined;
        if (depCompute && virtualFields.has(depCompute)) {
          includeWithVirtualDeps(depCompute);
        }
      }
    };
    targets.forEach(includeWithVirtualDeps);

    const orderedSubset = g.order.filter(field => required.has(field));
    const wrapped = createEntityBackedModelInstance(meta, entity);

    for (const field of orderedSubset) {
      const fieldMeta = meta.fields.get(field);
      const runSql = resolveSqlComputeExecution(meta, field);
      if (runSql) {
        const sqlCtx = createSqlBridgeContext(entity);
        let sqlVal = withBridgeFrame(wrapped as object, 'sql', sqlCtx, () => runSql(wrapped));
        sqlVal = ensureSyncBridgeResult(sqlVal, `@SqlCompute(${field})`);
        if (fieldMeta?.type === 'decimal') {
          sqlVal = quantizeByMeta(fieldMeta, entity, sqlVal);
        }
        entity[field] = sqlVal;
        continue;
      }

      const runtimeCompute = resolveRuntimeComputeExecution(meta, field);
      if (!runtimeCompute || runtimeCompute.store !== false) continue;

      let newVal = withComputeRunAsExecution(meta, field, runtimeCompute.runAs, 'expr', () => runtimeCompute.execute(wrapped), 'read');
      newVal = ensureSyncBridgeResult(newVal, `@Compute(${field})`);

      const currentFieldValue = entity[field];
      if (newVal === undefined && currentFieldValue !== undefined) {
        newVal = currentFieldValue;
      }

      if (fieldMeta?.type === 'decimal') {
        newVal = quantizeByMeta(fieldMeta, entity, newVal);
      }
      entity[field] = newVal;
    }
  }

  /**
   * Recomputes affected compute fields for the provided entity.
   */
  static async recompute(meta: ModelMetadata, entity: UnknownRecord, baseChanged: Set<string>, mode: ComputeMode): Promise<void> {
    const g = meta.computeGraph;
    if (!g || !g.computeFields.size) return;
    if (!baseChanged.size) return;

    // Normalize Decimal inputs during preview so compute logic sees Decimal instead of string or number.
    if (mode === 'preview') {
      try {
        normalizeDecimalFields(meta, entity);
      } catch {}
    }

    // 1) Collect the compute subset that must be recomputed using BFS over fastReverseDeps.
    const toRecompute = this.collectTriggers(g, baseChanged);
    if (mode === 'persist') {
      const persisted = g.persistedComputeFields;
      if (persisted && persisted.size > 0) {
        for (const field of [...toRecompute]) {
          if (!persisted.has(field)) {
            toRecompute.delete(field);
          }
        }
      }
    }
    if (!toRecompute.size) return;

    // 1.1) In persist mode, perform lightweight multi-hop ManyToOne prefetching via computePathDeps.
    if (mode === 'persist') {
      try {
        await this.prefetchM2oMultiHop(meta, entity, toRecompute);
      } catch (e) {
        if (typeof console !== 'undefined') {
          console.warn('[ComputeEngine] persist multi-hop M2O prefetch failed:', e);
        }
      }
    }

    // 2) Build the topological subset.
    const orderedSubset = this.sortSubsetTopologically(g, toRecompute);

    // 3) Execute the compute expressions.
    const wrapped = createEntityBackedModelInstance(meta, entity);

    for (const f of orderedSubset) {
      const runtimeCompute = resolveRuntimeComputeExecution(meta, f);
      if (!runtimeCompute) continue;

      const fieldMeta = meta.fields.get(f);
      try {
        const oldVal = entity[f];
        const result = await withComputeRunAsExecution(meta, f, runtimeCompute.runAs, 'expr', () => runtimeCompute.execute(wrapped), mode);

        let newVal = entity[f];
        if (newVal === oldVal && result !== undefined) {
          newVal = result;
        }

        // Quantize decimal results using the effective scale, either a fixed scale or the current scaleField value.
        if (fieldMeta?.type === 'decimal') {
          newVal = quantizeByMeta(fieldMeta, entity, newVal);
        }

        const isDecimalField = fieldMeta?.type === 'decimal';
        const changed = isDecimalField ? !(newVal == null && oldVal == null) && !decimalEqual(newVal, oldVal) && newVal !== oldVal : newVal !== oldVal;

        if (changed) {
          entity[f] = newVal;
          if (mode === 'persist') baseChanged.add(f);
        }
      } catch (e) {
        const modelName = meta.fullModelName || meta.modelName || meta.className || 'Unknown';
        const errRecord = asObjectRecord(e);
        throw new Error(`Compute execution failed: ${modelName}.${f} -> ${typeof errRecord?.message === 'string' ? errRecord.message : String(e)}`);
      }
    }
  }

  // Load the path leaf fields required by computePathDeps into the entity, including multi-hop ManyToOne chains.
  private static async prefetchM2oMultiHop(meta: ModelMetadata, entity: UnknownRecord, toRecompute: Set<string>) {
    const g = meta.computeGraph;
    if (!g?.computePathDeps?.size) return;

    // root -> chains[]
    const chainsByRoot = new Map<string, string[][]>();
    for (const cf of toRecompute) {
      const paths = g.computePathDeps.get(cf) || [];
      for (const p of paths) {
        // Only handle paths whose root is a ManyToOne field on the current model.
        if (!p.root) continue;
        const fm = meta.fields.get(p.root);
        if (fm?.type !== 'ManyToOne') continue;
        if (!p.chain || !p.chain.length) continue; // No chain means no prefetch is needed.
        if (!chainsByRoot.has(p.root)) chainsByRoot.set(p.root, []);
        chainsByRoot.get(p.root)!.push(p.chain.slice());
      }
    }
    if (!chainsByRoot.size) return;

    // Simple object cache keyed by ctorName#id.
    const objCache = new Map<string, UnknownRecord>();

    // Prefetch one root at a time.
    for (const [root, chains] of chainsByRoot.entries()) {
      const rootMeta = meta.fields.get(root);
      const rel = asObjectRecord(rootMeta?.relation);
      const targetModel = rel?.targetModel;
      const targetCtor = typeof targetModel === 'function' ? (targetModel() as RuntimeComputeModelCtor) : undefined;
      if (!targetCtor) continue;

      // Resolve the root object Id. entity[root] may already be an Id or an object containing Id.
      const holder = entity[root];
      const rootId = typeof holder === 'string' || typeof holder === 'number' ? String(holder) : isObject(holder) && holder.Id ? String(holder.Id) : undefined;
      if (!rootId) continue;

      // Prefetch the first segment and normalize root into an object.
      const firstSegs = Array.from(new Set(chains.map(c => c[0]).filter(Boolean)));
      if (firstSegs.length) {
        const cacheKey = `${targetCtor.name}#${rootId}`;
        let rootObj = objCache.get(cacheKey);
        if (!rootObj) {
          const repo = getRuntimeRepository(targetCtor);
          const rows = await repo.search(['Id', '=', rootId], { fields: ['Id', ...firstSegs] });
          rootObj = rows && rows.length ? rows[0] : undefined;
          if (rootObj) objCache.set(cacheKey, rootObj);
        }
        if (rootObj) {
          entity[root] = rootObj;
        } else {
          // If the root object cannot be loaded, skip drilling down for this root.
          continue;
        }
      } else {
        // Even without a first segment, normalize to { Id } to keep chained access consistent.
        if (!isObject(entity[root])) entity[root] = { Id: rootId };
      }

      // Walk the multi-hop chain.
      for (const chain of chains) {
        if (isObject(entity[root])) {
          await this.walkAndLoad(targetCtor, entity[root], chain, objCache);
        }
      }
    }
  }

  // Walk the chain segment by segment. When a ManyToOne is encountered, load the child object using the next field as selection and write it back.
  private static async walkAndLoad(currentCtor: RuntimeComputeModelCtor, currentNode: UnknownRecord, chain: string[], objCache: Map<string, UnknownRecord>) {
    let curCtor = currentCtor;
    let curNode = currentNode;

    for (let i = 0; i < chain.length; i++) {
      const seg = chain[i];
      const curMeta = MetadataStorage.instance.getModelMetadata(curCtor);
      const fm = curMeta.fields.get(seg);
      if (!fm) break;

      const isLast = i === chain.length - 1;

      if (fm.type === 'ManyToOne') {
        const targetModel = fm.relation?.targetModel;
        const nextCtor = typeof targetModel === 'function' ? (targetModel() as RuntimeComputeModelCtor) : undefined;
        if (!nextCtor) break;

        const holder = curNode?.[seg];
        const nextId =
          typeof holder === 'string' || typeof holder === 'number' ? String(holder) : isObject(holder) && holder.Id ? String(holder.Id) : undefined;
        if (!nextId) {
          // The current node has no Id for the next object, so traversal stops here.
          break;
        }

        // Select the field needed for the next hop or the final leaf.
        const nextField = isLast ? 'Id' : chain[i + 1];
        const cacheKey = `${nextCtor.name}#${nextId}`;

        let nextObj = objCache.get(cacheKey);
        if (!nextObj) {
          const repo = getRuntimeRepository(nextCtor);
          const select = Array.from(new Set<string>(['Id', nextField].filter(Boolean)));
          const rows = await repo.search(['Id', '=', nextId], { fields: select });
          nextObj = rows && rows.length ? rows[0] : undefined;
          if (nextObj) objCache.set(cacheKey, nextObj);
        }

        if (!nextObj) break;

        // Normalize the ManyToOne field into an object.
        curNode[seg] = nextObj;
        curNode = nextObj;
        curCtor = nextCtor;
      } else {
        // Scalar or terminal segment: the previous hop already loaded the required field.
        if (isLast) {
          // nothing to do
        }
        break;
      }
    }
  }

  // Collect triggered compute fields with BFS over fastReverseDeps.
  private static collectTriggers(g: ModelComputeGraph, baseChanged: Set<string>): Set<string> {
    const queue: string[] = [];
    const visited = new Set<string>();
    const enqueue = (src: string) => {
      const arr = g.fastReverseDeps.get(src);
      if (!arr) return;
      for (const t of arr) {
        if (!visited.has(t)) {
          visited.add(t);
          queue.push(t);
        }
      }
    };
    baseChanged.forEach(enqueue);
    for (let i = 0; i < queue.length; i++) enqueue(queue[i]);
    return visited;
  }

  // Sort a subset according to topological order.
  private static sortSubsetTopologically(g: ModelComputeGraph, subset: Set<string>): string[] {
    const arr = [...subset];
    arr.sort((a, b) => g.orderIndex.get(a)! - g.orderIndex.get(b)!);
    return arr;
  }
}
