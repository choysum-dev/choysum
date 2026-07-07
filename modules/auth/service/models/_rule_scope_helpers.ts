// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Normalize a relation reference into a trimmed Id string.
 *
 * Accepts a plain string id, an object with an Id (or id) property, or null/undefined.
 * Returns null when the input cannot be resolved to a non-empty string.
 */
export function normalizeRefId(v: any): string | null {
  if (v == null) return null;
  const raw = typeof v === 'object' && v !== null ? ((v as any).Id ?? (v as any).id ?? null) : v;
  const s = String(raw ?? '').trim();
  return s ? s : null;
}
