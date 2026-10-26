// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyChoyViewRouteGroups,
  resolveChoyViewRouteDefaultGroups,
  resolveChoyViewRouteKind,
  resolveChoyViewRoutePath,
} from './useChoyViewRouteSwitch';

test('resolveChoyViewRouteKind reads meta.viewKind with list fallback', () => {
  expect(resolveChoyViewRouteKind({ viewKind: 'kanban' })).toBe('kanban');
  expect(resolveChoyViewRouteKind({ viewKind: '  chart  ' })).toBe('chart');
  expect(resolveChoyViewRouteKind({ viewKind: '   ' })).toBe('list');
  expect(resolveChoyViewRouteKind({})).toBe('list');
  expect(resolveChoyViewRouteKind(null)).toBe('list');
  expect(resolveChoyViewRouteKind(undefined, 'board')).toBe('board');
});

test('resolveChoyViewRouteDefaultGroups copies string groups only', () => {
  expect(resolveChoyViewRouteDefaultGroups({ defaultGroups: ['State', 'IsActive'] })).toEqual([
    'State',
    'IsActive',
  ]);
  expect(resolveChoyViewRouteDefaultGroups({ defaultGroups: ['A', 1 as unknown as string] })).toEqual([
    'A',
  ]);
  expect(resolveChoyViewRouteDefaultGroups({ defaultGroups: 'State' })).toEqual([]);
  expect(resolveChoyViewRouteDefaultGroups(null)).toEqual([]);
});

test('resolveChoyViewRoutePath looks up viewPaths by kind', () => {
  const meta = {
    viewPaths: {
      list: '/sandbox/records',
      kanban: '/sandbox/records/kanban',
      chart: '  /sandbox/records/chart  ',
    },
  };
  expect(resolveChoyViewRoutePath(meta, 'list')).toBe('/sandbox/records');
  expect(resolveChoyViewRoutePath(meta, 'chart')).toBe('/sandbox/records/chart');
  expect(resolveChoyViewRoutePath(meta, 'board')).toBe('');
  expect(resolveChoyViewRoutePath({}, 'list')).toBe('');
});

test('applyChoyViewRouteGroups overwrites appliedGroups and preserves siblings', () => {
  const store = {
    state: { queryState: { appliedGroups: ['Revoked'], keyword: 'keep-me' } },
  };
  applyChoyViewRouteGroups(store, ['TokenType', 'Revoked']);
  expect(store.state.queryState.appliedGroups).toEqual(['TokenType', 'Revoked']);
  expect(store.state.queryState.keyword).toBe('keep-me');

  applyChoyViewRouteGroups(store, []);
  expect(store.state.queryState.appliedGroups).toEqual([]);
});

test('applyChoyViewRouteGroups creates queryState when missing', () => {
  const store: { state?: { queryState?: { appliedGroups?: unknown } } } = {};
  applyChoyViewRouteGroups(store, ['State']);
  expect(store.state?.queryState?.appliedGroups).toEqual(['State']);
});

test('applyChoyViewRouteGroups ignores nullish or non-object store', () => {
  applyChoyViewRouteGroups(null, ['State']);
  applyChoyViewRouteGroups(undefined, ['State']);
  applyChoyViewRouteGroups('nope', ['State']);
});
