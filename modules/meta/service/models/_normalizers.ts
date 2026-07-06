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
