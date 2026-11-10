// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  resolveChoyKanbanCardId,
  type ChoyKanbanCard,
} from './kanbanViewHelpers';

/**
 * Coalesce overlapping lane syncs: waiters resume after the in-flight pass (and any
 * follow-up resync) finishes, instead of resolving immediately.
 */
export function createLaneSyncGate() {
  let syncing = false;
  let pending = false;
  let waiters: Array<() => void> = [];
  return {
    /** @returns 'run' when this caller owns the sync loop; 'waited' after draining. */
    async enter(): Promise<'run' | 'waited'> {
      if (syncing) {
        pending = true;
        await new Promise<void>((resolve) => {
          waiters.push(resolve);
        });
        return 'waited';
      }
      syncing = true;
      return 'run';
    },
    beginPass(): void {
      pending = false;
    },
    shouldResync(): boolean {
      return pending;
    },
    leave(): void {
      syncing = false;
      const queued = waiters;
      waiters = [];
      for (const resolve of queued) resolve();
    },
  };
}

/**
 * True when an optimistic card move must be discarded (resync) instead of
 * persisted: overlapping writes/searches, missing record id, or the flat
 * "all" lane that is not a controller group lane.
 */
export function shouldRestoreKanbanMove(opts: {
  movePending: boolean;
  searchPending?: boolean;
  recordId: string;
  fromLaneKey: string;
  controllerLaneKeys: ReadonlyArray<string>;
}): boolean {
  if (opts.movePending || opts.searchPending || !opts.recordId) return true;
  return !opts.controllerLaneKeys.some((key) => key === opts.fromLaneKey);
}

/**
 * True when a completed search is stale and no newer apply is in flight, so the
 * latest query should be re-applied once.
 */
export function shouldRecoverStaleKanbanSearch(args: {
  completedSeq: number;
  latestSeq: number;
  inFlight: number;
}): boolean {
  return args.completedSeq !== args.latestSeq && args.inFlight === 0;
}

/**
 * Initial kanban load.
 * When deferToSearch is true (ChoySearchView in the search slot), skip applyEmpty
 * and wait for the first-frame query-update → applySearch (same contract as List).
 * Without a search chrome, apply an empty query once and sync lanes.
 */
export async function finishInitialKanbanLoad(opts: {
  getLastSearchQuery: () => unknown;
  applyEmpty: () => Promise<void>;
  syncLanes: () => Promise<void>;
  /** When true, do not applyEmpty; first applySearch owns the first load. */
  deferToSearch?: boolean;
}): Promise<void> {
  if (opts.getLastSearchQuery()) return;
  if (opts.deferToSearch) return;
  await opts.applyEmpty();
  await opts.syncLanes();
}

/** Unwrap controller RecordRow `{ payload }` or accept a raw model record. */
export function resolveKanbanRowPayload(row: unknown): Record<string, unknown> {
  if (!row || typeof row !== 'object') return {};
  if ('payload' in (row as object)) {
    const wrapped = (row as { payload?: unknown }).payload;
    if (wrapped && typeof wrapped === 'object' && !Array.isArray(wrapped)) {
      return wrapped as Record<string, unknown>;
    }
  }
  return row as Record<string, unknown>;
}

/**
 * Default card mapper: prefer payload Id, then row key, then lane-scoped index.
 */
export function defaultKanbanMapRowToCard(
  row: unknown,
  index: number,
  laneKey: string,
  titleField = 'Title',
): ChoyKanbanCard {
  const payload = resolveKanbanRowPayload(row);
  const key =
    row && typeof row === 'object' && 'key' in row
      ? String((row as { key?: unknown }).key ?? '').trim()
      : '';
  const idFromPayload = String(payload.Id ?? payload.id ?? '').trim();
  const id = idFromPayload || key || `${laneKey}-${index}`;
  const titleRaw = payload[titleField] ?? payload.Id ?? payload.id ?? id;
  return {
    id: id || resolveChoyKanbanCardId(payload, index),
    title: String(titleRaw ?? ''),
    laneKey,
    payload,
  };
}

/** Prefer payload.Id for move persistence; blank → empty (restore path). */
export function defaultKanbanMoveRecordId(
  cards: ReadonlyArray<{ id: string; payload?: Record<string, unknown> }>,
  moveCardId: string,
): string {
  const card = cards.find((c) => c.id === moveCardId);
  return String(card?.payload?.Id ?? '').trim();
}
