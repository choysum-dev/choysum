// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import { sql } from 'kysely';
import Job from '@/task/service/models/job';

type ModuleOriginType = 'local' | 'registry';
type ModuleSyncOriginType = ModuleOriginType | 'all';

type RequestSyncParams = {
  originType?: ModuleSyncOriginType;
  force?: boolean;
  ifStale?: boolean;
};

type ModuleIndexRecord = {
  Id?: string;
  ModuleName?: string;
  OriginType?: string;
  OriginRef?: string;
  Available?: boolean;
  Version?: string;
  ManifestJson?: Record<string, unknown> | null;
  LocalPath?: string;
  LastSyncAt?: Date | string | null;
  LastBatchSyncAt?: Date | string | null;
  SyncRevision?: string;
  LastErrorMessage?: string;
  InstalledStatus?: string;
  InstalledVersion?: string;
  OriginTypes?: string;
  LocalVersion?: string;
  RegistryVersion?: string;
};

function normalizeSearchCondition(condition: any[] | Record<string, any>): any {
  const emptyArray = Array.isArray(condition) && condition.length === 0;
  const emptyObject = !Array.isArray(condition) && condition && typeof condition === 'object' && Object.keys(condition).length === 0;
  return emptyArray || emptyObject ? (['Available', '=', true] as any) : condition;
}

type SortSpec = { field: string; desc: boolean };
type GroupSortSpec = { field: string; order: 'asc' | 'desc' };

function toText(value: unknown): string {
  return String(value ?? '')
    .trim()
    .toLowerCase();
}

function toComparableValue(value: unknown): unknown {
  if (value == null) return null;
  if (value instanceof Date) return value.getTime();
  if (typeof value === 'boolean') return value ? 1 : 0;
  if (typeof value === 'number' || typeof value === 'bigint') return value;
  if (typeof value === 'string') {
    const raw = value.trim();
    if (!raw) return '';
    const ts = Date.parse(raw);
    if (!isNaN(ts) && /\d{4}-\d{2}-\d{2}/.test(raw)) {
      return ts;
    }
    return raw.toLowerCase();
  }
  return String(value).toLowerCase();
}

function parseSortSpecs(orderBy: any): SortSpec[] {
  if (!orderBy) return [];
  const rawList = Array.isArray(orderBy) ? orderBy : [orderBy];
  const specs: SortSpec[] = [];
  for (const item of rawList) {
    if (!item) continue;
    if (typeof item === 'string') {
      specs.push({ field: item, desc: false });
      continue;
    }
    if (typeof item !== 'object') continue;
    const field = String((item as any).field ?? (item as any).Field ?? '').trim();
    if (!field) continue;
    const orderText = toText((item as any).order ?? (item as any).Order ?? 'asc');
    specs.push({ field, desc: orderText === 'desc' });
  }
  return specs;
}

function compareBySpecs(a: ModuleIndexRecord, b: ModuleIndexRecord, specs: SortSpec[]): number {
  for (const spec of specs) {
    const av = toComparableValue((a as any)?.[spec.field]);
    const bv = toComparableValue((b as any)?.[spec.field]);
    if (av == null && bv == null) continue;
    if (av == null) return spec.desc ? 1 : -1;
    if (bv == null) return spec.desc ? -1 : 1;
    if (av > bv) return spec.desc ? -1 : 1;
    if (av < bv) return spec.desc ? 1 : -1;
  }
  const am = toText(a?.ModuleName);
  const bm = toText(b?.ModuleName);
  if (am > bm) return 1;
  if (am < bm) return -1;
  return 0;
}

function normalizeOffset(raw: unknown): number {
  const value = Number(raw);
  if (!Number.isFinite(value) || value < 0) return 0;
  return Math.floor(value);
}

function normalizeLimit(raw: unknown): number | null {
  const value = Number(raw);
  if (!Number.isFinite(value) || value <= 0) return null;
  return Math.floor(value);
}

function normalizeFields(raw: unknown): string[] {
  if (!Array.isArray(raw)) return [];
  const out: string[] = [];
  for (const item of raw) {
    const field = String(item || '').trim();
    if (!field) continue;
    out.push(field);
  }
  return out;
}

function applySoftDeleteOptions(target: Record<string, unknown>, source: Record<string, unknown>): void {
  if (Object.prototype.hasOwnProperty.call(source, 'withDeleted')) {
    target.withDeleted = !!(source as any).withDeleted;
  }
  if (Object.prototype.hasOwnProperty.call(source, 'onlyDeleted')) {
    target.onlyDeleted = !!(source as any).onlyDeleted;
  }
}

function buildSortPushdownPlan(sortSpecs: SortSpec[]): {
  supported: boolean;
  orderBy: GroupSortSpec[];
  aggregateFields: Array<{ field: string; agg: 'max'; alias: string }>;
} {
  const orderBy: GroupSortSpec[] = [];
  const aggregateFields: Array<{ field: string; agg: 'max'; alias: string }> = [];
  const aggregateAliasByField = new Map<string, string>();

  const ensureAggregateAlias = (field: string, alias: string): string => {
    const existing = aggregateAliasByField.get(field);
    if (existing) return existing;
    aggregateAliasByField.set(field, alias);
    aggregateFields.push({ field, agg: 'max', alias });
    return alias;
  };

  for (const spec of sortSpecs) {
    const order: 'asc' | 'desc' = spec.desc ? 'desc' : 'asc';
    if (spec.field === 'ModuleName') {
      orderBy.push({ field: 'ModuleName', order });
      continue;
    }
    if (spec.field === 'Available') {
      orderBy.push({ field: ensureAggregateAlias('Available', '__order_available'), order });
      continue;
    }
    if (spec.field === 'LastSyncAt') {
      orderBy.push({ field: ensureAggregateAlias('LastSyncAt', '__order_last_sync_at'), order });
      continue;
    }
    if (spec.field === 'LastBatchSyncAt') {
      orderBy.push({ field: ensureAggregateAlias('LastBatchSyncAt', '__order_last_batch_sync_at'), order });
      continue;
    }
    return { supported: false, orderBy: [], aggregateFields: [] };
  }

  if (!orderBy.some(item => item.field === 'ModuleName')) {
    orderBy.push({ field: 'ModuleName', order: 'asc' });
  }

  return {
    supported: true,
    orderBy,
    aggregateFields,
  };
}

function extractGroupedModuleNames(rows: any[]): string[] {
  const out: string[] = [];
  for (const row of rows || []) {
    const moduleName = String((row as any)?.ModuleName ?? (row as any)?.module_name ?? '').trim();
    if (!moduleName) continue;
    out.push(moduleName);
  }
  return out;
}

function buildModuleNamesCondition(baseCondition: any, moduleNames: string[]): any {
  if (!Array.isArray(moduleNames) || moduleNames.length === 0) {
    return ['Id', '=', '__never_match__'] as any;
  }
  return {
    And: [baseCondition as any, ['ModuleName', 'in', moduleNames] as any],
  } as any;
}

function projectFields(rows: ModuleIndexRecord[], requestedFields: string[]): ModuleIndexRecord[] {
  if (!Array.isArray(requestedFields) || requestedFields.length === 0) return rows;
  const fields = Array.from(new Set(requestedFields.map(field => String(field || '').trim()).filter(Boolean)));
  if (fields.length === 0) return rows;

  return rows.map(row => {
    const projected: ModuleIndexRecord = {};
    for (const field of fields) {
      (projected as any)[field] = (row as any)?.[field];
    }
    return projected;
  });
}

function toPlainRecord(input: any): ModuleIndexRecord {
  if (!input || typeof input !== 'object') return {};
  if (typeof input.toPlainObject === 'function') {
    try {
      return { ...(input.toPlainObject() as Record<string, unknown>) } as ModuleIndexRecord;
    } catch {
      // fall back to enumerable keys
    }
  }
  const out: Record<string, unknown> = {};
  for (const key of Object.keys(input)) {
    out[key] = input[key];
  }
  return out as ModuleIndexRecord;
}

function pickNewestTimestamp(values: Array<Date | string | null | undefined>): Date | string | null | undefined {
  let picked: Date | string | null | undefined;
  let pickedTs = -1;
  for (const value of values) {
    if (!value) continue;
    const ts = value instanceof Date ? value.getTime() : Date.parse(String(value));
    if (isNaN(ts)) {
      if (picked == null) picked = value;
      continue;
    }
    if (ts > pickedTs) {
      pickedTs = ts;
      picked = value;
    }
  }
  return picked;
}

function aggregateRows(rows: ModuleIndexRecord[]): ModuleIndexRecord[] {
  const byModule = new Map<string, ModuleIndexRecord[]>();
  for (const raw of rows) {
    const moduleName = String(raw?.ModuleName || '').trim();
    if (!moduleName) continue;
    const bucket = byModule.get(moduleName);
    if (bucket) bucket.push(raw);
    else byModule.set(moduleName, [raw]);
  }

  const merged: ModuleIndexRecord[] = [];
  for (const [moduleName, bucket] of byModule.entries()) {
    const local = bucket.find(row => toText(row?.OriginType) === 'local');
    const registry = bucket.find(row => toText(row?.OriginType) === 'registry');
    const base = local || registry || bucket[0];

    const originTypes = Array.from(new Set(bucket.map(row => toText(row?.OriginType)).filter(value => value === 'local' || value === 'registry'))).sort(
      (a, b) => {
        if (a === b) return 0;
        if (a === 'local') return -1;
        if (b === 'local') return 1;
        return a.localeCompare(b);
      }
    );

    const localVersion = String(local?.Version || '').trim();
    const registryVersion = String(registry?.Version || '').trim();
    const installedStatus =
      String(local?.InstalledStatus || '').trim() || String(registry?.InstalledStatus || '').trim() || String(base?.InstalledStatus || '').trim();
    const installedVersion =
      String(local?.InstalledVersion || '').trim() || String(registry?.InstalledVersion || '').trim() || String(base?.InstalledVersion || '').trim();

    merged.push({
      ...base,
      Id: String(base?.Id || '').trim() || String(local?.Id || '').trim() || String(registry?.Id || '').trim(),
      ModuleName: moduleName,
      OriginType: String(base?.OriginType || '').trim() || (originTypes[0] as string) || 'local',
      OriginTypes: originTypes.length ? originTypes.join(', ') : String(base?.OriginType || '').trim(),
      OriginRef: String(local?.OriginRef || '').trim() || String(registry?.OriginRef || '').trim() || String(base?.OriginRef || '').trim(),
      Available: bucket.some(row => row?.Available !== false),
      Version: registryVersion || localVersion || String(base?.Version || '').trim(),
      LocalVersion: localVersion || undefined,
      RegistryVersion: registryVersion || undefined,
      ManifestJson: local?.ManifestJson ?? registry?.ManifestJson ?? base?.ManifestJson ?? null,
      LocalPath: String(local?.LocalPath || '').trim() || String(base?.LocalPath || '').trim() || undefined,
      LastSyncAt: pickNewestTimestamp([local?.LastSyncAt, registry?.LastSyncAt, base?.LastSyncAt]),
      LastBatchSyncAt: pickNewestTimestamp([local?.LastBatchSyncAt, registry?.LastBatchSyncAt, base?.LastBatchSyncAt]),
      SyncRevision:
        String(registry?.SyncRevision || '').trim() || String(local?.SyncRevision || '').trim() || String(base?.SyncRevision || '').trim() || undefined,
      InstalledStatus: installedStatus || 'uninstalled',
      InstalledVersion: installedVersion || undefined,
      LastErrorMessage:
        String(registry?.LastErrorMessage || '').trim() ||
        String(local?.LastErrorMessage || '').trim() ||
        String(base?.LastErrorMessage || '').trim() ||
        undefined,
    });
  }

  return merged;
}

function normalizeOriginType(value?: string): ModuleSyncOriginType | '' {
  const raw = String(value || '')
    .trim()
    .toLowerCase();
  if (raw === 'all') return 'all';
  if (raw === '') return 'all';
  if (raw === 'local') return 'local';
  if (raw === 'registry') return 'registry';
  return '';
}

function canReuseRunningSync(requested: ModuleSyncOriginType, running: ModuleSyncOriginType): boolean {
  if (requested === 'all') return running === 'all';
  if (running === 'all') return true;
  return running === requested;
}

function getBackendEnv(): Record<string, unknown> {
  return (((import.meta as any)?.env || (globalThis as any)?.__choysumBackendEnv || {}) as Record<string, unknown>) || {};
}

function getBackendEnvText(...keys: string[]): string {
  const env = getBackendEnv();
  for (const key of keys) {
    const value = String((env as any)?.[key] || '').trim();
    if (value) return value;
  }
  return '';
}

function isTruthyFlag(value: string): boolean {
  const raw = value.trim().toLowerCase();
  return raw === '1' || raw === 'true' || raw === 'yes' || raw === 'on';
}

function ensureCurrentUserId(): string {
  const userId = String(getUserId() || '').trim();
  if (userId) return userId;
  const fallback = getBackendEnvText('CHOYSUM_E2E_OPERATOR_USER_ID', 'choysum_e2e_operator_user_id');
  if (fallback) return fallback;
  throw new Error('current user is required');
}

function getModuleManagementBridge(): any {
  const root: any = (globalThis as any)?.$choysum;
  if (!root?.moduleManagement) {
    throw new Error('moduleManagement bridge is not injected');
  }
  return root.moduleManagement;
}

async function findRunningJobId(fullMethod: string, requestedOrigin: ModuleSyncOriginType): Promise<string> {
  const running = await Job.Search(
    {
      And: [
        ['TargetApp', '=', 'meta'],
        ['FullMethod', '=', fullMethod],
        ['Status', 'in', ['queued', 'dispatching'] as any],
      ],
    } as any,
    { limit: 20, orderBy: { field: 'CreatedAt', order: 'desc' } as any, fields: ['Id', 'PayloadJson'] as any } as any
  );
  for (const row of running || []) {
    const jobId = String((row as any)?.Id || '').trim();
    if (!jobId) continue;
    const runningOrigin = normalizeOriginType((row as any)?.PayloadJson?.originType);
    if (!runningOrigin) continue;
    if (canReuseRunningSync(requestedOrigin, runningOrigin)) {
      return jobId;
    }
  }
  return '';
}

@Model('IrModuleIndex', {
  tableName: 'meta_ir_module_index',
  autoMigrate: false,
})
export default class IrModuleIndex extends BaseModel {
  @Field({ type: 'varchar', column: { size: 255, notNull: true } })
  ModuleName!: string;

  @Field({ type: 'varchar', column: { size: 32, notNull: true } })
  OriginType!: ModuleOriginType;

  @Field({ type: 'varchar', column: { size: 255, notNull: true } })
  OriginRef!: string;

  @Field({ type: 'boolean', column: { notNull: true } })
  Available!: boolean;

  @Field({ type: 'varchar', column: { size: 255 } })
  Version?: string;

  @Field({ type: 'jsonobject' })
  ManifestJson?: Record<string, unknown> | null;

  @Field({ type: 'varchar', column: { size: 512 } })
  LocalPath?: string;

  @Field({ type: 'datetime' })
  LastSyncAt!: Date;

  @Field({ type: 'datetime' })
  LastBatchSyncAt!: Date;

  @Field({ type: 'varchar', column: { size: 255 } })
  SyncRevision?: string;

  @Field({ type: 'text' })
  LastErrorMessage?: string;

  @Field({
    type: 'varchar',
    select: {
      expr: ({ col }) => col('meta_ir_module_index', 'origin_type'),
      size: 64,
    },
  })
  OriginTypes?: string;

  @Field({
    type: 'varchar',
    select: {
      expr: ({ col }) => col('meta_ir_module_index', 'version'),
      size: 255,
    },
  })
  LocalVersion?: string;

  @Field({
    type: 'varchar',
    select: {
      expr: ({ col }) => col('meta_ir_module_index', 'version'),
      size: 255,
    },
  })
  RegistryVersion?: string;

  @Field({
    type: 'varchar',
    select: {
      expr: ({ selectFrom, col }) =>
        sql<string>`coalesce((${selectFrom('meta_ir_module as m')
          .select('m.status')
          .whereRef('m.name', '=', col('meta_ir_module_index', 'module_name'))
          .limit(1)}), 'uninstalled')`,
      size: 64,
    },
  })
  InstalledStatus?: string;

  @Field({
    type: 'varchar',
    select: {
      expr: ({ selectFrom, col }) =>
        selectFrom('meta_ir_module as m').select('m.version').whereRef('m.name', '=', col('meta_ir_module_index', 'module_name')).limit(1),
      size: 255,
    },
  })
  InstalledVersion?: string;

  static async Search<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: any[] | Record<string, any> = [],
    options?: any
  ): Promise<T[]> {
    const normalized = normalizeSearchCondition(condition);
    const rawOptions = { ...(options || {}) };
    const requestedFields = normalizeFields(rawOptions.fields);
    const sortSpecs = parseSortSpecs(rawOptions.orderBy);
    const offset = normalizeOffset(rawOptions.offset);
    const limit = normalizeLimit(rawOptions.limit);
    const sortPlan = buildSortPushdownPlan(sortSpecs);

    const readGroupOptions: Record<string, unknown> = {
      groupby: 'ModuleName',
      fields: sortPlan.aggregateFields.map(item => `${item.field}:${item.agg}`),
    };
    if (sortPlan.supported) {
      readGroupOptions.offset = offset;
      if (limit != null) {
        readGroupOptions.limit = limit;
      }
      readGroupOptions.orderBy = sortPlan.orderBy;
    }
    applySoftDeleteOptions(readGroupOptions, rawOptions);

    const groupedRows = await this.getRepository().readGroup({
      ...readGroupOptions,
      condition: normalized,
    } as any);
    const groupedModuleNames = extractGroupedModuleNames(groupedRows as any[]);
    if (groupedModuleNames.length === 0) {
      return [] as unknown as T[];
    }

    const detailFields = [
      'Id',
      'ModuleName',
      'OriginType',
      'OriginRef',
      'Available',
      'Version',
      'ManifestJson',
      'LocalPath',
      'LastSyncAt',
      'LastBatchSyncAt',
      'SyncRevision',
      'LastErrorMessage',
      'InstalledStatus',
      'InstalledVersion',
    ];
    const detailOptions: Record<string, unknown> = {
      fields: detailFields,
      limit: Math.max(groupedModuleNames.length * 2, groupedModuleNames.length),
      orderBy: [{ field: 'ModuleName', order: 'asc' }],
    };
    applySoftDeleteOptions(detailOptions, rawOptions);

    const detailRows = (await (BaseModel as any).Search.call(this, buildModuleNamesCondition(normalized, groupedModuleNames), detailOptions)) as any[];

    const mergedByModule = new Map<string, ModuleIndexRecord>();
    for (const row of aggregateRows((detailRows || []).map(toPlainRecord))) {
      const moduleName = String(row?.ModuleName || '').trim();
      if (!moduleName) continue;
      mergedByModule.set(moduleName, row);
    }

    const ordered: ModuleIndexRecord[] = [];
    for (const moduleName of groupedModuleNames) {
      const hit = mergedByModule.get(moduleName);
      if (hit) ordered.push(hit);
    }

    let finalRows = ordered;
    if (!sortPlan.supported) {
      ordered.sort((a, b) => compareBySpecs(a, b, sortSpecs));
      const start = offset;
      const end = limit == null ? undefined : start + limit;
      finalRows = ordered.slice(start, end);
    }

    return projectFields(finalRows, requestedFields) as unknown as T[];
  }

  static async Count<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: any[] | Record<string, any> = [],
    options?: any
  ): Promise<number> {
    const normalized = normalizeSearchCondition(condition);
    const readGroupCountOptions: Record<string, unknown> = {
      groupby: 'ModuleName',
      condition: normalized,
    };
    applySoftDeleteOptions(readGroupCountOptions, { ...(options || {}) });
    return await this.getRepository().readGroupCount(readGroupCountOptions as any);
  }

  static async RequestSync(params: RequestSyncParams = {}): Promise<string> {
    const force = !!params.force;
    const ifStale = !!params.ifStale;
    if (!force && !ifStale) return '';
    const originType = normalizeOriginType(params.originType);
    if (!originType) {
      throw new Error('originType must be one of: local, registry, all');
    }

    if (ifStale && !force && isTruthyFlag(getBackendEnvText('CHOYSUM_E2E_SKIP_INDEX_STALE_SYNC', 'choysum_e2e_skip_index_stale_sync'))) {
      return '';
    }

    const fullMethod = 'meta.IrModuleIndex/Sync';

    // Reuse in-flight jobs for non-force requests to reduce contention.
    if (!force) {
      const runningJobId = await findRunningJobId(fullMethod, originType);
      if (runningJobId) return runningJobId;
    }
    if (ifStale && !force) {
      const repo = this.getRepository();
      const isOriginStale = async (target: ModuleOriginType): Promise<boolean> => {
        let query = repo
          .selectQueryBuilder()
          .select((eb: any) => eb.fn.max('last_batch_sync_at').as('last_batch_sync_at'))
          .where('meta_ir_module_index.origin_type' as any, '=', target as any);
        if (target === 'local') {
          query = query.where('meta_ir_module_index.origin_ref' as any, '=', 'local');
        }
        const rows = await repo.execute(query);
        const row = rows?.[0] as any;
        const lastBatchSyncAt = row?.lastBatchSyncAt ?? row?.last_batch_sync_at ?? null;
        if (!lastBatchSyncAt) {
          return true;
        }

        const lastTime = new Date(lastBatchSyncAt as string).getTime();
        if (isNaN(lastTime)) {
          return true;
        }

        const ttlMs = target === 'registry' ? 10 * 60 * 1000 : 1 * 60 * 1000;
        return Date.now() - lastTime >= ttlMs;
      };

      if (originType === 'all') {
        const [registryStale, localStale] = await Promise.all([isOriginStale('registry'), isOriginStale('local')]);
        if (!registryStale && !localStale) {
          return ''; // both origins are still fresh
        }
      } else {
        const stale = await isOriginStale(originType);
        if (!stale) {
          return ''; // within staleness window, skip
        }
      }
    }

    const userId = ensureCurrentUserId();
    const job = await Job.EnqueueJob('meta', fullMethod, { originType, force }, userId, userId, undefined, 0, 0);
    return String((job as any)?.Id || '').trim();
  }

  static async Sync(originType?: ModuleSyncOriginType, force?: boolean): Promise<any> {
    const bridge = getModuleManagementBridge();
    const syncIndex = (bridge as any)?.syncIndex;
    if (typeof syncIndex !== 'function') {
      throw new Error('moduleManagement.syncIndex is not implemented');
    }
    const normalizedOriginType = normalizeOriginType(originType);
    if (!normalizedOriginType) {
      throw new Error('originType must be one of: local, registry, all');
    }
    return await syncIndex({ originType: normalizedOriginType, force: !!force });
  }
}
