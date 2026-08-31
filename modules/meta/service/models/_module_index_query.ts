// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Pure query/sort helpers for MetaModuleIndex Search/Count. */

export type ModuleOriginType = 'local' | 'registry';
export type ModuleSyncOriginType = ModuleOriginType | 'all';

export type RequestSyncParams = {
  originType?: ModuleSyncOriginType;
  force?: boolean;
  ifStale?: boolean;
};

export type ModuleIndexRecord = {
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

export function normalizeSearchCondition(condition: any[] | Record<string, any>): any {
  const emptyArray = Array.isArray(condition) && condition.length === 0;
  const emptyObject = !Array.isArray(condition) && condition && typeof condition === 'object' && Object.keys(condition).length === 0;
  return emptyArray || emptyObject ? (['Available', '=', true] as any) : condition;
}

type SortSpec = { field: string; desc: boolean };
type GroupSortSpec = { field: string; order: 'asc' | 'desc' };

export function toText(value: unknown): string {
  return String(value ?? '')
    .trim()
    .toLowerCase();
}

export function toComparableValue(value: unknown): unknown {
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

export function parseSortSpecs(orderBy: any): SortSpec[] {
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

export function compareBySpecs(a: ModuleIndexRecord, b: ModuleIndexRecord, specs: SortSpec[]): number {
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

export function applySoftDeleteOptions(target: Record<string, unknown>, source: Record<string, unknown>): void {
  if (Object.prototype.hasOwnProperty.call(source, 'withDeleted')) {
    target.withDeleted = !!(source as any).withDeleted;
  }
  if (Object.prototype.hasOwnProperty.call(source, 'onlyDeleted')) {
    target.onlyDeleted = !!(source as any).onlyDeleted;
  }
}

export function buildSortPushdownPlan(sortSpecs: SortSpec[]): {
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

export function extractGroupedModuleNames(rows: any[]): string[] {
  const out: string[] = [];
  for (const row of rows || []) {
    const moduleName = String((row as any)?.ModuleName ?? (row as any)?.module_name ?? '').trim();
    if (!moduleName) continue;
    out.push(moduleName);
  }
  return out;
}

export function buildModuleNamesCondition(baseCondition: any, moduleNames: string[]): any {
  if (!Array.isArray(moduleNames) || moduleNames.length === 0) {
    return ['Id', '=', '__never_match__'] as any;
  }
  const inCondition = ['ModuleName', 'in', moduleNames] as any;
  const isEmptyArray = Array.isArray(baseCondition) && baseCondition.length === 0;
  const isEmptyObject = !!baseCondition && !Array.isArray(baseCondition) && typeof baseCondition === 'object' && Object.keys(baseCondition).length === 0;
  if (!baseCondition || isEmptyArray || isEmptyObject) {
    return inCondition;
  }
  return {
    And: [baseCondition as any, inCondition],
  } as any;
}

export function projectFields(rows: ModuleIndexRecord[], requestedFields: string[]): ModuleIndexRecord[] {
  if (!Array.isArray(requestedFields) || requestedFields.length === 0) return rows;
  const blockedFields = new Set(['__proto__', 'constructor', 'prototype']);
  const fields = Array.from(new Set(requestedFields.map(field => String(field || '').trim()).filter(field => !!field && !blockedFields.has(field))));
  if (fields.length === 0) return rows;

  return rows.map(row => {
    const projected = {} as ModuleIndexRecord;
    for (const field of fields) {
      (projected as any)[field] = (row as any)?.[field];
    }
    return projected;
  });
}

export function toPlainRecord(input: any): ModuleIndexRecord {
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

export function pickNewestTimestamp(values: Array<Date | string | null | undefined>): Date | string | null | undefined {
  let picked: Date | string | null | undefined;
  let pickedTs = -Infinity;
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

export function aggregateRows(rows: ModuleIndexRecord[]): ModuleIndexRecord[] {
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

    // Only local|registry remain after the filter; put local first.
    const originTypes = Array.from(new Set(bucket.map(row => toText(row?.OriginType)).filter(value => value === 'local' || value === 'registry'))).sort(
      (a, b) => Number(b === 'local') - Number(a === 'local')
    );

    const localVersion = String(local?.Version || '').trim();
    const registryVersion = String(registry?.Version || '').trim();
    const installedStatus =
      String(local?.InstalledStatus || '').trim() || String(registry?.InstalledStatus || '').trim() || String(base?.InstalledStatus || '').trim();
    const installedVersion =
      String(local?.InstalledVersion || '').trim() || String(registry?.InstalledVersion || '').trim() || String(base?.InstalledVersion || '').trim();
    const rawOriginType = String(base.OriginType ?? '').trim();

    merged.push({
      ...base,
      Id: String(base?.Id || '').trim() || String(local?.Id || '').trim() || String(registry?.Id || '').trim(),
      ModuleName: moduleName,
      OriginType: rawOriginType || 'local',
      OriginTypes: originTypes.length > 0 ? originTypes.join(', ') : rawOriginType,
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

export function normalizeOriginType(value?: string): ModuleSyncOriginType | '' {
  const raw = String(value || '')
    .trim()
    .toLowerCase();
  if (raw === 'all') return 'all';
  if (raw === '') return 'all';
  if (raw === 'local') return 'local';
  if (raw === 'registry') return 'registry';
  return '';
}

export function canReuseRunningSync(requested: ModuleSyncOriginType, running: ModuleSyncOriginType): boolean {
  if (requested === 'all') return running === 'all';
  if (running === 'all') return true;
  return running === requested;
}
