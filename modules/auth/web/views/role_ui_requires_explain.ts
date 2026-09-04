// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

function invalidUiResourceRequires(): never {
  throw new Error('invalid_ui_resource_requires');
}

/**
 * Normalize MetaUiResource.Requires for the RoleForm explain panel.
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

  if (!Array.isArray(value)) {
    invalidUiResourceRequires();
  }

  const out: string[] = [];
  const seen = new Set<string>();
  for (const item of value) {
    if (typeof item !== 'string') {
      invalidUiResourceRequires();
    }
    const s = item.trim();
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

/**
 * Whether the tree node matches the currently inspected UI resource id.
 */
export function isInspectedUiResourceRow(inspectedId: string, row: unknown): boolean {
  const selected = String(inspectedId ?? '').trim();
  if (!selected) return false;
  if (!row || typeof row !== 'object') return false;
  const id = String((row as Record<string, any>).Id ?? '').trim();
  return id !== '' && id === selected;
}
