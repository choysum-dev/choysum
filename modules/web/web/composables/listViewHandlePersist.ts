// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure helpers for OListView handle reorder persistence and inline-edit discard.
 */

import { isListRecordRow, listRecordId, unwrapListRecord } from '@/web/web/composables/listRowEdit';

export type HandleReorderWrite = { id: string; previous: number | undefined; next: number };

/** Map renumberSequence output to UpdateById writes (skip rows without id). */
export function buildHandleReorderWrites(
  changed: { row: any; previous: number | undefined; next: number }[]
): HandleReorderWrite[] {
  return changed
    .map(({ row, previous, next }) => {
      const rec = unwrapListRecord(row);
      const id = listRecordId(rec);
      if (!id) return null;
      return { id, previous, next };
    })
    .filter(Boolean) as HandleReorderWrite[];
}

/** Persist sequence writes with rollback + refresh on failure. */
export async function persistHandleReorder(opts: {
  writes: HandleReorderWrite[];
  handleField: string;
  updateById: (id: string, payload: Record<string, any>) => Promise<void>;
  refresh: () => Promise<void>;
  rollbackFlat: () => void;
  onError: () => void;
}): Promise<void> {
  const { writes, handleField, updateById, refresh, rollbackFlat, onError } = opts;
  if (!writes.length) return;
  try {
    for (const w of writes) {
      await updateById(w.id, { [handleField]: w.next });
    }
    await refresh();
  } catch {
    for (const w of writes) {
      if (w.previous === undefined) continue;
      try {
        await updateById(w.id, { [handleField]: w.previous });
      } catch {
        /* ignore rollback errors; refresh will reconcile */
      }
    }
    rollbackFlat();
    onError();
    try {
      await refresh();
    } catch {
      /* ignore */
    }
  }
}

/** Whether items watcher should discard active inline edit (row left the visible set). */
export function shouldDiscardInvisibleEdit(
  isEditing: boolean,
  editingId: string | null | undefined,
  items: any[],
  isDirty: boolean
): { discard: boolean; warn: boolean } {
  if (!isEditing || !editingId) return { discard: false, warn: false };
  const stillVisible = (items || []).some(row => isListRecordRow(row) && listRecordId(row) === editingId);
  if (stillVisible) return { discard: false, warn: false };
  return { discard: true, warn: isDirty };
}

/** Normalize controller visible nodes into flat record rows for handle reorder. */
export function syncFlatRowsFromVisibleItems(items: any[], isGroupMode: boolean): any[] {
  if (isGroupMode) return [];
  return (items || [])
    .filter(isListRecordRow)
    .map(row => ({
      ...row,
      payload: { ...unwrapListRecord(row) },
    }));
}
