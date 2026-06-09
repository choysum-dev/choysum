// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../../orm/metadata/model';
import { MetadataStorage } from '../../../orm/metadata/storage';
import type { BaseQueryCondition, FieldSelection } from '../../../api';
import type { PathPrefetchPlan, PrefetchBatchStat, PrefetchExecStats } from '../types';
import { ENABLE_MULTI_HOP_PREVIEW, MAX_MULTI_HOP_DEPTH, PREFETCH_BATCH_SIZE } from '../constants';
import { getRuntimeRepository } from '../../runtime_repository_facade';
import { chunk, isObject, type ModelCtor, uniq } from './shared';
import type { UnknownRecord } from '../../../../utils/types';

type StatEntry = Omit<PrefetchBatchStat, 'batchCount' | 'rowCount'> & { batchCount?: number; rowCount?: number };

/**
 * Shared callback surface used by path-plan prefetch execution helpers.
 */
export type PathPlanPreloadContext = {
  plan: PathPrefetchPlan;
  recordStat: (stats: PrefetchExecStats, entry: StatEntry) => void;
  searchWithCache: (ctor: ModelCtor, ids: string | string[], fields: string[]) => Promise<UnknownRecord[]>;
  getNestedAt: (draft: UnknownRecord, root: string, prefix: string[]) => unknown;
  upsertNestedAt: (draft: UnknownRecord, root: string, prefix: string[], id?: string) => UnknownRecord;
};

/**
 * Prefetches first-hop ManyToOne relations required by a path plan.
 */
export async function executeFirstHopPrefetch(context: PathPlanPreloadContext, meta: ModelMetadata, draft: UnknownRecord, stats: PrefetchExecStats) {
  for (const [root, firstLevelFields] of context.plan.rootManyToOne.entries()) {
    if (!firstLevelFields.size) continue;

    const fieldMeta = meta.fields.get(root);
    if (!fieldMeta || fieldMeta.type !== 'ManyToOne') continue;

    const cur = draft[root];
    let fkId: string | undefined;
    if (isObject(cur) && 'Id' in cur) fkId = String(cur.Id);
    else if (typeof cur === 'string' || typeof cur === 'number') fkId = String(cur);

    if (!fkId) continue;

    const rel = fieldMeta.relation;
    const targetCtor: ModelCtor | undefined = rel?.targetModel?.();
    if (!targetCtor) continue;

    const fields = ['Id', ...firstLevelFields];
    try {
      const rows = await context.searchWithCache(targetCtor, fkId, fields);
      if (!rows.length) {
        context.recordStat(stats, {
          phase: 'm2o',
          level: 1,
          model: targetCtor.name || 'UnknownModel',
          fields,
          rowCount: 0,
          idsSample: [fkId],
        });
        continue;
      }
      const row = rows[0];
      /**
       * Prefetches multi-hop ManyToOne relations required by a path plan.
       */
      const filled: UnknownRecord = { Id: row.Id };
      firstLevelFields.forEach(k => {
        if (k in row) filled[k] = row[k];
      });

      const existing = isObject(cur) ? cur : undefined;
      draft[root] = existing ? { ...filled, ...existing } : filled;

      context.recordStat(stats, {
        phase: 'm2o',
        level: 1,
        model: targetCtor.name || 'UnknownModel',
        fields,
        rowCount: rows.length,
        idsSample: [fkId],
      });
    } catch {}
  }
}

export async function executeMultiHopM2OPrefetch(context: PathPlanPreloadContext, meta: ModelMetadata, draft: UnknownRecord, stats: PrefetchExecStats) {
  if (!ENABLE_MULTI_HOP_PREVIEW || MAX_MULTI_HOP_DEPTH < 2) return;

  const level1CtorByRoot = new Map<string, { ctor: ModelCtor; meta: ModelMetadata }>();
  for (const [root] of context.plan.m2oChains.entries()) {
    const rootFieldMeta = meta.fields.get(root);
    if (!rootFieldMeta || rootFieldMeta.type !== 'ManyToOne') continue;
    const ctor1: ModelCtor | undefined = rootFieldMeta.relation?.targetModel?.();
    if (!ctor1) continue;
    level1CtorByRoot.set(root, { ctor: ctor1, meta: MetadataStorage.instance.getModelMetadata(ctor1) });
  }

  for (let level = 2; level <= MAX_MULTI_HOP_DEPTH; level++) {
    type Bucket = { ctor: ModelCtor; fields: Set<string>; idToNodes: Map<string, UnknownRecord[]> };
    const buckets = new Map<string, Bucket>();

    for (const [root, chains] of context.plan.m2oChains.entries()) {
      if (!chains.length) continue;
      const level1 = level1CtorByRoot.get(root);
      if (!level1) continue;

      for (const chain of chains) {
        if (chain.length < level) continue;
        const prefix = chain.slice(0, Math.max(0, level - 2));
        const prevField = chain[level - 2];
        const currField = chain[level - 1];

        let currentMeta: ModelMetadata | null = level1.meta;
        for (let i = 0; i < prefix.length; i++) {
          const seg = prefix[i];
          const f = currentMeta.fields.get(seg);
          if (!f || f.type !== 'ManyToOne') {
            currentMeta = null;
            break;
          }
          const nextCtor: ModelCtor | undefined = f.relation?.targetModel?.();
          if (!nextCtor) {
            currentMeta = null;
            break;
          }
          currentMeta = MetadataStorage.instance.getModelMetadata(nextCtor);
        }
        if (!currentMeta) continue;

        const prevMetaField = currentMeta.fields.get(prevField);
        if (!prevMetaField || prevMetaField.type !== 'ManyToOne') continue;
        const targetCtor: ModelCtor | undefined = prevMetaField.relation?.targetModel?.();
        if (!targetCtor) continue;
        const bucketKey = targetCtor.name || `Ctor@L${level}`;

        const prevNode = context.getNestedAt(draft, root, prefix);
        if (!isObject(prevNode)) continue;
        const val = prevNode[prevField];
        let id: string | undefined;
        if (isObject(val) && 'Id' in val) id = String(val.Id);
        else if (typeof val === 'string' || typeof val === 'number') id = String(val);
        if (!id) continue;

        const childNode = context.upsertNestedAt(draft, root, [...prefix, prevField], id);

        let bucket = buckets.get(bucketKey);
        if (!bucket) {
          bucket = { ctor: targetCtor, fields: new Set<string>(), idToNodes: new Map<string, UnknownRecord[]>() };
          buckets.set(bucketKey, bucket);
        }
        bucket.fields.add(currField);
        const list = bucket.idToNodes.get(id) || [];
        list.push(childNode);
        bucket.idToNodes.set(id, list);
      }
    }

    for (const [, bucket] of buckets.entries()) {
      const ids = [...bucket.idToNodes.keys()];
      if (!ids.length) continue;
      const fields = ['Id', ...bucket.fields];
      for (const batch of chunk(ids, PREFETCH_BATCH_SIZE)) {
        try {
          const rows = await context.searchWithCache(bucket.ctor, batch, fields);

          const map = new Map<string, UnknownRecord>();
          for (const r of rows) map.set(String(r.Id), r);

          context.recordStat(stats, {
            phase: 'm2o',
            level,
            model: bucket.ctor.name || 'UnknownModel',
            fields,
            batchCount: 1,
            rowCount: rows.length,
            idsSample: batch.slice(0, 3),
          });

          for (const id of batch) {
            const row = map.get(String(id));
            const nodes = bucket.idToNodes.get(String(id)) || [];
            if (!row || !nodes.length) continue;

            for (const node of nodes) {
              const filled: UnknownRecord = { Id: row.Id };
              bucket.fields.forEach(k => {
                if (k in row) filled[k] = row[k];
              });
              Object.assign(node, { ...filled, ...node });
            }
          }
        } catch {}
      }
    }
  }
}

/**
 * Prefetches collection roots required by a path plan.
 */
export async function executeCollectionsPrefetch(context: PathPlanPreloadContext, meta: ModelMetadata, draft: UnknownRecord, stats: PrefetchExecStats) {
  if (!context.plan.collections.size) return;

  for (const [coll, spec] of context.plan.collections.entries()) {
    const fieldMeta = meta.fields.get(coll);
    if (!fieldMeta || (fieldMeta.type !== 'OneToMany' && fieldMeta.type !== 'ManyToMany')) continue;

    if (Array.isArray(draft[coll])) continue;

    const parentId = draft.Id;
    if (!parentId) continue;

    const rel = fieldMeta.relation;
    const childCtor: ModelCtor | undefined = rel?.targetModel?.();
    if (!childCtor) continue;

    const inverseField = rel?.inverseField as string | undefined;
    if (!inverseField) continue;

    const firsts = new Set<string>();
    for (const ch of spec.chains) if (ch.length >= 1) firsts.add(ch[0]);

    const fields = ['Id', ...firsts];
    try {
      const repo = getRuntimeRepository(childCtor);
      const condition: BaseQueryCondition = [inverseField, '=', String(parentId)];
      const rows = await repo.search(condition, { fields: fields as FieldSelection<UnknownRecord> });

      context.recordStat(stats, {
        phase: 'collection',
        level: 1,
        model: childCtor.name || 'UnknownModel',
        fields,
        rowCount: rows.length,
        idsSample: [String(parentId)],
      });

      const arr: UnknownRecord[] = [];
      for (const row of rows) {
        const item: UnknownRecord = { Id: row.Id };
        firsts.forEach(k => {
          if (k in row) item[k] = row[k];
        });
        arr.push(item);
      }
      draft[coll] = arr;

      const childMeta = MetadataStorage.instance.getModelMetadata(childCtor);
      const m2oFirsts = [...firsts].filter(k => childMeta.fields.get(k)?.type === 'ManyToOne');

      for (const a of m2oFirsts) {
        const secondSet = new Set<string>();
        for (const ch of spec.chains) {
          if (ch[0] === a && ch.length >= 2) secondSet.add(ch[1]);
        }
        if (!secondSet.size) continue;

        const nextCtor: ModelCtor | undefined = childMeta.fields.get(a)!.relation?.targetModel?.();
        if (!nextCtor) continue;

        const idList = arr
          .map(it => it[a])
          .filter(v => !!v)
          .map(v => (isObject(v) ? String(v.Id) : String(v)));
        const ids = uniq(idList);
        if (!ids.length) continue;

        const fields2 = ['Id', ...secondSet];
        for (const batch of chunk(ids, PREFETCH_BATCH_SIZE)) {
          try {
            const rows2 = await context.searchWithCache(nextCtor, batch, fields2);

            context.recordStat(stats, {
              phase: 'collection',
              level: 2,
              model: nextCtor.name || 'UnknownModel',
              fields: fields2,
              batchCount: 1,
              rowCount: rows2.length,
              idsSample: batch.slice(0, 3),
            });

            const map2 = new Map<string, UnknownRecord>();
            for (const r2 of rows2) map2.set(String(r2.Id), r2);

            for (const it of arr) {
              const v = it[a];
              const aId = isObject(v) ? String(v.Id) : String(v);
              if (!aId) continue;
              const r2 = map2.get(aId);
              if (!r2) continue;
              const obj = isObject(v) ? v : { Id: aId };
              secondSet.forEach(b => {
                if (b in r2) obj[b] = r2[b];
              });
              it[a] = obj;
            }
          } catch {}
        }
      }
    } catch {}
  }
}
