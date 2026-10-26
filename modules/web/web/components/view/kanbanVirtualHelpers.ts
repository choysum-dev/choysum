// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Min loaded cards before auto virtualization kicks in. */
export const CHOY_KANBAN_VIRTUAL_THRESHOLD = 40;

/**
 * Default estimated card height (px), content only — not inter-card gap.
 * Gap is applied separately via TanStack Virtual `gap` (see CHOY_KANBAN_CARD_GAP).
 */
export const CHOY_KANBAN_DEFAULT_CARD_SIZE = 88;

/** Inter-card gap for virtualized lanes; matches non-virtual `gap-2` (8px). */
export const CHOY_KANBAN_CARD_GAP = 8;

/** MIME type for HTML5 lane-drop card payloads. */
export const CHOY_KANBAN_LANE_DRAG_MIME = 'application/x-choy-kanban-card';

export type ChoyKanbanVirtualizeMode = boolean | 'auto';

export type ChoyKanbanDragMode = 'none' | 'lane';

export type ChoyKanbanLaneDragPayload = {
  cardId: string;
  fromLaneKey: string;
};

/** Whether the lane body should mount TanStack Virtual for `cardCount` cards. */
export function choyKanbanShouldVirtualize(
  virtualize: ChoyKanbanVirtualizeMode | undefined,
  cardCount: number,
): boolean {
  const n = Math.max(0, Math.floor(Number(cardCount) || 0));
  if (virtualize === false) return false;
  if (virtualize === true) return n > 0;
  return n >= CHOY_KANBAN_VIRTUAL_THRESHOLD;
}

export function resolveChoyKanbanEstimateCardSize(raw: unknown): number {
  const n = Number(raw);
  if (!Number.isFinite(n) || n <= 0) return CHOY_KANBAN_DEFAULT_CARD_SIZE;
  return Math.floor(n);
}

/** Serialize a lane-drop drag payload (empty string when invalid). */
export function encodeChoyKanbanLaneDragPayload(
  payload: ChoyKanbanLaneDragPayload | null | undefined,
): string {
  const cardId = String(payload?.cardId ?? '').trim();
  const fromLaneKey = String(payload?.fromLaneKey ?? '').trim();
  if (!cardId || !fromLaneKey) return '';
  return JSON.stringify({ cardId, fromLaneKey });
}

/** Parse a lane-drop drag payload (null when invalid). */
export function decodeChoyKanbanLaneDragPayload(raw: unknown): ChoyKanbanLaneDragPayload | null {
  if (typeof raw !== 'string' || !raw.trim()) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<ChoyKanbanLaneDragPayload>;
    const cardId = String(parsed?.cardId ?? '').trim();
    const fromLaneKey = String(parsed?.fromLaneKey ?? '').trim();
    if (!cardId || !fromLaneKey) return null;
    return { cardId, fromLaneKey };
  } catch {
    return null;
  }
}

/**
 * Build a cross-lane move from a drop on a whole lane (append to end).
 * Returns null for same-lane drops or missing payload fields.
 */
export function resolveChoyKanbanLaneDropMove(opts: {
  payload: ChoyKanbanLaneDragPayload | null | undefined;
  toLaneKey: string;
  toLaneCardCount: number;
}): { cardId: string; fromLaneKey: string; toLaneKey: string; toIndex: number } | null {
  const cardId = String(opts.payload?.cardId ?? '').trim();
  const fromLaneKey = String(opts.payload?.fromLaneKey ?? '').trim();
  const toLaneKey = String(opts.toLaneKey ?? '').trim();
  if (!cardId || !fromLaneKey || !toLaneKey || fromLaneKey === toLaneKey) return null;
  const toIndex = Math.max(0, Math.floor(Number(opts.toLaneCardCount) || 0));
  return { cardId, fromLaneKey, toLaneKey, toIndex };
}
