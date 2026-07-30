// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared helpers for list inline edit (S2) and handle / Sequence reorder.
 */

import { collectChangedPaths } from '@/core/utils/diff';
import { deepClonePreserve as deepClone } from '@/core/utils/clone';
import type { WebModelStore, WebFieldMetadata } from '@/web/web/stores/modelStore';

const getByPath = (obj: any, path: string) =>
  String(path)
    .split('.')
    .filter(Boolean)
    .reduce((a, k) => (a == null ? a : a[k]), obj);

/** Unwrap list / grouped row wrappers to the business record payload. */
export function unwrapListRecord(row: any): any {
  if (!row) return row;
  if (row.kind === 'record' && row.payload) return row.payload;
  if (row.type === 'record' && row.record) return row.record;
  if (row.payload && typeof row.payload === 'object') return row.payload;
  return row;
}

/** Whether a visible list node is an editable record row (not group / load-more). */
export function isListRecordRow(row: any): boolean {
  if (!row || typeof row !== 'object') return false;
  if (row.kind === 'group' || row.kind === 'more') return false;
  if (row.kind === 'record' || row.type === 'record') return true;
  const rec = unwrapListRecord(row);
  return rec != null && (rec.Id != null || rec.id != null);
}

/** Deep-enough clone for a row editing draft. */
export function cloneRowDraft<T extends Record<string, any>>(record: T): T {
  return deepClone(record);
}

/** Record id as string, or empty when missing. */
export function listRecordId(rowOrRecord: any): string {
  const rec = unwrapListRecord(rowOrRecord);
  const id = rec?.Id ?? rec?.id;
  return id == null ? '' : String(id);
}

/** True when metadata exposes a numeric handle field suitable for drag reorder. */
export function isNumericHandleField(meta: WebFieldMetadata | undefined): boolean {
  if (!meta) return false;
  const t = String(meta.type || '').toLowerCase();
  return t === 'int' || t === 'integer' || t === 'number' || t === 'decimal' || t === 'float';
}

/** Whether the store metadata contains a writable numeric handle field. */
export function hasHandleField(store: WebModelStore<any> | undefined, handleField = 'Sequence'): boolean {
  if (!store || !handleField) return false;
  const meta = (store.fieldsMetadata || {})[handleField] as WebFieldMetadata | undefined;
  if (!meta) return false;
  if (meta.isReadonly === true) return false;
  return isNumericHandleField(meta);
}

/**
 * Assign handleField = start + index (default start=1 → 1..n) for each row in order.
 * Pass `start = pagination.offset + 1` on paginated lists so page-local reorder
 * does not collide with sequences on other pages.
 * Returns rows whose sequence value changed.
 */
export function renumberSequence<T extends Record<string, any>>(
  rows: T[],
  handleField = 'Sequence',
  start = 1
): { row: T; previous: number | undefined; next: number }[] {
  const base = Number.isFinite(start) && start >= 1 ? Math.floor(start) : 1;
  const changed: { row: T; previous: number | undefined; next: number }[] = [];
  rows.forEach((row, index) => {
    const next = base + index;
    const previous = row[handleField] as number | undefined;
    if (previous !== next) {
      (row as Record<string, any>)[handleField] = next;
      changed.push({ row, previous, next });
    }
  });
  return changed;
}

/** Build a top-level Update payload from original vs draft, skipping readonly metadata fields. */
export function collectRowDirtyPayload(
  original: Record<string, any>,
  draft: Record<string, any>,
  fieldsMetadata?: Record<string, WebFieldMetadata | undefined>
): Record<string, any> {
  const paths = collectChangedPaths(original, draft, {
    includeTopLevel: true,
    includeFullPath: false,
    pruneRelationChildren: true,
    collapseFinal: true,
  });

  const payload: Record<string, any> = {};
  for (const path of paths) {
    if (!path || path.includes('.')) continue;
    const meta = fieldsMetadata?.[path];
    if (meta?.isReadonly === true) continue;
    payload[path] = draft[path];
  }
  return payload;
}

/** Swap a list row wrapper's payload with the active draft when ids match. */
export function withEditingPayload(row: any, editingRowId: string | null, draft: any): any {
  if (!editingRowId || !draft) return row;
  const id = listRecordId(row);
  if (!id || id !== editingRowId) return row;
  if (row?.kind === 'record') return { ...row, payload: draft };
  if (row?.type === 'record') return { ...row, record: draft };
  return draft;
}

/** Whether original and draft differ on any top-level writable field. */
export function isRowDraftDirty(
  original: Record<string, any> | null | undefined,
  draft: Record<string, any> | null | undefined,
  fieldsMetadata?: Record<string, WebFieldMetadata | undefined>
): boolean {
  if (!original || !draft) return false;
  return Object.keys(collectRowDirtyPayload(original, draft, fieldsMetadata)).length > 0;
}

/** Read field from draft using dotted path (for form-root helpers). */
export function getDraftField(draft: any, path: string): any {
  return getByPath(draft, path);
}

/** Write field on draft using dotted path (for form-root helpers). */
export function setDraftField(draft: any, path: string, value: any): void {
  if (!draft || typeof draft !== 'object') return;
  const segs = String(path).split('.').filter(Boolean);
  let cur = draft;
  for (let i = 0; i < segs.length - 1; i++) {
    const k = segs[i]!;
    if (!cur[k] || typeof cur[k] !== 'object') cur[k] = {};
    cur = cur[k];
  }
  cur[segs[segs.length - 1]!] = value;
}
