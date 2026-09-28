// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  finishInitialTokenKanbanLoad,
  resolveTokenDetailId,
  resolveTokenKanbanCardId,
  resolveTokenKanbanRowPayload,
  resolveTokenMoveRecordId,
  resolveTokenUsernameLabel,
  shouldRestoreTokenKanbanMove,
} from './token_kanban_nav';

test('resolveTokenDetailId: returns trimmed payload Id', () => {
  expect(resolveTokenDetailId({ Id: '  tok-1  ' })).toBe('tok-1');
});

test('resolveTokenDetailId: ignores missing or blank Id', () => {
  expect(resolveTokenDetailId(undefined)).toBe('');
  expect(resolveTokenDetailId(null)).toBe('');
  expect(resolveTokenDetailId({})).toBe('');
  expect(resolveTokenDetailId({ Id: '   ' })).toBe('');
});

test('resolveTokenKanbanRowPayload: unwraps RecordRow payload', () => {
  expect(
    resolveTokenKanbanRowPayload({
      kind: 'record',
      key: 'k1',
      payload: { Id: 't1', TokenType: 'access' },
    }),
  ).toEqual({ Id: 't1', TokenType: 'access' });
});

test('resolveTokenKanbanRowPayload: accepts raw model records', () => {
  expect(resolveTokenKanbanRowPayload({ Id: 't2', TokenType: 'refresh' })).toEqual({
    Id: 't2',
    TokenType: 'refresh',
  });
});

test('resolveTokenKanbanRowPayload: empty or nullish payload falls back to row', () => {
  expect(resolveTokenKanbanRowPayload({ kind: 'record', key: 'k', payload: null })).toEqual({
    kind: 'record',
    key: 'k',
    payload: null,
  });
  expect(resolveTokenKanbanRowPayload(undefined)).toEqual({});
});

test('resolveTokenKanbanCardId: prefers payload Id then row key', () => {
  expect(resolveTokenKanbanCardId({ payload: { Id: 't1' } }, 0, 'lane-a')).toBe('t1');
  expect(resolveTokenKanbanCardId({ key: 'row-k', payload: {} }, 0, 'lane-a')).toBe('row-k');
  expect(resolveTokenKanbanCardId({ payload: { Id: '  padded  ' } }, 0, 'lane-a')).toBe('padded');
  // Blank / whitespace Id is treated as missing so the stable row key wins.
  expect(resolveTokenKanbanCardId({ key: 'row-k', payload: { Id: '   ' } }, 0, 'lane-a')).toBe('row-k');
});

test('resolveTokenKanbanCardId: reads Id from raw records', () => {
  expect(resolveTokenKanbanCardId({ Id: 'raw-9', TokenType: 'access' }, 0, 'all')).toBe('raw-9');
});

test('resolveTokenKanbanCardId: scopes index fallback by lane', () => {
  expect(resolveTokenKanbanCardId({ payload: {} }, 0, 'Revoked=true')).toBe('Revoked=true-0');
  expect(resolveTokenKanbanCardId({ payload: {} }, 0, 'Revoked=false')).toBe('Revoked=false-0');
});

test('resolveTokenMoveRecordId: requires a real payload Id', () => {
  const cards = [
    { id: 'Revoked=true-0', payload: {} },
    { id: 'tok-9', payload: { Id: 'tok-9' } },
  ];
  expect(resolveTokenMoveRecordId(cards, 'Revoked=true-0')).toBe('');
  expect(resolveTokenMoveRecordId(cards, 'tok-9')).toBe('tok-9');
  expect(resolveTokenMoveRecordId(cards, 'missing')).toBe('');
});

test('resolveTokenUsernameLabel: prefers UserId.Username then string UserId', () => {
  expect(resolveTokenUsernameLabel({ 'UserId.Username': 'alice' })).toBe('alice');
  expect(resolveTokenUsernameLabel({ UserId: { Username: 'bob' } })).toBe('bob');
  expect(resolveTokenUsernameLabel({ UserId: 'user-42' })).toBe('user-42');
  // Blank flattened label must fall through to the relation object.
  expect(resolveTokenUsernameLabel({ 'UserId.Username': '  ', UserId: { Username: 'carol' } })).toBe('carol');
  // Relation object without Username must not stringify to "[object Object]".
  expect(resolveTokenUsernameLabel({ UserId: { Id: 'u1' } })).toBe('');
  expect(resolveTokenUsernameLabel({})).toBe('');
  expect(resolveTokenUsernameLabel(null)).toBe('');
});

test('shouldRestoreTokenKanbanMove: pending, missing id, or unknown lane', () => {
  expect(
    shouldRestoreTokenKanbanMove({
      movePending: true,
      recordId: 't1',
      fromLaneKey: 'Revoked=false',
      controllerLaneKeys: ['Revoked=false'],
    }),
  ).toBe(true);
  expect(
    shouldRestoreTokenKanbanMove({
      movePending: false,
      searchPending: true,
      recordId: 't1',
      fromLaneKey: 'Revoked=false',
      controllerLaneKeys: ['Revoked=false'],
    }),
  ).toBe(true);
  expect(
    shouldRestoreTokenKanbanMove({
      movePending: false,
      recordId: '',
      fromLaneKey: 'Revoked=false',
      controllerLaneKeys: ['Revoked=false'],
    }),
  ).toBe(true);
  expect(
    shouldRestoreTokenKanbanMove({
      movePending: false,
      recordId: 't1',
      fromLaneKey: 'all',
      controllerLaneKeys: [],
    }),
  ).toBe(true);
  expect(
    shouldRestoreTokenKanbanMove({
      movePending: false,
      recordId: 't1',
      fromLaneKey: 'Revoked=false',
      controllerLaneKeys: ['Revoked=false', 'Revoked=true'],
    }),
  ).toBe(false);
});

test('finishInitialTokenKanbanLoad: skips empty apply when search already landed', async () => {
  const calls: string[] = [];
  await finishInitialTokenKanbanLoad({
    getLastSearchQuery: () => ({ keyword: 'x' }),
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
  expect(calls).toEqual([]);
});

test('finishInitialTokenKanbanLoad: re-applies late first-frame search', async () => {
  const calls: string[] = [];
  let late: { keyword: string } | null = null;
  await finishInitialTokenKanbanLoad({
    getLastSearchQuery: () => late,
    applyEmpty: async () => {
      calls.push('empty');
      late = { keyword: 'hi' };
    },
    onSearch: async (q) => {
      calls.push(`search:${q.keyword}`);
    },
    syncLanes: async () => {
      calls.push('sync');
    },
  });
  expect(calls).toEqual(['empty', 'search:hi']);
});

test('finishInitialTokenKanbanLoad: syncs when no search arrives', async () => {
  const calls: string[] = [];
  await finishInitialTokenKanbanLoad({
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
