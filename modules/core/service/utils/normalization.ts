// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Return a trimmed string, or undefined when input is empty/null.
 */
export function normalizeOptionalString(value: unknown): string | undefined {
  const normalized = String(value || '').trim();
  return normalized || undefined;
}

/**
 * Normalize a possibly-mixed array into a deduplicated list of non-empty trimmed strings.
 */
export function normalizeStringArray(value: unknown): string[] {
  const arr = Array.isArray(value) ? value : [];
  return Array.from(new Set(arr.map(item => String(item || '').trim()).filter(Boolean)));
}

/**
 * Extract the identifier from either a plain string value or an object with
 * an `Id` property (e.g. a FK reference). Returns undefined for empty input.
 */
export function readRefId(value: unknown): string | undefined {
  if (!value) return undefined;
  if (typeof value === 'string') return normalizeOptionalString(value);
  if (typeof value === 'object') return normalizeOptionalString((value as any).Id);
  return undefined;
}

/**
 * Normalize a relation reference into a trimmed Id string.
 *
 * Accepts a plain string id, an object with an Id (or id) property, or null/undefined.
 * Returns null when the input cannot be resolved to a non-empty string.
 */
export function normalizeRefId(value: unknown): string | null {
  if (value == null) return null;
  const raw = typeof value === 'object' ? ((value as any).Id ?? (value as any).id ?? null) : value;
  const s = String(raw ?? '').trim();
  return s ? s : null;
}

/**
 * Normalize a mixed value or array into a deduplicated list of non-null Id strings.
 *
 * - Wraps a non-array, non-null input as a single-element array (singleton coercion).
 * - Extracts the Id from each element via {@link normalizeRefId}.
 * - Filters null results and deduplicates.
 */
export function normalizeRefIdList(value: unknown): string[] {
  if (value == null) return [];
  const arr = Array.isArray(value) ? value : [value];
  const set = new Set<string>();
  for (const it of arr) {
    const id = normalizeRefId(it);
    if (id) set.add(id);
  }
  return Array.from(set);
}

/**
 * Normalize an offset (non-negative finite integer, floor).
 */
export function normalizeOffset(raw: unknown): number {
  const value = Number(raw);
  if (!Number.isFinite(value) || value < 0) return 0;
  return Math.floor(value);
}

/**
 * Normalize a limit (positive finite integer, floor). Returns null for invalid/zero/negative.
 */
export function normalizeLimit(raw: unknown): number | null {
  const value = Number(raw);
  if (!Number.isFinite(value) || value <= 0) return null;
  return Math.floor(value);
}

/**
 * Normalize a field name list: trim, deduplicate, filter empty.
 */
export function normalizeFields(raw: unknown): string[] {
  if (!Array.isArray(raw)) return [];
  const seen = new Set<string>();
  for (const item of raw) {
    const field = String(item || '').trim();
    if (!field) continue;
    seen.add(field);
  }
  return Array.from(seen);
}

/**
 * Normalize a mixed list into unique non-empty trimmed strings.
 *
 * Accepts an array of values (strings, numbers, etc.) and returns a
 * deduplicated array of non-empty strings. Non-array input is treated
 * as an empty list.
 */
export function uniqStrings(xs: unknown): string[] {
  return Array.from(new Set((Array.isArray(xs) ? xs : []).map(v => String(v ?? '').trim()).filter(Boolean)));
}

/**
 * Normalize supported require keys into canonical rpc:/ form.
 */
export function normalizeRpcRequireKey(key: string): string {
  const k = String(key || '').trim();
  if (!k) return '';
  if (k.startsWith('rpc:/')) return k;
  if (k.startsWith('service:/')) return `rpc:/${k.slice('service:/'.length)}`;
  return '';
}

/**
 * Convert an rpc:/model/method key into rpc:/model/* wildcard.
 */
export function rpcServiceWildcard(key: string): string {
  const k = String(key || '').trim();
  if (!k.startsWith('rpc:/')) return '';
  if (k.endsWith('/*')) return k;
  const i = k.lastIndexOf('/');
  if (i <= 'rpc:/'.length) return '';
  return `${k.slice(0, i)}/*`;
}

/**
 * Return a stable sorted copy of a string array.
 */
export function sortStrings(xs: string[]): string[] {
  return (xs || []).slice().sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));
}

/**
 * Extract the identifier from either a plain string value or an object with
 * an `Id` (or `id`) property. Returns undefined for empty input.
 *
 * This is a relaxed variant of {@link readRefId} that also checks lowercase `id`.
 */
export function maybeRefId(value: unknown): string | undefined {
  if (!value) return undefined;
  if (typeof value === 'string') return normalizeOptionalString(value);
  if (typeof value === 'object')
    return normalizeOptionalString((value as Record<string, unknown>).Id) ?? normalizeOptionalString((value as Record<string, unknown>).id);
  return undefined;
}

function normalizeRefLikeIdString(raw: unknown): string {
  if (raw == null) return '';
  if (typeof raw === 'object') {
    return String((raw as Record<string, unknown>).Id ?? (raw as Record<string, unknown>).id ?? '').trim();
  }
  return String(raw ?? '').trim();
}

/**
 * Normalize an application or scope reference into its string id.
 *
 * Returns '' when the input cannot be resolved to a non-empty string (unlike
 * {@link normalizeRefId} which returns null for the same case).
 */
export function normalizeScopeRefId(raw: unknown): string {
  return normalizeRefLikeIdString(raw);
}

/**
 * Normalize a UI resource reference into its string id.
 */
export function normalizeUiResourceId(raw: unknown): string {
  return normalizeRefLikeIdString(raw);
}

/**
 * Parse a string or structured value into a normalized string array.
 *
 * Handles plain JSON arrays, objects with `value`/`values`/`items` keys,
 * numeric-indexed objects, and singleton string inputs.
 */
export function parseJsonStringArray(raw: unknown): string[] {
  const normalize = (xs: unknown): string[] => uniqStrings(xs);

  const tryObjectSnapshot = (value: unknown): string[] | null => {
    if (!value || typeof value !== 'object') return null;

    for (const key of ['value', 'values', 'items']) {
      try {
        const direct = (value as Record<string, unknown>)[key];
        if (Array.isArray(direct)) return normalize(direct);
      } catch {
        // fallthrough
      }
    }

    try {
      const numericKeys = Object.keys(value as Record<string, unknown>)
        .filter(key => /^\d+$/.test(key))
        .sort((a, b) => Number(a) - Number(b));
      if (numericKeys.length > 0) return normalize(numericKeys.map(key => (value as Record<string, unknown>)[key]));
    } catch {
      // fallthrough
    }

    try {
      const snap = JSON.parse(JSON.stringify(value));
      if (Array.isArray(snap)) return normalize(snap);
      if (!snap || typeof snap !== 'object') return null;

      for (const key of ['value', 'values', 'items']) {
        if (Array.isArray((snap as Record<string, unknown>)[key])) return normalize((snap as Record<string, unknown>)[key]);
      }

      const numericKeys = Object.keys(snap)
        .filter(key => /^\d+$/.test(key))
        .sort((a, b) => Number(a) - Number(b));
      if (numericKeys.length > 0) return normalize(numericKeys.map(key => (snap as Record<string, unknown>)[key]));
    } catch {
      // fallthrough
    }
    return null;
  };

  if (Array.isArray(raw)) return normalize(raw);
  if (raw == null) return [];

  if (typeof raw === 'string') {
    const s = raw.trim();
    if (!s) return [];
    try {
      const parsed = JSON.parse(s);
      if (Array.isArray(parsed)) return normalize(parsed);
    } catch {
      // fallthrough
    }
    return normalize([s]);
  }

  const snapResult = tryObjectSnapshot(raw);
  if (snapResult) return snapResult;

  try {
    if (typeof (raw as Record<string, unknown>)?.toString === 'function') {
      const s = String((raw as Record<string, unknown>).toString() || '').trim();
      if (!s) return [];
      const parsed = JSON.parse(s);
      if (Array.isArray(parsed)) return normalize(parsed);
    }
  } catch {
    // fallthrough
  }
  return [];
}

/**
 * Normalize a company id-like value to a trimmed string.
 */
export function normalizeScopeId(value: unknown): string {
  const id = maybeRefId(value);
  if (id) return String(id).trim();
  return String(value ?? '').trim();
}

/**
 * Return a stable unique copy of scope ids while preserving first-seen order.
 */
export function uniqScopeIds(ids: string[]): string[] {
  return Array.from(new Set((ids || []).map(v => normalizeScopeId(v)).filter(Boolean)));
}

/**
 * Normalize a dynamic preferences value into a plain JSON object.
 */
export function normalizePreferences(value: unknown): Record<string, unknown> {
  if (!value) return {};

  if (typeof value === 'string') {
    const s = value.trim();
    if (!s) return {};
    try {
      const parsed = JSON.parse(s);
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? (parsed as Record<string, unknown>) : {};
    } catch {
      return {};
    }
  }

  if (typeof value === 'object') {
    try {
      const snapshot = JSON.parse(JSON.stringify(value));
      if (snapshot && typeof snapshot === 'object' && !Array.isArray(snapshot)) return snapshot as Record<string, unknown>;
    } catch {
      // fallthrough
    }
    return value as Record<string, unknown>;
  }

  return {};
}

/**
 * Merge company scope preferences while preserving unrelated preference fields.
 */
export function buildScopePreferences(basePrefs: Record<string, unknown>, active: string, enabled: string[]): Record<string, unknown> {
  const base = basePrefs && typeof basePrefs === 'object' && !Array.isArray(basePrefs) ? basePrefs : {};
  return {
    ...base,
    activeCompanyId: active,
    enabledCompanyIds: enabled,
  };
}
