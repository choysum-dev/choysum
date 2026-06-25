// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import { sql } from 'kysely';
import Job from '@/task/service/models/job';

type ModuleOriginType = 'local' | 'registry';

type RequestSyncParams = {
  originType?: ModuleOriginType;
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

type SortSpec = { field: string; desc: boolean };

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

    const originTypes = Array.from(
      new Set(
        bucket
          .map(row => toText(row?.OriginType))
          .filter(value => value === 'local' || value === 'registry')
      )
    ).sort((a, b) => {
      if (a === b) return 0;
      if (a === 'local') return -1;
      if (b === 'local') return 1;
      return a.localeCompare(b);
    });

    const localVersion = String(local?.Version || '').trim();
    const registryVersion = String(registry?.Version || '').trim();
    const installedStatus =
      String(local?.InstalledStatus || '').trim() ||
      String(registry?.InstalledStatus || '').trim() ||
      String(base?.InstalledStatus || '').trim();
    const installedVersion =
      String(local?.InstalledVersion || '').trim() ||
      String(registry?.InstalledVersion || '').trim() ||
      String(base?.InstalledVersion || '').trim();

    merged.push({
      ...base,
      Id: String(base?.Id || '').trim() || String(local?.Id || '').trim() || String(registry?.Id || '').trim(),
      ModuleName: moduleName,
      OriginType: String(base?.OriginType || '').trim() || (originTypes[0] as string) || 'local',
      OriginTypes: originTypes.length ? originTypes.join(', ') : String(base?.OriginType || '').trim(),
      OriginRef:
        String(local?.OriginRef || '').trim() ||
        String(registry?.OriginRef || '').trim() ||
        String(base?.OriginRef || '').trim(),
      Available: bucket.some(row => row?.Available !== false),
      Version: registryVersion || localVersion || String(base?.Version || '').trim(),
      LocalVersion: localVersion || undefined,
      RegistryVersion: registryVersion || undefined,
      ManifestJson: local?.ManifestJson ?? registry?.ManifestJson ?? base?.ManifestJson ?? null,
      LocalPath: String(local?.LocalPath || '').trim() || String(base?.LocalPath || '').trim() || undefined,
      LastSyncAt: pickNewestTimestamp([local?.LastSyncAt, registry?.LastSyncAt, base?.LastSyncAt]),
      LastBatchSyncAt: pickNewestTimestamp([local?.LastBatchSyncAt, registry?.LastBatchSyncAt, base?.LastBatchSyncAt]),
      SyncRevision:
        String(registry?.SyncRevision || '').trim() ||
        String(local?.SyncRevision || '').trim() ||
        String(base?.SyncRevision || '').trim() ||
        undefined,
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

function normalizeOriginType(value?: string): ModuleOriginType {
  const raw = String(value || '')
    .trim()
    .toLowerCase();
  if (raw === 'registry') return 'registry';
  return 'local';
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

async function findRunningJobId(fullMethod: string): Promise<string> {
  const running = await Job.Search(
    {
      And: [
        ['TargetApp', '=', 'meta'],
        ['FullMethod', '=', fullMethod],
        ['Status', 'in', ['queued', 'dispatching'] as any],
      ],
    } as any,
    { limit: 1, orderBy: { field: 'CreatedAt', order: 'desc' } as any, fields: ['Id'] as any } as any
  );
  const jobId = String(running?.[0]?.Id || '').trim();
  return jobId;
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
    const emptyArray = Array.isArray(condition) && condition.length === 0;
    const emptyObject = !Array.isArray(condition) && condition && typeof condition === 'object' && Object.keys(condition).length === 0;
    const normalized = emptyArray || emptyObject ? (['Available', '=', true] as any) : condition;

    const rawOptions = { ...(options || {}) };
    const sortSpecs = parseSortSpecs(rawOptions.orderBy);
    const offset = normalizeOffset(rawOptions.offset);
    const limit = normalizeLimit(rawOptions.limit);
    const requestedFields = normalizeFields(rawOptions.fields);
    const aggregationRequiredFields = [
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
    rawOptions.fields = Array.from(new Set([...requestedFields, ...aggregationRequiredFields]));
    delete rawOptions.limit;
    delete rawOptions.offset;

    const rows = (await (BaseModel as any).Search.call(this, normalized, rawOptions)) as any[];
    const merged = aggregateRows((rows || []).map(toPlainRecord));
    merged.sort((a, b) => compareBySpecs(a, b, sortSpecs));

    const start = offset;
    const end = limit == null ? undefined : start + limit;
    return merged.slice(start, end) as unknown as T[];
  }

  static async Count<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: any[] | Record<string, any> = [],
    options?: any
  ): Promise<number> {
    const emptyArray = Array.isArray(condition) && condition.length === 0;
    const emptyObject = !Array.isArray(condition) && condition && typeof condition === 'object' && Object.keys(condition).length === 0;
    const normalized = emptyArray || emptyObject ? (['Available', '=', true] as any) : condition;

    const rawOptions = { ...(options || {}) };
    delete rawOptions.limit;
    delete rawOptions.offset;
    delete rawOptions.fields;
    delete rawOptions.orderBy;

    const rows = (await (BaseModel as any).Search.call(this, normalized, rawOptions)) as any[];
    const merged = aggregateRows((rows || []).map(toPlainRecord));
    return merged.length;
  }

  static async RequestSync(params: RequestSyncParams = {}): Promise<string> {
    const force = !!params.force;
    const ifStale = !!params.ifStale;
    if (!force && !ifStale) return '';

    if (ifStale && !force && isTruthyFlag(getBackendEnvText('CHOYSUM_E2E_SKIP_INDEX_STALE_SYNC', 'choysum_e2e_skip_index_stale_sync'))) {
      return '';
    }

    const fullMethod = 'meta.IrModuleIndex/Sync';

    // Reuse an in-flight sync job so stale-triggered calls do not enqueue
    // competing writers against the same index table on sqlite.
    const runningJobId = await findRunningJobId(fullMethod);
    if (runningJobId) return runningJobId;

    const originType = normalizeOriginType(params.originType);
    if (ifStale && !force) {
      const repo = this.getRepository();
      let query = repo
        .selectQueryBuilder()
        .select((eb: any) => eb.fn.max('last_batch_sync_at').as('last_batch_sync_at'))
        .where('meta_ir_module_index.origin_type' as any, '=', originType as any);
      if (originType === 'local') {
        query = query.where('meta_ir_module_index.origin_ref' as any, '=', 'local');
      }
      const rows = await repo.execute(query);
      const row = rows?.[0] as any;
      const lastBatchSyncAt = row?.lastBatchSyncAt ?? row?.last_batch_sync_at ?? null;
      if (lastBatchSyncAt) {
        const now = Date.now();
        const lastTime = new Date(lastBatchSyncAt as string).getTime();
        if (!isNaN(lastTime)) {
          const ttlMs = originType === 'registry' ? 10 * 60 * 1000 : 1 * 60 * 1000;
          if (now - lastTime < ttlMs) {
            return ''; // within staleness window, skip
          }
        }
      }
    }

    const userId = ensureCurrentUserId();
    const job = await Job.EnqueueJob('meta', fullMethod, { originType, force }, userId, userId, undefined, 0, 0);
    return String((job as any)?.Id || '').trim();
  }

  static async Sync(originType?: ModuleOriginType, force?: boolean): Promise<any> {
    const bridge = getModuleManagementBridge();
    const syncIndex = (bridge as any)?.syncIndex;
    if (typeof syncIndex !== 'function') {
      throw new Error('moduleManagement.syncIndex is not implemented');
    }
    return await syncIndex({ originType: normalizeOriginType(originType), force: !!force });
  }
}
