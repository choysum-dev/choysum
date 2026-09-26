// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Parse / stringify helpers for ChoyJsonField.
 */

export type ChoyJsonValue = Record<string, unknown> | unknown[] | null;

export type ChoyJsonParseResult =
  | { ok: true; value: ChoyJsonValue }
  | { ok: false; error: string };

/** Stable pretty print (object keys sorted). */
export function stringifyChoyJson(value: unknown, pretty = true): string {
  if (value == null) return '';
  try {
    if (Array.isArray(value)) {
      return pretty ? JSON.stringify(value, null, 2) : JSON.stringify(value);
    }
    if (typeof value === 'object') {
      const sorted = Object.keys(value as Record<string, unknown>).sort();
      return pretty
        ? JSON.stringify(value, sorted, 2)
        : JSON.stringify(value, sorted);
    }
    return String(value);
  } catch {
    return '';
  }
}

/** Coerce incoming model values (object / JSON string) to ChoyJsonValue. */
export function normalizeChoyJsonIncoming(raw: unknown): ChoyJsonValue {
  if (raw == null) return null;
  if (typeof raw === 'object') return raw as ChoyJsonValue;
  if (typeof raw === 'string') {
    const parsed = tryParseChoyJson(raw, { allowArray: true, nullable: true });
    return parsed.ok ? parsed.value : null;
  }
  return null;
}

export function tryParseChoyJson(
  raw: string,
  opts: { allowArray?: boolean; nullable?: boolean } = {},
): ChoyJsonParseResult {
  const allowArray = opts.allowArray === true;
  const nullable = opts.nullable !== false;
  const text = String(raw ?? '').trim();
  if (!text) {
    if (nullable) return { ok: true, value: null };
    return { ok: false, error: 'JSON is required' };
  }
  try {
    const v = JSON.parse(text) as unknown;
    if (v === null) {
      if (nullable) return { ok: true, value: null };
      return { ok: false, error: 'JSON is required' };
    }
    const isArr = Array.isArray(v);
    const isObj = typeof v === 'object' && !isArr;
    if (isArr && !allowArray) return { ok: false, error: 'Arrays are not allowed' };
    if (!isArr && !isObj) return { ok: false, error: 'Must be an object' };
    return { ok: true, value: v as ChoyJsonValue };
  } catch {
    return { ok: false, error: 'JSON parse failed' };
  }
}
