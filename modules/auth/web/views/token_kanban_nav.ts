// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  finishInitialKanbanLoad,
  shouldRestoreKanbanMove,
} from '@/web/web/components/view/kanbanStoreHelpers';

export type TokenKanbanRow =
  | { payload?: Record<string, unknown> | null; key?: string; kind?: string }
  | Record<string, unknown>;

/**
 * Normalize controller RecordRow (`{ payload }`) or a raw model record into a
 * payload object used for card fields / navigation.
 */
export function resolveTokenKanbanRowPayload(row: TokenKanbanRow | null | undefined): Record<string, unknown> {
  if (!row || typeof row !== 'object') return {};
  if ('payload' in row) {
    const wrapped = (row as { payload?: unknown }).payload;
    if (wrapped && typeof wrapped === 'object' && !Array.isArray(wrapped)) {
      return wrapped as Record<string, unknown>;
    }
  }
  // Raw record rows (no wrapper) carry Id / fields on the object itself.
  return row as Record<string, unknown>;
}

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
export function resolveTokenKanbanCardId(row: TokenKanbanRow, index: number, laneKey: string): string {
  const payload = resolveTokenKanbanRowPayload(row);
  const key = row && typeof row === 'object' && 'key' in row ? (row as { key?: string }).key : undefined;
  // A blank payload Id must fall back to the row key (Vue key), not a synthetic index.
  const id = String(payload.Id ?? '').trim() || key;
  const cardId = id == null ? '' : String(id).trim();
  if (cardId !== '') {
    return cardId;
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

/**
 * Username shown on a kanban card. Prefers `UserId.Username`, then a nested
 * UserId.Username object field, then a plain string UserId.
 */
export function resolveTokenUsernameLabel(payload: Record<string, unknown> | null | undefined): string {
  const row = payload ?? {};
  const flat = row['UserId.Username'];
  // A blank flattened label must not shadow a populated UserId relation.
  const user = typeof flat === 'string' && flat.trim() !== '' ? flat : (row.UserId ?? flat);
  if (user && typeof user === 'object') {
    return String((user as { Username?: string }).Username ?? '');
  }
  return String(user ?? '');
}

/** Token alias for kit shouldRestoreKanbanMove. */
export function shouldRestoreTokenKanbanMove(opts: {
  movePending: boolean;
  searchPending?: boolean;
  recordId: string;
  fromLaneKey: string;
  controllerLaneKeys: ReadonlyArray<string>;
}): boolean {
  return shouldRestoreKanbanMove(opts);
}

/** Token alias for kit finishInitialKanbanLoad. */
export async function finishInitialTokenKanbanLoad(opts: {
  getLastSearchQuery: () => unknown;
  applyEmpty: () => Promise<void>;
  onSearch: (query: any) => Promise<void>;
  syncLanes: () => Promise<void>;
}): Promise<void> {
  return finishInitialKanbanLoad(opts);
}
