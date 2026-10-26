// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isRef, unref } from 'vue';
import type { QueryCondition } from '@/core/rpc';
import { resetRelationPickerQueryState } from '@/web/web/query/state';

/** True for a leaf triple like `['Id', '=', 'x']` (not a list of conditions). */
function isQueryConditionLeaf(v: unknown): boolean {
  return Array.isArray(v) && v.length >= 3 && typeof v[0] === 'string' && typeof v[1] === 'string';
}

/**
 * Normalize a single condition or list of conditions.
 * Leaf triples are arrays, so they must not be treated as `QueryCondition[]`.
 */
export function toConditionArray(
  v: QueryCondition<any> | QueryCondition<any>[] | undefined | null
): QueryCondition<any>[] {
  if (v == null) return [];
  if (isQueryConditionLeaf(v)) return [v as QueryCondition<any>];
  if (Array.isArray(v)) return v as QueryCondition<any>[];
  return [v as QueryCondition<any>];
}

/** Build `Id not in selected` when the picker already has selected rows. */
export function buildExcludePickedCondition(selectedIds: readonly string[]): QueryCondition<any> | undefined {
  const ids = Array.from(new Set((selectedIds || []).map(id => String(id || '').trim()).filter(Boolean)));
  if (!ids.length) return undefined;
  return ['Id', 'not in', ids] as unknown as QueryCondition<any>;
}

/**
 * Merge exclude-picked + external + onchange parts into a single QueryCondition (or []).
 */
export function buildM2MEffectiveCondition(parts: {
  selectedIds?: readonly string[];
  external?: QueryCondition<any> | QueryCondition<any>[] | null;
  onchangeParts?: QueryCondition<any>[];
}): QueryCondition<any> | [] {
  const merged: QueryCondition<any>[] = [];
  const exclude = buildExcludePickedCondition(parts.selectedIds || []);
  if (exclude) merged.push(exclude);
  merged.push(...toConditionArray(parts.external as any));
  if (Array.isArray(parts.onchangeParts)) {
    for (const c of parts.onchangeParts) {
      if (c) merged.push(c);
    }
  }
  if (merged.length === 0) return [] as any;
  if (merged.length === 1) return merged[0];
  return { And: merged } as any;
}

const DEFAULT_LABEL_FIELDS = ['DisplayName', 'Name', 'Title', 'Code', 'Id'] as const;

/** Resolve a chip / option label from a relation row. */
export function resolveRelationRecordLabel(
  row: any,
  fallback?: string,
  labelFields: readonly string[] = DEFAULT_LABEL_FIELDS
): string {
  if (!row || typeof row !== 'object') return fallback || '';
  for (const key of labelFields) {
    const value = (row as any)?.[key];
    if (value != null && String(value).trim()) return String(value);
  }
  return fallback || String((row as any)?.Id ?? '');
}

/** Clear leftover List query UI before opening an M2M Search more / Add row picker. */
export function resetM2MPickerQueryState(store: { state?: { queryState?: unknown } } | null | undefined): void {
  resetRelationPickerQueryState(store as any);
}

/** Peel Ref / `{ value: T[] }` layers from ListView / useListViewExpose exposes. */
function unwrapRefLayers(picked: unknown): unknown {
  let raw: any = picked;
  for (let i = 0; i < 4; i++) {
    if (isRef(raw)) {
      raw = unref(raw);
      continue;
    }
    // Mount-test stubs and some expose shapes: `{ value: T[] }`.
    if (raw && typeof raw === 'object' && !Array.isArray(raw) && Array.isArray((raw as any).value)) {
      raw = (raw as any).value;
      continue;
    }
    break;
  }
  return raw;
}

/** Unwrap ListView selection rows (ref layers + record wrappers → plain objects). */
export function unwrapListSelectionRecords(picked: unknown): any[] {
  const raw = unwrapRefLayers(picked);
  const toRecord = (x: any) =>
    x && typeof x === 'object' && x.kind === 'record' && x.payload
      ? x.payload
      : x && typeof x === 'object' && x.type === 'record' && x.record
        ? x.record
        : x;
  return Array.isArray(raw) ? raw.map(toRecord) : [];
}
