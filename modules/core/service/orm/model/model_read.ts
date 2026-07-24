// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import {
  QueryCondition,
  SearchOptions,
  FieldSelection,
  SoftDeleteOptions,
  CountOptions,
  ReadGroupOptions,
  ReadGroupResult,
  BaseQueryCondition,
  ReadGroupCountOptions,
  TemporalGranularity,
  GroupBySpec,
  GroupRow,
} from '../repository/types';
import { coerceToBucketStart, nextBucket, enumerateBuckets } from '../repository/read';
import {
  normalizeGroupBySpec,
  normalizeGroupBySpecs,
  normalizeFieldAggregation,
  rebuildGroupSpec,
  rebuildCompositeGroupSpec,
  rebuildAggFields,
} from '../repository/read';
import { andAll, toRepoCondition } from '../repository/query';
import type { NormalizedGroupSpec, NormalizedCompositeGroupSpec, NormalizedAgg } from '../repository/read';
import moment from 'moment';
import { MetadataStorage } from '../metadata/storage';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import { getModelRepository } from './model_internal_facade';
import { resolveRepositoryWithSoftDeleteOptions } from './model_soft_delete_scope';
import { ComputeEngine } from '../../runtime/compute/engine';
import { getModelRuntimeMetadata } from './model_runtime_service_facade';
import type { ObjectRecord } from '../../../utils/types';
import type { RuntimeModelCtor } from './types';
import { createServiceByModel } from '../../rpc';
import { _t } from '@/core/service/i18n_binder';

/**
 * Read-related delegated operations.
 * - Only perform data access and lightweight validation, returning raw entity records as plain objects.
 * - Do not perform isTopLevelGrpcRequest checks and do not create proxy instances.
 * - The upper model.ts layer converts results into plain values or proxy instances.
 */
export class ReadOperations {
  private static resolveRepository<T extends BaseModel>(
    ModelCtor: RuntimeModelCtor<T>,
    options?: SoftDeleteOptions
  ): {
    search: (condition: unknown, options?: SearchOptions<T>) => Promise<ObjectRecord[]>;
    count: (condition: unknown) => Promise<number>;
    readTotals: (options: { fields: unknown[]; condition: unknown; timezone?: string }) => Promise<ObjectRecord>;
    readGroup: (options: {
      groupby: GroupBySpec<T> | GroupBySpec<T>[];
      fields: unknown[];
      condition: unknown;
      having?: BaseQueryCondition;
      orderBy?: unknown;
      limit?: number;
      offset?: number;
      timezone?: string;
    }) => Promise<ObjectRecord[]>;
    readGroupCount: (options: {
      groupby: GroupBySpec<T> | GroupBySpec<T>[];
      fields: unknown[];
      condition: unknown;
      having?: BaseQueryCondition;
      timezone?: string;
    }) => Promise<number>;
  } {
    return resolveRepositoryWithSoftDeleteOptions(ModelCtor, options) as {
      search: (condition: unknown, options?: SearchOptions<T>) => Promise<ObjectRecord[]>;
      count: (condition: unknown) => Promise<number>;
      readTotals: (options: { fields: unknown[]; condition: unknown; timezone?: string }) => Promise<ObjectRecord>;
      readGroup: (options: {
        groupby: GroupBySpec<T> | GroupBySpec<T>[];
        fields: unknown[];
        condition: unknown;
        having?: BaseQueryCondition;
        orderBy?: unknown;
        limit?: number;
        offset?: number;
        timezone?: string;
      }) => Promise<ObjectRecord[]>;
      readGroupCount: (options: {
        groupby: GroupBySpec<T> | GroupBySpec<T>[];
        fields: unknown[];
        condition: unknown;
        having?: BaseQueryCondition;
        timezone?: string;
      }) => Promise<number>;
    };
  }

  private static normalizeRequestedReadFields(fields?: FieldSelection<BaseModel>): Set<string> | undefined {
    if (!Array.isArray(fields)) return undefined;
    return new Set(fields.map(field => String(field || '').trim()).filter(Boolean));
  }

  private static isAttachmentFieldType(type: unknown): boolean {
    const normalized = String(type || '')
      .trim()
      .toLowerCase();
    return normalized === 'binary' || normalized === 'image';
  }

  private static isStorageBlobCarrierModel(meta: { fullModelName?: string; application?: string; modelName?: string; name?: string }): boolean {
    const full = String(meta.fullModelName || '').trim();
    if (
      full === 'document.AttachmentObject' ||
      full === 'document.UploadSession' ||
      full === 'document.AttachmentContent' ||
      full === 'document.AttachmentUploadSession' ||
      full === 'document.StoredContent'
    ) {
      return true;
    }

    const app = String(meta.application || '')
      .trim()
      .toLowerCase();
    const model = String(meta.modelName || meta.name || '')
      .trim()
      .toLowerCase();
    return (
      (app === 'storage' && (model === 'attachmentobject' || model === 'uploadsession')) ||
      (app === 'document' && (model === 'attachmentcontent' || model === 'attachmentuploadsession' || model === 'storedcontent'))
    );
  }

  private static resolveOwnerModelLabel<T extends BaseModel>(ModelCtor: RuntimeModelCtor<T>): string | undefined {
    const meta = MetadataStorage.instance.getModelMetadata(ModelCtor);
    const full = String(meta?.fullModelName || '').trim();
    if (full) return full;

    const app = String((meta as { application?: string })?.application || '').trim();
    const model = String(meta?.modelName || meta?.name || '').trim();
    if (app && model) return `${app}.${model}`;
    return model || undefined;
  }

  private static resolveRequestedRootFields(fields?: FieldSelection<BaseModel>): { wildcard: boolean; names: Set<string> } {
    if (!Array.isArray(fields)) {
      return { wildcard: true, names: new Set<string>() };
    }

    const names = new Set<string>();
    let wildcard = false;

    for (const field of fields) {
      if (typeof field === 'string') {
        const normalized = String(field || '').trim();
        if (!normalized) continue;
        if (normalized === '*') {
          wildcard = true;
          continue;
        }
        const root = normalized.split('.')[0]?.trim();
        if (root) names.add(root);
        continue;
      }

      if (field && typeof field === 'object' && !Array.isArray(field)) {
        const keys = Object.keys(field as Record<string, unknown>);
        for (const key of keys) {
          const normalized = String(key || '').trim();
          if (normalized) names.add(normalized);
        }
      }
    }

    return { wildcard, names };
  }

  private static resolveAttachmentReadFieldNames<T extends BaseModel>(ModelCtor: RuntimeModelCtor<T>, fields?: FieldSelection<BaseModel>): string[] {
    const runtimeMeta = getModelRuntimeMetadata(ModelCtor);
    if (ReadOperations.isStorageBlobCarrierModel(runtimeMeta as any)) {
      return [];
    }

    const allAttachmentFields: string[] = [];
    for (const [fieldName, fieldMeta] of runtimeMeta.fields.entries()) {
      if (ReadOperations.isAttachmentFieldType((fieldMeta as { type?: string })?.type)) {
        allAttachmentFields.push(fieldName);
      }
    }
    if (!allAttachmentFields.length) return [];

    const requested = ReadOperations.resolveRequestedRootFields(fields);
    if (requested.wildcard) {
      return allAttachmentFields;
    }

    return allAttachmentFields.filter(fieldName => requested.names.has(fieldName));
  }

  private static resolveAttachmentBindingService(): { Search: (condition: unknown, options?: unknown) => Promise<ObjectRecord[]> } | undefined {
    try {
      const service = createServiceByModel('document.AttachmentBinding') as unknown as {
        Search?: (condition: unknown, options?: unknown) => Promise<ObjectRecord[]>;
      };
      if (!service || typeof service.Search !== 'function') {
        return undefined;
      }
      return {
        Search: service.Search.bind(service),
      };
    } catch {
      return undefined;
    }
  }

  private static async injectAttachmentBindingsForRead<T extends BaseModel>(
    ModelCtor: RuntimeModelCtor<T>,
    rows: ObjectRecord[],
    fields?: FieldSelection<BaseModel>
  ): Promise<void> {
    if (!rows.length) return;

    const attachmentFields = ReadOperations.resolveAttachmentReadFieldNames(ModelCtor, fields);
    if (!attachmentFields.length) return;

    const ownerModel = ReadOperations.resolveOwnerModelLabel(ModelCtor);
    if (!ownerModel) return;

    const ownerRecordIds = Array.from(new Set(rows.map(row => String((row as Record<string, unknown>)?.Id || '').trim()).filter(Boolean)));
    if (!ownerRecordIds.length) return;

    const bindingService = ReadOperations.resolveAttachmentBindingService();
    if (!bindingService) {
      return;
    }

    let bindings: Array<{ Id?: unknown; OwnerRecordId?: unknown; FieldName?: unknown }> = [];
    try {
      bindings =
        ((await bindingService.Search(
          {
            And: [
              ['OwnerModel', '=', ownerModel],
              ['OwnerRecordId', 'in', ownerRecordIds],
              ['FieldName', 'in', attachmentFields],
              ['Status', '=', 'active'],
            ],
          } as any,
          {
            fields: ['Id', 'OwnerRecordId', 'FieldName'] as any,
            limit: Math.max(1, ownerRecordIds.length * attachmentFields.length),
          } as any
        )) as Array<{ Id?: unknown; OwnerRecordId?: unknown; FieldName?: unknown }>) || [];
    } catch {
      return;
    }

    const byOwner = new Map<string, Map<string, string>>();
    for (const binding of bindings) {
      const bindingId = String(binding?.Id || '').trim();
      const ownerRecordId = String(binding?.OwnerRecordId || '').trim();
      const fieldName = String(binding?.FieldName || '').trim();
      if (!bindingId || !ownerRecordId || !fieldName) continue;

      if (!byOwner.has(ownerRecordId)) {
        byOwner.set(ownerRecordId, new Map<string, string>());
      }
      byOwner.get(ownerRecordId)!.set(fieldName, bindingId);
    }

    for (const row of rows) {
      if (!row || typeof row !== 'object') continue;
      const rowRecord = row as Record<string, unknown>;
      const ownerRecordId = String(rowRecord.Id || '').trim();
      if (!ownerRecordId) continue;
      const fieldBindings = byOwner.get(ownerRecordId);

      for (const fieldName of attachmentFields) {
        const bindingId = fieldBindings?.get(fieldName);
        rowRecord[fieldName] = bindingId || null;
      }
    }
  }

  private static injectVirtualComputeForRead<T extends BaseModel>(
    ModelCtor: RuntimeModelCtor<T>,
    rows: ObjectRecord[],
    fields?: FieldSelection<BaseModel>
  ): void {
    if (!rows.length) return;
    const meta = getModelRuntimeMetadata(ModelCtor);
    const requested = ReadOperations.normalizeRequestedReadFields(fields);
    for (const row of rows) {
      if (!row || typeof row !== 'object') continue;
      ComputeEngine.injectVirtualForRead(meta, row, requested);
    }
  }

  static async Browse<T extends BaseModel>(
    ModelCtor: RuntimeModelCtor<T>,
    id: string,
    fields?: FieldSelection<T>,
    softDeleteOptions?: SoftDeleteOptions
  ): Promise<ObjectRecord> {
    const repository = ReadOperations.resolveRepository(ModelCtor, softDeleteOptions);
    const searchOptions: SearchOptions<T> = fields ? { fields } : {};
    const results = await repository.search(['Id', '=', id], searchOptions);

    if (!results.length) {
      const meta = MetadataStorage.instance.getModelMetadata(ModelCtor);
      const domain = typeof meta?.application === 'string' && meta.application.trim() ? meta.application.trim() : 'core';
      const modelLabel = meta?.fullModelName || meta?.modelName || meta?.name || 'Record';
      throw new ChoysumError({
        domain,
        code: 'NotFound',
        message: _t('%s not found', { scope: 'service/orm/model/model_read' }, modelLabel),
      }).withGrpcCode(GrpcCode.NotFound);
    }
    const typedResults = results as ObjectRecord[];
    await ReadOperations.injectAttachmentBindingsForRead(ModelCtor, typedResults, fields as FieldSelection<BaseModel> | undefined);
    ReadOperations.injectVirtualComputeForRead(ModelCtor, typedResults, fields as FieldSelection<BaseModel> | undefined);
    return typedResults[0]!;
  }

  static async Search<T extends BaseModel>(
    ModelCtor: RuntimeModelCtor<T>,
    condition: QueryCondition<T> | [] = [],
    options?: SearchOptions<T>
  ): Promise<ObjectRecord[]> {
    const cond: QueryCondition<T> | [] = condition === undefined || condition === null ? [] : condition;
    const repository = ReadOperations.resolveRepository(ModelCtor, options);
    const rows = (await repository.search(cond as QueryCondition<T>, options)) as ObjectRecord[];
    await ReadOperations.injectAttachmentBindingsForRead(ModelCtor, rows, options?.fields as FieldSelection<BaseModel> | undefined);
    ReadOperations.injectVirtualComputeForRead(ModelCtor, rows, options?.fields as FieldSelection<BaseModel> | undefined);
    return rows;
  }

  static async Count<T extends BaseModel>(ModelCtor: RuntimeModelCtor<T>, condition: QueryCondition<T> | [] = [], options?: CountOptions): Promise<number> {
    const repository = ReadOperations.resolveRepository(ModelCtor, options);
    return await repository.count(condition as QueryCondition<T>);
  }

  /**
   * Public entry point for multi-level ReadGroup calls.
   * - Recursively calls repository readGroup, which only supports one level at a time, then combines the results into flat or tree output.
   * - Supports expand, shape, fillTemporalGaps, and includeTotals.
   */
  static async ReadGroup<T extends BaseModel>(
    ModelCtor: RuntimeModelCtor<T>,
    groupby: Array<GroupBySpec<T> | GroupBySpec<T>[]> | [],
    condition: QueryCondition<T> | [] = [],
    options: ReadGroupOptions<T> = {}
  ): Promise<ReadGroupResult> {
    const repo = ReadOperations.resolveRepository(ModelCtor, options);
    const timezone = options.timezone;
    // Use the explicitly supplied condition directly.
    const baseCondition = (condition ?? []) as BaseQueryCondition | [];

    // Handle empty groupby (Totals)
    if (!groupby || groupby.length === 0) {
      const fieldsAgg = (options.fields ?? []).map(normalizeFieldAggregation);
      const row = await repo.readTotals({
        fields: rebuildAggFields(fieldsAgg),
        condition: toRepoCondition<T>(baseCondition),
        timezone,
      });

      const metrics: ObjectRecord = {};
      for (const a of fieldsAgg) {
        metrics[a.alias] = pickAliasedValue(row as ObjectRecord, a.alias);
      }

      const result: GroupRow = {
        depth: 0,
        keys: {},
        labels: {},
        metrics,
        count: Number(pickAliasedValue(row as ObjectRecord, '__count') ?? 0),
        condition: normalizeGroupCondition(baseCondition),
        remainingGroupby: [],
        children: [],
      };
      return [result];
    }

    // New structure: each level may be a single column or a composite column set.
    // Multi-level grouping allows either shape; composite levels with length > 1 are normalized into composite specs.
    const groupLevels: Array<NormalizedGroupSpec[] | NormalizedGroupSpec> = groupby.map(level => {
      if (Array.isArray(level)) {
        if (level.length === 0) throw new Error('Composite groupby level cannot be empty');
        return level.map(s => normalizeGroupBySpec(s));
      }
      return [normalizeGroupBySpec(level)];
    });
    if (!groupLevels.length) throw new Error('ReadGroup requires non-empty groupby');

    const fieldsAgg = (options.fields ?? []).map(normalizeFieldAggregation);

    // New semantics: always expand all levels. Lazy expansion can be added later if needed.
    const expandLevels = groupLevels.length;

    const getHavingForLevel = (level: number): BaseQueryCondition | undefined => {
      if (Array.isArray(options.having)) return options.having[level];
      return level === 0 ? options.having : undefined;
    };
    const getOrderForLevel = (level: number): unknown | undefined => {
      if (Array.isArray(options.orderBy)) return options.orderBy[level];
      return level === 0 ? options.orderBy : undefined;
    };
    const getLimitForLevel = (level: number): number | undefined => {
      const lim = options.limit;
      if (typeof lim === 'number') return level === 0 ? lim : undefined;
      if (lim && typeof lim === 'object' && Array.isArray(lim.perLevel)) {
        const v = lim.perLevel[level];
        return typeof v === 'number' ? v : undefined;
      }
      return undefined;
    };
    const getOffsetForLevel = (level: number): number | undefined => (level === 0 && typeof options.offset === 'number' ? options.offset : undefined);

    // Recursively fetch nodes for the requested level.
    const fetchLevel = async (level: number, parentCondition: BaseQueryCondition | [] | undefined): Promise<TreeNodeInternal[]> => {
      const specsArr = groupLevels[level]; // array of one or more specs
      const specs: NormalizedGroupSpec[] = Array.isArray(specsArr) ? specsArr : [specsArr as NormalizedGroupSpec];
      const where = andAll(baseCondition, parentCondition);
      const groupbyParam = specs.length === 1 ? rebuildGroupSpec(specs[0]) : specs.map(s => rebuildGroupSpec(s));
      const rows = await repo.readGroup({
        groupby: groupbyParam as GroupBySpec<T> | GroupBySpec<T>[],
        fields: rebuildAggFields(fieldsAgg),
        condition: toRepoCondition<T>(where),
        having: getHavingForLevel(level),
        orderBy: getOrderForLevel(level),
        limit: getLimitForLevel(level),
        offset: getOffsetForLevel(level),
        timezone,
      });

      const nodesRaw: TreeNodeInternal[] = rows.map((r: ObjectRecord) => {
        const rowRecord = r as ObjectRecord;
        const key: ObjectRecord = {};
        const keyAliases: string[] = [];
        const metrics: ObjectRecord = {};
        for (const s of specs) {
          const val = pickAliasedValue(rowRecord, s.alias);
          key[s.alias] = val;
          keyAliases.push(s.alias);
        }
        for (const a of fieldsAgg) {
          metrics[a.alias] = pickAliasedValue(rowRecord, a.alias);
        }
        // Composite-level condition: merge each field condition with AND.
        let thisGroupCondition: BaseQueryCondition | [] | undefined = undefined;
        for (const s of specs) {
          const val = key[s.alias];
          const condSingle = buildGroupCondition(s, val, timezone);
          thisGroupCondition = thisGroupCondition ? andAll(thisGroupCondition, condSingle) : condSingle;
        }
        return {
          keyAliases,
          keyValues: keyAliases.map(a => key[a]),
          key,
          metrics,
          count: Number(pickAliasedValue(rowRecord, '__count') ?? 0),
          condition: andAll(baseCondition, andAll(parentCondition, thisGroupCondition)),
          children: [],
        };
      });

      // Fill temporal gaps only on the last expanded level, and only for a single temporal grouping field.
      const isLastExpandedLevel = level === expandLevels - 1;
      let nodes = nodesRaw;
      if (isLastExpandedLevel && options.fillTemporalGaps && specs.length === 1 && specs[0].isTime) {
        nodes = fillTemporalGapsForLevel(nodesRaw, specs[0], fieldsAgg, timezone, where);
      }

      // Continue descending into the next level.
      if (level + 1 < expandLevels) {
        for (const n of nodes) {
          // The child parentCondition must include every condition contributed by the current level.
          let layerCond: BaseQueryCondition | [] | undefined = undefined;
          for (const s of specs) {
            const val = n.key[s.alias];
            const condSingle = buildGroupCondition(s, val, timezone);
            layerCond = layerCond ? andAll(layerCond, condSingle) : condSingle;
          }
          n.children = await fetchLevel(level + 1, andAll(parentCondition, layerCond));
        }
      }
      return nodes;
    };

    const topNodes = await fetchLevel(0, undefined);

    // Build a display-name cache for ManyToOne grouping by batch-loading target-model DisplayName values.
    // alias -> whether the alias represents a temporal granularity.
    const isTemporalAlias = (alias: string) => !!pickGranularityFromAlias(alias);
    const baseFieldFromAlias = (alias: string) => String(alias || '').split('__')[0];
    const meta = MetadataStorage.instance.getModelMetadata(ModelCtor);
    const isManyToOneField = (field: string) => {
      const fm = meta.fields.get(field);
      return fm?.type === 'ManyToOne';
    };

    // Recursively collect alias -> Id sets from the currently expanded levels.
    const idsByAlias = new Map<string, Set<string>>();
    const collectIds = (nodes: TreeNodeInternal[], level: number) => {
      for (const n of nodes) {
        for (const alias of n.keyAliases) {
          const val = n.key[alias];
          if (!isTemporalAlias(alias)) {
            const base = baseFieldFromAlias(alias);
            if (isManyToOneField(base) && typeof val === 'string' && val) {
              if (!idsByAlias.has(alias)) idsByAlias.set(alias, new Set<string>());
              idsByAlias.get(alias)!.add(val);
            }
          }
        }
        if (n.children && level + 1 < expandLevels) {
          collectIds(n.children, level + 1);
        }
      }
    };
    collectIds(topNodes, 0);

    // Batch query cache: alias -> Map<id, displayName>.
    const relLabelCache = new Map<string, Map<string, string>>();
    for (const [alias, idSet] of idsByAlias) {
      const base = baseFieldFromAlias(alias);
      const fm = meta.fields.get(base);
      const targetCtor = fm?.relation?.targetModel?.();
      if (!targetCtor) continue;
      const repo = getModelRepository(targetCtor);
      const ids = Array.from(idSet);
      if (!ids.length) continue;
      const rows = (await repo.search(['Id', 'in', ids], {
        fields: ['Id', 'DisplayName'] as FieldSelection<BaseModel>,
      })) as Array<{ Id: string; DisplayName: string }>;
      const map = new Map<string, string>();
      for (const r of rows || []) map.set(String(r.Id), String(r.DisplayName ?? r.Id));
      // Fallback: preserve the raw Id when no display name is found.
      for (const id of ids) if (!map.has(id)) map.set(id, id);
      relLabelCache.set(alias, map);
    }

    // Normalize everything into a GroupRow tree. The root array is the top level.

    const buildGroupRows = (nodes: TreeNodeInternal[], depth: number): GroupRow[] => {
      return nodes.map(n => {
        const keys: ObjectRecord = {};
        const labels: Record<string, string> = {};
        for (const a of n.keyAliases) {
          const val = n.key[a];
          keys[a] = val;
          if (isTemporalAlias(a)) {
            labels[a] = formatGroupDisplay(a, val);
          } else {
            const base = baseFieldFromAlias(a);
            if (isManyToOneField(base)) {
              const m = relLabelCache.get(a);
              const keyStr = String(val ?? '');
              labels[a] = m?.get(keyStr) ?? keyStr;
            } else {
              labels[a] = String(val ?? '');
            }
          }
        }
        const row: GroupRow = {
          depth,
          keys,
          labels,
          metrics: { ...n.metrics },
          count: n.count,
          condition: normalizeGroupCondition(n.condition),
        };
        if (depth === expandLevels - 1) {
          row.remainingGroupby = [];
        }
        if (n.children && n.children.length) {
          row.children = buildGroupRows(n.children, depth + 1);
        }
        return row;
      });
    };

    const resultTree = buildGroupRows(topNodes, 0);
    return resultTree as ReadGroupResult;
  }

  /**
   * Count the number of top-level ReadGroup groups.
   * - Only counts groups produced by the first groupby level.
   * - having and fields apply only to the top level.
   */
  static async ReadGroupCount<T extends BaseModel>(
    ModelCtor: RuntimeModelCtor<T>,
    groupby: Array<GroupBySpec<T> | GroupBySpec<T>[]> | [],
    condition: QueryCondition<T> | [] = [],
    options: ReadGroupCountOptions<T> = {}
  ): Promise<number> {
    const repo = ReadOperations.resolveRepository(ModelCtor, options);
    const timezone = options.timezone;
    const baseCondition = (condition ?? []) as BaseQueryCondition | [];

    if (!groupby || groupby.length === 0) {
      return 1;
    }

    const firstLevel = groupby[0];
    if (!firstLevel) throw new Error('ReadGroupCount requires non-empty groupby');
    const normalizedFirst = Array.isArray(firstLevel)
      ? normalizeGroupBySpecs(firstLevel as GroupBySpec<T>[])
      : normalizeGroupBySpec(firstLevel as GroupBySpec<T>);
    const fieldsAgg = (options.fields ?? []).map(normalizeFieldAggregation);
    const havingTop = Array.isArray(options.having) ? options.having[0] : options.having;

    // Composite levels must pass every component field down so the repository can count distinct composites correctly.
    const groupbyParam = Array.isArray(firstLevel)
      ? (rebuildCompositeGroupSpec(normalizedFirst as NormalizedCompositeGroupSpec) as GroupBySpec<T>[])
      : (rebuildGroupSpec(normalizedFirst as NormalizedGroupSpec) as GroupBySpec<T>);

    const total = await repo.readGroupCount({
      groupby: groupbyParam as GroupBySpec<T> | GroupBySpec<T>[],
      fields: rebuildAggFields(fieldsAgg),
      condition: toRepoCondition<T>(baseCondition),
      having: havingTop,
      timezone,
    });

    return Number(total) || 0;
  }
}

/* ---------------- Helpers and internal structures ---------------- */

type TreeNodeInternal = {
  keyAliases: string[]; // Group aliases for the current level. Single-column levels have length 1.
  keyValues: unknown[]; // Values aligned with keyAliases.
  key: ObjectRecord; // alias -> value
  metrics: ObjectRecord; // aggregation alias -> value
  count: number; // __count
  condition: BaseQueryCondition | [] | undefined; // Condition for the current group, including parent-chain filters.
  children?: TreeNodeInternal[];
};

function toArray<T>(x: T | T[]): T[] {
  return Array.isArray(x) ? x : [x];
}

// Fill temporal gaps for a specific level, but only when that level is the last expanded temporal-granularity layer.
function fillTemporalGapsForLevel(
  nodes: TreeNodeInternal[],
  spec: NormalizedGroupSpec,
  aggs: NormalizedAgg[],
  timezone: string | undefined,
  whereForLevel: BaseQueryCondition | [] | undefined
): TreeNodeInternal[] {
  if (!spec.isTime || !spec.granularity) return nodes;

  // Infer the range from groupby.range first, then fall back to the min and max timestamps already present in the nodes.
  let start: Date | undefined;
  let end: Date | undefined;

  if (spec.range?.start && spec.range?.end) {
    start = new Date(spec.range.start);
    end = new Date(spec.range.end);
  } else {
    const vals = nodes
      .map(n => {
        const v = n.key[spec.alias];
        return toDateIfPossible(v);
      })
      .filter((d): d is Date => d instanceof Date && !isNaN(d.getTime()));
    if (vals.length >= 1) {
      vals.sort((a, b) => a.getTime() - b.getTime());
      start = vals[0];
      end = vals[vals.length - 1];
    }
  }

  if (!start || !end) return nodes;

  const alignedStart = coerceToBucketStart(start, spec.granularity, timezone);
  const buckets = enumerateBuckets({ start: alignedStart, end, granularity: spec.granularity!, timezone, maxBuckets: 10000 });

  // Index existing nodes by UTC bucket-start milliseconds.
  const idx = new Map<number, TreeNodeInternal>();
  for (const n of nodes) {
    const dRaw = n.key[spec.alias];
    const d = toDateIfPossible(dRaw);
    if (!d || isNaN(d.getTime())) continue;
    const st = coerceToBucketStart(d, spec.granularity!, timezone);
    idx.set(st.getTime(), n);
  }

  const out: TreeNodeInternal[] = [];
  for (const b of buckets) {
    const key = b.getTime();
    const exists = idx.get(key);
    if (exists) {
      out.push(exists);
    } else {
      // Build a synthetic zero-value node.
      const zeroMetrics: ObjectRecord = {};
      for (const a of aggs) {
        if (a.agg === 'sum' || a.agg === 'count' || a.agg === 'count_distinct') zeroMetrics[a.alias] = 0;
        else zeroMetrics[a.alias] = null;
      }
      const thisCondition = buildTemporalCondition(spec.field, spec.granularity!, b, timezone);
      out.push({
        keyAliases: [spec.alias],
        keyValues: [b],
        key: { [spec.alias]: b },
        metrics: zeroMetrics,
        count: 0,
        condition: andAll(whereForLevel, thisCondition),
        children: [],
      });
    }
  }

  return out;
}

// Build a group condition using either plain equality or a temporal range.
// moment.parseZone is used so we do not depend on Intl behavior.
//
// D14 (drill inherit): when the bucket key is already an offset-aware ISO string
// from DATE_TRUNC, the UTC [start, end) is derived from that key alone — the
// timezone argument must not re-bucket. Downstream Search uses this condition as
// parentCondition; do not call businessYesterday / re-resolve "yesterday" in another TZ.
function buildGroupCondition(spec: NormalizedGroupSpec, value: unknown, timezone?: string): BaseQueryCondition {
  if (!spec.isTime || !spec.granularity) {
    return [spec.field, '=', value];
  }

  // 1) When value is an offset-aware ISO string returned by DATE_TRUNC, derive the local [start, end) range using that offset and convert it to UTC.
  const fromIso = rangeFromGroupedValue(value, spec.granularity);
  if (fromIso) {
    return {
      And: [
        [spec.field, '>=', fromIso.start.toISOString()],
        [spec.field, '<', fromIso.end.toISOString()],
      ],
    };
  }

  // 2) Fallback: compute bucket boundaries from timezone using the existing alignment helpers, then emit UTC ISO strings.
  const d = toDateIfPossible(value);
  if (!d) {
    return [spec.field, '=', value];
  }
  const bucketStart = coerceToBucketStart(d, spec.granularity, timezone);
  return buildTemporalCondition(spec.field, spec.granularity, bucketStart, timezone);
}

function toDateIfPossible(value: unknown): Date | undefined {
  if (value instanceof Date) return value;
  if (typeof value === 'string' || typeof value === 'number') {
    const date = new Date(value);
    return isNaN(date.getTime()) ? undefined : date;
  }
  return undefined;
}

// Use moment.parseZone to compute [start, end) with the input offset preserved, matching DB date_trunc semantics.
function rangeFromGroupedValue(value: unknown, granularity: TemporalGranularity): { start: Date; end: Date } | undefined {
  if (typeof value !== 'string') return undefined;
  const m = moment.parseZone(value, moment.ISO_8601, true); // Preserve offsets such as '+08:00'.
  if (!m.isValid()) return undefined;

  const unit = (granularity === 'week' ? 'isoWeek' : granularity) as moment.unitOfTime.StartOf;
  const start = m.clone().startOf(unit);
  const end = start.clone().add(1, granularity as moment.unitOfTime.DurationConstructor);

  return { start: start.toDate(), end: end.toDate() }; // Date instances represent UTC instants.
}

// Keep this timezone-based bucket-boundary builder for the fallback path.
function buildTemporalCondition(field: string, granularity: TemporalGranularity, bucketStart: Date, _timezone?: string): BaseQueryCondition {
  const start = bucketStart;
  const end = nextBucket(bucketStart, granularity);
  return {
    And: [
      [field, '>=', start.toISOString()],
      [field, '<', end.toISOString()],
    ],
  };
}

type TreeResultNode = {
  key: ObjectRecord;
  metrics: ObjectRecord;
  count: number;
  condition?: BaseQueryCondition;
  total?: true;
  children: TreeResultNode[];
};

function toTreeResult(nodes: TreeNodeInternal[]): TreeResultNode[] {
  const convert = (n: TreeNodeInternal): TreeResultNode => ({
    key: { ...n.key },
    metrics: { ...n.metrics },
    count: n.count,
    condition: normalizeGroupCondition(n.condition),
    total: n.keyAliases.length === 0 && Object.keys(n.key || {}).length === 0 ? true : undefined,
    children: (n.children || []).map(convert),
  });
  return nodes.map(convert);
}

function normalizeGroupCondition(condition: BaseQueryCondition | [] | undefined): BaseQueryCondition | undefined {
  if (!condition) return undefined;
  if (Array.isArray(condition) && condition.length === 0) return undefined;
  return condition as BaseQueryCondition;
}

function toPascal(parts: string[]): string {
  return parts
    .filter(Boolean)
    .map(p =>
      p
        .replace(/[_\s]+/g, ' ')
        .split(' ')
        .filter(Boolean)
        .map(w => w[0].toUpperCase() + w.slice(1).toLowerCase())
        .join('')
    )
    .join('');
}

function aliasCandidates(alias: string): string[] {
  const raw = String(alias || '');
  if (!raw) return [];

  const set = new Set<string>();
  set.add(raw);

  const lower = raw.toLowerCase();
  set.add(lower);

  const collapsed = raw.replace(/__+/g, '_');
  set.add(collapsed);
  set.add(collapsed.toLowerCase());

  const parts = raw.split('__');

  if (parts.length > 0) {
    const head = parts.shift() as string;
    const restPascal = toPascal(parts);
    const headSnake = head.replace(/([a-z0-9])([A-Z])/g, '$1_$2').toLowerCase();
    const headPascal = toPascal(headSnake.split('_'));
    const pascal = `${headPascal}${restPascal}`;
    if (pascal) set.add(pascal);
    const camel = pascal ? pascal[0].toLowerCase() + pascal.slice(1) : '';
    if (camel) set.add(camel);
  }

  if (raw.startsWith('__')) {
    const trimmed = raw.replace(/^_+/, '');
    if (trimmed) {
      const camel = trimmed[0].toLowerCase() + trimmed.slice(1);
      const pascal = trimmed[0].toUpperCase() + trimmed.slice(1);
      set.add(trimmed);
      set.add(camel);
      set.add(pascal);
    }
  }

  return Array.from(set);
}

function pickAliasedValue(row: ObjectRecord, alias: string): unknown {
  for (const key of aliasCandidates(alias)) {
    if (Object.prototype.hasOwnProperty.call(row, key)) {
      return row[key];
    }
  }
  return row[alias];
}

// ===== Display labels (ISO-style) =====
function pickGranularityFromAlias(alias: string): TemporalGranularity | '' {
  const m = /__(year|quarter|month|week|day)$/i.exec(alias || '');
  const value = m?.[1]?.toLowerCase();
  if (value === 'year' || value === 'quarter' || value === 'month' || value === 'week' || value === 'day') {
    return value;
  }
  return '';
}
function pad2(n: number) {
  return n < 10 ? `0${n}` : String(n);
}
function formatGroupDisplay(alias: string, raw: unknown): string {
  const g = pickGranularityFromAlias(alias);
  if (!g) return String(raw ?? '');
  const m =
    typeof raw === 'string'
      ? moment.parseZone(raw, moment.ISO_8601, true)
      : raw === undefined
        ? moment()
        : raw instanceof Date || typeof raw === 'number'
          ? moment(raw)
          : moment.invalid();
  if (!m.isValid()) return String(raw ?? '');
  const formatters: Record<TemporalGranularity, () => string> = {
    year: () => m.format('YYYY'), // 2025
    quarter: () => {
      const q = Math.floor(m.month() / 3) + 1; // 1..4
      return `${m.year()}-Q${q}`; // 2025-Q4
    },
    month: () => m.format('YYYY-MM'), // 2025-11
    week: () => m.format('GGGG-[W]WW'), // 2025-W45
    day: () => m.format('YYYY-MM-DD'), // 2025-11-06
  };

  return formatters[g]();
}

export const __toArrayForTest = toArray;
export const __fillTemporalGapsForLevelForTest = fillTemporalGapsForLevel;
export const __buildGroupConditionForTest = buildGroupCondition;
export const __rangeFromGroupedValueForTest = rangeFromGroupedValue;
export const __toTreeResultForTest = toTreeResult;
export const __aliasCandidatesForTest = aliasCandidates;
export const __pickAliasedValueForTest = pickAliasedValue;
export const __formatGroupDisplayForTest = formatGroupDisplay;
