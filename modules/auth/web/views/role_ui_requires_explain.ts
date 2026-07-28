// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Normalize IrUiResource.Requires for RoleForm explain panel (UI-Option-A).
 */
export function normalizeUiResourceRequires(raw: unknown): string[] {
  if (raw == null) return [];
  let value: unknown = raw;
  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (!trimmed) return [];
    try {
      value = JSON.parse(trimmed);
    } catch {
      return [trimmed];
    }
  }
  if (!Array.isArray(value)) return [];
  const out: string[] = [];
  const seen = new Set<string>();
  for (const item of value) {
    const s = String(item ?? '').trim();
    if (!s || seen.has(s)) continue;
    seen.add(s);
    out.push(s);
  }
  return out;
}

/**
 * Choose the UI resource row to inspect (null clears the panel).
 */
export function selectInspectedUiResource(row: unknown): Record<string, any> | null {
  if (!row || typeof row !== 'object') return null;
  return row as Record<string, any>;
}

/**
 * Stable id string for the inspected UI resource row.
 */
export function getInspectedUiResourceId(row: Record<string, any> | null | undefined): string {
  return String(row?.Id ?? '').trim();
}

/**
 * Requires list for the inspected UI resource (Supports Requires / requires).
 */
export function getInspectedUiResourceRequires(row: Record<string, any> | null | undefined): string[] {
  return normalizeUiResourceRequires(row?.Requires ?? row?.requires);
}
