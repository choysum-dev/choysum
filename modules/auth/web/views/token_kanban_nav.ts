// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve a navigable token record id from a kanban card payload.
 * Ignores Vue-key fallbacks (row key / array index) that are not record ids.
 */
export function resolveTokenDetailId(payload: Record<string, unknown> | null | undefined): string {
  return String(payload?.Id ?? '').trim();
}

/**
 * Stable card id for ChoyKanbanView. Prefers record Id, then row key, then a
 * lane-scoped synthetic id so indexes do not collide across lanes.
 */
export function resolveTokenKanbanCardId(
  row: { payload?: Record<string, unknown>; key?: string },
  index: number,
  laneKey: string,
): string {
  const payload = (row.payload ?? {}) as Record<string, unknown>;
  const id = payload.Id ?? row.key;
  if (id != null && String(id).trim() !== '') {
    return String(id);
  }
  return `${laneKey}-${index}`;
}

/**
 * Resolve the record id for a drag-move write. Synthetic Vue keys must not
 * drive UpdateById.
 */
export function resolveTokenMoveRecordId(
  cards: ReadonlyArray<{ id: string; payload?: Record<string, unknown> }>,
  moveCardId: string,
): string {
  const card = cards.find(c => c.id === moveCardId);
  return resolveTokenDetailId(card?.payload);
}
