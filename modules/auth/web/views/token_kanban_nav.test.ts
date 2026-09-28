// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  resolveTokenDetailId,
  resolveTokenKanbanCardId,
  resolveTokenMoveRecordId,
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

test('resolveTokenKanbanCardId: prefers payload Id then row key', () => {
  expect(resolveTokenKanbanCardId({ payload: { Id: 't1' } }, 0, 'lane-a')).toBe('t1');
  expect(resolveTokenKanbanCardId({ key: 'row-k', payload: {} }, 0, 'lane-a')).toBe('row-k');
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
