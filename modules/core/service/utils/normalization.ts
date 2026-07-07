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
  const raw = typeof value === 'object' && value !== null ? ((value as any).Id ?? (value as any).id ?? null) : value;
  const s = String(raw ?? '').trim();
  return s ? s : null;
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
