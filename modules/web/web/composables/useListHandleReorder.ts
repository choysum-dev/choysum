// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * HTML5 drag-and-drop reorder for list / O2M handle columns.
 */

import { ref, type Ref } from 'vue';
import { renumberSequence } from '@/web/web/composables/listRowEdit';

export const LIST_HANDLE_API_KEY = Symbol('list-handle-api');

/** data-testid values that identify a drag-image source row/card. */
const LIST_HANDLE_DRAG_IMAGE_TEST_IDS = new Set([
  'choy-o2m-kanban-card',
  'choy-properties-definition-row',
  'choy-properties-definition-selection-row',
]);

/**
 * Snapshot `target` as the HTML5 drag ghost (Kanban / list handle shared offset math).
 * No-ops when dataTransfer is missing; never throws.
 */
export function setHtml5DragImage(target: Element, event: DragEvent): void {
  const dt = event.dataTransfer;
  if (!dt || !(target instanceof HTMLElement)) return;
  try {
    const rect = target.getBoundingClientRect();
    const offsetX = Math.max(0, Math.min(rect.width, event.clientX - rect.left));
    const offsetY = Math.max(0, Math.min(rect.height, event.clientY - rect.top));
    dt.setDragImage(target, offsetX, offsetY);
  } catch {
    /* ignore */
  }
}

/** True for list/kanban/definition row containers used as drag ghosts. */
export function isListHandleDragImageSource(el: Element): boolean {
  if (el.getAttribute('role') === 'row') return true;
  const testId = el.getAttribute('data-testid');
  return !!testId && LIST_HANDLE_DRAG_IMAGE_TEST_IDS.has(testId);
}

/**
 * Walk ancestors from the drag handle to the nearest row/card and set the drag image.
 * Avoids compound CSS selectors so minimal/test DOMs without full `closest` still work.
 */
export function setListHandleDragImage(event: DragEvent): void {
  const start = event.currentTarget;
  if (!(start instanceof Element)) return;
  let node: Element | null = start;
  while (node) {
    if (isListHandleDragImageSource(node)) {
      setHtml5DragImage(node, event);
      return;
    }
    node = node.parentElement;
  }
}

export type ListHandleReorderApi = {
  enabled: Ref<boolean>;
  draggingIndex: Ref<number | null>;
  /** Row currently hovered as the drop target while dragging (null when idle). */
  overIndex: Ref<number | null>;
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
  const overIndex = ref<number | null>(null);
  const recordOf = (row: T) => opts.getRecord?.(row) ?? (row as Record<string, any>);

  function clearDragState() {
    draggingIndex.value = null;
    overIndex.value = null;
  }

  function onDragStart(index: number, event: DragEvent) {
    if (!opts.enabled.value) {
      event.preventDefault();
      return;
    }
    draggingIndex.value = index;
    overIndex.value = index;
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
    overIndex.value = index;
    try {
      event.dataTransfer!.dropEffect = 'move';
    } catch {
      /* ignore */
    }
  }

  async function onDrop(index: number, event: DragEvent) {
    event.preventDefault();
    if (!opts.enabled.value) {
      clearDragState();
      return;
    }
    const from = draggingIndex.value;
    clearDragState();
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
    clearDragState();
  }

  return {
    enabled: opts.enabled,
    draggingIndex,
    overIndex,
    onDragStart,
    onDragOver,
    onDrop,
    onDragEnd,
  };
}
