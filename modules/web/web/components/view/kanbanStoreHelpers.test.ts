// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createLaneSyncGate,
  defaultKanbanMapRowToCard,
  defaultKanbanMoveRecordId,
  finishInitialKanbanLoad,
  resolveKanbanRowPayload,
  shouldRecoverStaleKanbanSearch,
  shouldRestoreKanbanMove,
} from './kanbanStoreHelpers';

describe('kanbanStoreHelpers', () => {
  test('createLaneSyncGate: waiters drain after owner leave', async () => {
    const gate = createLaneSyncGate();
    expect(await gate.enter()).toBe('run');
    let waited = false;
    const waiter = gate.enter().then((mode) => {
      waited = true;
      expect(mode).toBe('waited');
    });
    expect(gate.shouldResync()).toBe(true);
    gate.beginPass();
    expect(gate.shouldResync()).toBe(false);
    expect(waited).toBe(false);
    gate.leave();
    await waiter;
    expect(waited).toBe(true);
  });

  test('shouldRestoreKanbanMove: pending, missing id, or unknown lane', () => {
    expect(
      shouldRestoreKanbanMove({
        movePending: true,
        recordId: 't1',
        fromLaneKey: 'a',
        controllerLaneKeys: ['a'],
      }),
    ).toBe(true);
    expect(
      shouldRestoreKanbanMove({
        movePending: false,
        searchPending: true,
        recordId: 't1',
        fromLaneKey: 'a',
        controllerLaneKeys: ['a'],
      }),
    ).toBe(true);
    expect(
      shouldRestoreKanbanMove({
        movePending: false,
        recordId: '',
        fromLaneKey: 'a',
        controllerLaneKeys: ['a'],
      }),
    ).toBe(true);
    expect(
      shouldRestoreKanbanMove({
        movePending: false,
        recordId: 't1',
        fromLaneKey: 'all',
        controllerLaneKeys: ['Revoked=true'],
      }),
    ).toBe(true);
    expect(
      shouldRestoreKanbanMove({
        movePending: false,
        recordId: 't1',
        fromLaneKey: 'Revoked=true',
        controllerLaneKeys: ['Revoked=true'],
      }),
    ).toBe(false);
  });

  test('shouldRecoverStaleKanbanSearch: only when stale and idle', () => {
    expect(shouldRecoverStaleKanbanSearch({ completedSeq: 1, latestSeq: 2, inFlight: 0 })).toBe(true);
    expect(shouldRecoverStaleKanbanSearch({ completedSeq: 2, latestSeq: 2, inFlight: 0 })).toBe(false);
    expect(shouldRecoverStaleKanbanSearch({ completedSeq: 1, latestSeq: 2, inFlight: 1 })).toBe(false);
  });

  test('finishInitialKanbanLoad: skips empty apply when search already landed', async () => {
    let applied = false;
    await finishInitialKanbanLoad({
      getLastSearchQuery: () => ({ keyword: 'x' }),
      applyEmpty: async () => {
        applied = true;
      },
      onSearch: async () => undefined,
      syncLanes: async () => undefined,
    });
    expect(applied).toBe(false);
  });

  test('finishInitialKanbanLoad: re-applies late first-frame search', async () => {
    const calls: string[] = [];
    let query: unknown = null;
    await finishInitialKanbanLoad({
      getLastSearchQuery: () => query,
      applyEmpty: async () => {
        calls.push('empty');
        query = { keyword: 'late' };
      },
      onSearch: async (q) => {
        calls.push(`search:${(q as { keyword: string }).keyword}`);
      },
      syncLanes: async () => {
        calls.push('sync');
      },
    });
    expect(calls).toEqual(['empty', 'search:late']);
  });

  test('finishInitialKanbanLoad: syncs when no search arrives', async () => {
    const calls: string[] = [];
    await finishInitialKanbanLoad({
      getLastSearchQuery: () => null,
      applyEmpty: async () => {
        calls.push('empty');
      },
      onSearch: async () => {
        calls.push('search');
      },
      syncLanes: async () => {
        calls.push('sync');
      },
    });
    expect(calls).toEqual(['empty', 'sync']);
  });

  test('resolveKanbanRowPayload / default mappers', () => {
    expect(resolveKanbanRowPayload({ payload: { Id: '1', Title: 'A' } })).toEqual({ Id: '1', Title: 'A' });
    expect(resolveKanbanRowPayload({ Id: '2', Title: 'B' })).toEqual({ Id: '2', Title: 'B' });
    const card = defaultKanbanMapRowToCard({ payload: { Id: '9', Title: 'Hi' } }, 0, 'lane');
    expect(card).toEqual({ id: '9', title: 'Hi', laneKey: 'lane', payload: { Id: '9', Title: 'Hi' } });
    expect(
      defaultKanbanMoveRecordId([{ id: 'vue-key', payload: { Id: 'real' } }], 'vue-key'),
    ).toBe('real');
    expect(defaultKanbanMoveRecordId([{ id: 'x', payload: {} }], 'x')).toBe('');
  });
});
