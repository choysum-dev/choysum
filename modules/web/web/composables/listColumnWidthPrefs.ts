// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Per-user list / embed column width overrides (localStorage, v1).
 * Server sync is intentionally deferred; keep bucket ids stable for a future model.
 */

import { normalizeScopeKey } from '@/web/web/composables/search/scopeKey';

export const LIST_COLUMN_WIDTH_PREFS_PREFIX = 'choysum.web.listColumnWidths.v1';

/** Drag clamp — keep tracks usable and avoid runaway widths. */
export const LIST_COLUMN_RESIZE_MIN_PX = 40;
export const LIST_COLUMN_RESIZE_MAX_PX = 800;

export const LIST_COLUMN_WIDTH_WRITE_DEBOUNCE_MS = 400;

export type ListColumnWidthPrefsPayload = {
  columns: Record<string, number>;
  updatedAt?: number;
};

export type ListColumnWidthListBucket = {
  kind: 'list';
  application: string;
  modelName: string;
  scopePath: string;
};

export type ListColumnWidthEmbedBucket = {
  kind: 'embed';
  application: string;
  parentModel: string;
  fieldProp: string;
  relation: 'o2m' | 'm2m';
};

export type ListColumnWidthBucket = ListColumnWidthListBucket | ListColumnWidthEmbedBucket;

/** Normalize login id for storage; empty → anon. */
export function normalizeListColumnWidthUserId(userId: unknown): string {
  const s = String(userId ?? '').trim();
  return s || 'anon';
}

/** Prefer identity.userId, then currentUser.Id. */
export function resolveListColumnPrefsUserId(auth: {
  identity?: { userId?: unknown } | null;
  currentUser?: { Id?: unknown } | null;
} | null | undefined): string {
  const fromIdentity = auth?.identity?.userId;
  if (fromIdentity != null && String(fromIdentity).trim()) {
    return normalizeListColumnWidthUserId(fromIdentity);
  }
  return normalizeListColumnWidthUserId(auth?.currentUser?.Id);
}

/** localStorage key for one user (all buckets live under this JSON map). */
export function listColumnWidthStorageKey(userId: unknown): string {
  return `${LIST_COLUMN_WIDTH_PREFS_PREFIX}.${normalizeListColumnWidthUserId(userId)}`;
}

/** Stable bucket id used inside the per-user JSON map. */
export function listColumnWidthBucketId(bucket: ListColumnWidthBucket): string {
  if (bucket.kind === 'list') {
    const app = String(bucket.application || '').trim() || '_';
    const model = String(bucket.modelName || '').trim() || '_';
    const scope = normalizeScopeKey(bucket.scopePath) || '/';
    return `list|${app}|${model}|${scope}`;
  }
  const app = String(bucket.application || '').trim() || '_';
  const parent = String(bucket.parentModel || '').trim() || '_';
  const prop = String(bucket.fieldProp || '').trim() || '_';
  const rel = bucket.relation === 'm2m' ? 'm2m' : 'o2m';
  return `embed|${app}|${parent}|${prop}|${rel}`;
}

export function clampListColumnWidthPx(px: unknown): number | null {
  const n = typeof px === 'number' ? px : Number(px);
  if (!Number.isFinite(n) || n <= 0) return null;
  return Math.min(
    LIST_COLUMN_RESIZE_MAX_PX,
    Math.max(LIST_COLUMN_RESIZE_MIN_PX, Math.round(n)),
  );
}

function emptyPayload(): ListColumnWidthPrefsPayload {
  return { columns: {} };
}

function sanitizePayload(raw: unknown): ListColumnWidthPrefsPayload {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return emptyPayload();
  const colsIn = (raw as { columns?: unknown }).columns;
  if (!colsIn || typeof colsIn !== 'object' || Array.isArray(colsIn)) return emptyPayload();
  const columns: Record<string, number> = {};
  for (const [k, v] of Object.entries(colsIn as Record<string, unknown>)) {
    const key = String(k || '').trim();
    if (!key) continue;
    const clamped = clampListColumnWidthPx(v);
    if (clamped != null) columns[key] = clamped;
  }
  const updatedAtRaw = (raw as { updatedAt?: unknown }).updatedAt;
  const updatedAt =
    typeof updatedAtRaw === 'number' && Number.isFinite(updatedAtRaw) ? updatedAtRaw : undefined;
  return updatedAt != null ? { columns, updatedAt } : { columns };
}

type UserStoreMap = Record<string, ListColumnWidthPrefsPayload>;

function readUserStore(
  storage: Pick<Storage, 'getItem'> | null | undefined,
  userId: unknown,
): UserStoreMap {
  try {
    if (!storage) return {};
    const raw = storage.getItem(listColumnWidthStorageKey(userId));
    if (!raw) return {};
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {};
    const out: UserStoreMap = {};
    for (const [bucketId, payload] of Object.entries(parsed as Record<string, unknown>)) {
      const id = String(bucketId || '').trim();
      if (!id) continue;
      out[id] = sanitizePayload(payload);
    }
    return out;
  } catch {
    return {};
  }
}

function writeUserStore(
  storage: Pick<Storage, 'setItem'> | null | undefined,
  userId: unknown,
  map: UserStoreMap,
): void {
  try {
    if (!storage) return;
    storage.setItem(listColumnWidthStorageKey(userId), JSON.stringify(map));
  } catch {
    // Denied / quota — ignore (same posture as theme prefs).
  }
}

function resolveStorage(
  storage?: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> | null,
): Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> | null {
  if (storage === undefined) {
    try {
      return typeof localStorage !== 'undefined' ? localStorage : null;
    } catch {
      return null;
    }
  }
  return storage;
}

/** Read width overrides for one List / embed bucket. */
export function readListColumnWidthPrefs(
  bucket: ListColumnWidthBucket,
  userId: unknown,
  storage?: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> | null,
): ListColumnWidthPrefsPayload {
  const store = resolveStorage(storage);
  const map = readUserStore(store, userId);
  const id = listColumnWidthBucketId(bucket);
  return map[id] ? sanitizePayload(map[id]) : emptyPayload();
}

/** Replace all overrides for a bucket (empty columns deletes the bucket entry). */
export function writeListColumnWidthPrefs(
  bucket: ListColumnWidthBucket,
  userId: unknown,
  payload: ListColumnWidthPrefsPayload,
  storage?: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> | null,
): void {
  const store = resolveStorage(storage);
  if (!store) return;
  const map = readUserStore(store, userId);
  const id = listColumnWidthBucketId(bucket);
  const next = sanitizePayload(payload);
  if (!Object.keys(next.columns).length) {
    delete map[id];
  } else {
    map[id] = {
      columns: next.columns,
      updatedAt: typeof next.updatedAt === 'number' ? next.updatedAt : Date.now(),
    };
  }
  writeUserStore(store, userId, map);
}

/** Merge one column width into the bucket (clamped). */
export function upsertListColumnWidthPref(
  bucket: ListColumnWidthBucket,
  userId: unknown,
  fieldKey: string,
  widthPx: number,
  storage?: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> | null,
): ListColumnWidthPrefsPayload {
  const key = String(fieldKey || '').trim();
  const clamped = clampListColumnWidthPx(widthPx);
  if (!key || clamped == null) {
    return readListColumnWidthPrefs(bucket, userId, storage);
  }
  const current = readListColumnWidthPrefs(bucket, userId, storage);
  const columns = { ...current.columns, [key]: clamped };
  const next = { columns, updatedAt: Date.now() };
  writeListColumnWidthPrefs(bucket, userId, next, storage);
  return next;
}

/** Drop one column override (or clear the whole bucket when fieldKey omitted). */
export function clearListColumnWidthPrefs(
  bucket: ListColumnWidthBucket,
  userId: unknown,
  fieldKey?: string,
  storage?: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> | null,
): ListColumnWidthPrefsPayload {
  const current = readListColumnWidthPrefs(bucket, userId, storage);
  if (!fieldKey) {
    writeListColumnWidthPrefs(bucket, userId, emptyPayload(), storage);
    return emptyPayload();
  }
  const key = String(fieldKey).trim();
  if (!key || !(key in current.columns)) return current;
  const columns = { ...current.columns };
  delete columns[key];
  const next = { columns, updatedAt: Date.now() };
  writeListColumnWidthPrefs(bucket, userId, next, storage);
  return sanitizePayload(next);
}

/** Merge helper for adapter: apply user px over auto size when author did not fix width. */
export function applyUserColumnWidthOverride(
  autoSize: number,
  userWidth: number | undefined | null,
): number {
  const clamped = clampListColumnWidthPx(userWidth);
  return clamped != null ? clamped : autoSize;
}
