// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * HTML5 drag-and-drop reorder for list / O2M handle columns.
 */

import { ref, type Ref } from 'vue';
import { renumberSequence } from '@/web/web/composables/listRowEdit';

export const LIST_HANDLE_API_KEY = Symbol('list-handle-api');

export type ListHandleReorderApi = {
  enabled: Ref<boolean>;
  draggingIndex: Ref<number | null>;
  onDragStart: (index: number, event: DragEvent) => void;
  onDragOver: (index: number, event: DragEvent) => void;
  onDrop: (index: number, event: DragEvent) => void | Promise<void>;
  onDragEnd: () => void;
};

export function useListHandleReorder<T extends Record<string, any>>(opts: {
  rows: () => T[];
  enabled: Ref<boolean>;
  handleField?: string;
  /** First sequence number for the first visible row (default 1). Use offset+1 when paginated. */
  sequenceStart?: () => number;
  /** Map a table row to the object that receives Sequence writes (defaults to identity). */
  getRecord?: (row: T) => Record<string, any>;
  onReorder: (rows: T[], changed: ReturnType<typeof renumberSequence<Record<string, any>>>) => void | Promise<void>;
}): ListHandleReorderApi {
  const draggingIndex = ref<number | null>(null);
  const recordOf = (row: T) => opts.getRecord?.(row) ?? (row as Record<string, any>);

  function onDragStart(index: number, event: DragEvent) {
    if (!opts.enabled.value) {
      event.preventDefault();
      return;
    }
    draggingIndex.value = index;
    try {
      event.dataTransfer!.effectAllowed = 'move';
      event.dataTransfer!.setData('text/plain', String(index));
    } catch {
      /* ignore */
    }
  }

  function onDragOver(index: number, event: DragEvent) {
    if (!opts.enabled.value || draggingIndex.value == null) return;
    event.preventDefault();
    try {
      event.dataTransfer!.dropEffect = 'move';
    } catch {
      /* ignore */
    }
  }

  async function onDrop(index: number, event: DragEvent) {
    event.preventDefault();
    if (!opts.enabled.value) return;
    const from = draggingIndex.value;
    draggingIndex.value = null;
    if (from == null || from === index) return;

    const rows = opts.rows().slice();
    if (from < 0 || from >= rows.length || index < 0 || index >= rows.length) return;

    const [item] = rows.splice(from, 1);
    rows.splice(index, 0, item!);

    const handleField = opts.handleField ?? 'Sequence';
    const start = opts.sequenceStart?.() ?? 1;
    const payloads = rows.map(r => recordOf(r));
    const seqChanges = renumberSequence(payloads, handleField, start);
    const changed = seqChanges.map(({ row: rec, previous, next }) => {
      const idx = payloads.indexOf(rec);
      return { row: rows[idx]!, previous, next };
    });
    try {
      await opts.onReorder(rows, changed);
    } catch {
      /* Caller/UI surfaces errors; avoid unhandled rejection from void onDrop. */
    }
  }

  function onDragEnd() {
    draggingIndex.value = null;
  }

  return {
    enabled: opts.enabled,
    draggingIndex,
    onDragStart,
    onDragOver,
    onDrop,
    onDragEnd,
  };
}
