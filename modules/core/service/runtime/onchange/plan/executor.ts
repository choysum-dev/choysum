// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../../orm/metadata/model';
import { MetadataStorage } from '../../../orm/metadata/storage';
import type { BaseQueryCondition, FieldSelection } from '../../../api';
import type { PathPrefetchPlan, PrefetchBatchStat, PrefetchExecStats } from '../types';
import { DIAG_BATCH_STATS_ENABLED, ENABLE_MULTI_HOP_PREVIEW, LRU_CACHE_SIZE, MAX_MULTI_HOP_DEPTH, REQUEST_CACHE_ENABLED } from '../constants';
import { OnchangeCacheManager } from '../cache';
import { getRuntimeRepository } from '../../runtime_repository_facade';
import { executeCollectionsPrefetch, executeFirstHopPrefetch, executeMultiHopM2OPrefetch, type PathPlanPreloadContext } from './preload';
import { isObject, type ModelCtor } from './shared';
import type { UnknownRecord } from '../../../../utils/types';

export class PathPlanExecutor {
  private readonly plan: PathPrefetchPlan;
  private requestCache?: Map<string, Map<string, UnknownRecord>>;

  constructor(plan: PathPrefetchPlan) {
    this.plan = plan;
  }

  async execute(meta: ModelMetadata, draft: UnknownRecord): Promise<PrefetchExecStats> {
    this.ensureCachesInit();

    const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
    const context = this.createPreloadContext();

    await executeFirstHopPrefetch(context, meta, draft, stats);

    if (ENABLE_MULTI_HOP_PREVIEW && MAX_MULTI_HOP_DEPTH >= 2) {
      await executeMultiHopM2OPrefetch(context, meta, draft, stats);
    }

    await executeCollectionsPrefetch(context, meta, draft, stats);

    return stats;
  }

  private createPreloadContext(): PathPlanPreloadContext {
    return {
      plan: this.plan,
      recordStat: (stats, entry) => this.recordStat(stats, entry),
      searchWithCache: (ctor, ids, fields) => this.searchWithCache(ctor, ids, fields),
      getNestedAt: (draft, root, prefix) => this.getNestedAt(draft, root, prefix),
      upsertNestedAt: (draft, root, prefix, id) => this.upsertNestedAt(draft, root, prefix, id),
    };
  }

  private recordStat(stats: PrefetchExecStats, entry: Omit<PrefetchBatchStat, 'batchCount' | 'rowCount'> & { batchCount?: number; rowCount?: number }) {
    const batchCount = entry.batchCount ?? 1;
    const rowCount = entry.rowCount ?? 0;
    stats.totalBatches += batchCount;
    stats.totalRows += rowCount;
    if (DIAG_BATCH_STATS_ENABLED) {
      stats.batches.push({
        phase: entry.phase,
        level: entry.level,
        model: entry.model,
        fields: entry.fields.slice(),
        batchCount,
        rowCount,
        idsSample: entry.idsSample?.slice(),
      });
    }
  }

  private getRecordId(value: unknown): string | undefined {
    if (isObject(value) && (typeof value.Id === 'string' || typeof value.Id === 'number')) {
      return String(value.Id);
    }
    return typeof value === 'string' || typeof value === 'number' ? String(value) : undefined;
  }

  private normalizeNestedNode(value: unknown): UnknownRecord {
    const id = this.getRecordId(value);
    return id ? { Id: id } : {};
  }

  private getNestedAt(draft: UnknownRecord, root: string, prefix: string[]): UnknownRecord {
    const node = draft[root];
    const rootNode: UnknownRecord = isObject(node) ? node : this.normalizeNestedNode(node);
    if (!isObject(node)) {
      draft[root] = rootNode;
    }

    let current: UnknownRecord = rootNode;
    for (const seg of prefix) {
      const v = current[seg];
      if (isObject(v)) {
        current = v;
        continue;
      }
      const obj = this.normalizeNestedNode(v);
      current[seg] = obj;
      current = obj;
    }
    return current;
  }

  private upsertNestedAt(draft: UnknownRecord, root: string, prefix: string[], id?: string): UnknownRecord {
    const node = this.getNestedAt(draft, root, prefix);
    if (id && this.getRecordId(node.Id) == null) {
      node.Id = id;
    }
    return node;
  }

  private ensureCachesInit() {
    if (REQUEST_CACHE_ENABLED && !this.requestCache) {
      this.requestCache = new Map<string, Map<string, UnknownRecord>>();
    }
  }

  private getModelKey(ctor: ModelCtor): string {
    const m = MetadataStorage.instance.getModelMetadata(ctor);
    return (m.fullModelName || m.modelName || ctor.name || 'UnknownModel') as string;
  }

  private buildCacheKey(ctor: ModelCtor, fields: string[]): { cacheKey: string; fieldsNormalized: string[] } {
    const set = new Set<string>(['Id', ...fields]);
    const fieldsNormalized = Array.from(set).sort();
    const fieldsSig = fieldsNormalized.join(',');
    const cacheKey = `${this.getModelKey(ctor)}#${fieldsSig}`;
    return { cacheKey, fieldsNormalized };
  }

  private async searchWithCache(ctor: ModelCtor, ids: string | string[], fields: string[]): Promise<UnknownRecord[]> {
    const idsArr = Array.isArray(ids) ? ids.map(String) : [String(ids)];
    if (!idsArr.length) return [];

    const { cacheKey, fieldsNormalized } = this.buildCacheKey(ctor, fields);

    const hits: UnknownRecord[] = [];
    const remaining = new Set<string>(idsArr);

    let reqBucket: Map<string, UnknownRecord> | undefined;
    if (REQUEST_CACHE_ENABLED) {
      reqBucket = this.requestCache?.get(cacheKey);
      if (reqBucket) {
        for (const id of idsArr) {
          const row = reqBucket.get(id);
          if (row) {
            hits.push(row);
            remaining.delete(id);
          }
        }
      }
    }

    let lruBucket: Map<string, UnknownRecord> | undefined;
    if (remaining.size && LRU_CACHE_SIZE > 0) {
      lruBucket = OnchangeCacheManager.get(cacheKey) as Map<string, UnknownRecord> | undefined;
      if (lruBucket) {
        for (const id of Array.from(remaining)) {
          const row = lruBucket.get(id);
          if (row) {
            hits.push(row);
            remaining.delete(id);
          }
        }
      }
    }

    let missRows: UnknownRecord[] = [];
    if (remaining.size) {
      const repo = getRuntimeRepository(ctor);
      const missIds = Array.from(remaining);
      const condition: BaseQueryCondition = missIds.length === 1 ? ['Id', '=', missIds[0]] : ['Id', 'in', missIds];
      const rows = (await repo.search(condition, { fields: fieldsNormalized as FieldSelection<UnknownRecord> })) || [];
      missRows = rows.filter(isObject);
    }

    const mergedMap = new Map<string, UnknownRecord>();
    for (const r of hits) {
      const rowId = this.getRecordId(r.Id);
      if (rowId) mergedMap.set(rowId, r);
    }
    for (const r of missRows) {
      const rowId = this.getRecordId(r.Id);
      if (rowId) mergedMap.set(rowId, r);
    }
    const merged = Array.from(mergedMap.values());

    if (merged.length) {
      if (REQUEST_CACHE_ENABLED) {
        if (!reqBucket) {
          reqBucket = new Map<string, UnknownRecord>();
          this.requestCache?.set(cacheKey, reqBucket);
        }
        for (const r of merged) {
          const rowId = this.getRecordId(r.Id);
          if (rowId) reqBucket.set(rowId, r);
        }
      }
      if (LRU_CACHE_SIZE > 0) {
        const bucket = (OnchangeCacheManager.get(cacheKey) as Map<string, UnknownRecord> | undefined) || new Map<string, UnknownRecord>();
        for (const r of merged) {
          const rowId = this.getRecordId(r.Id);
          if (rowId) bucket.set(rowId, r);
        }
        OnchangeCacheManager.set(cacheKey, bucket);
      }
    }

    return merged;
  }
}
