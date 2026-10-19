// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  ensureTokenViewGroups,
  formatTokenRevokedChartLabel,
  resolveTokenViewKind,
  tokenViewDefaultGroups,
  tokenViewPath,
} from './token_view_nav';

test('resolveTokenViewKind maps list / kanban / chart paths', () => {
  expect(resolveTokenViewKind('/auth/tokens')).toBe('list');
  expect(resolveTokenViewKind('/auth/tokens/')).toBe('list');
  expect(resolveTokenViewKind('/auth/tokens/kanban')).toBe('kanban');
  expect(resolveTokenViewKind('/auth/tokens/chart')).toBe('chart');
  expect(resolveTokenViewKind('/web/auth/tokens/chart')).toBe('chart');
});

test('tokenViewPath returns stable management URLs', () => {
  expect(tokenViewPath('list')).toBe('/auth/tokens');
  expect(tokenViewPath('kanban')).toBe('/auth/tokens/kanban');
  expect(tokenViewPath('chart')).toBe('/auth/tokens/chart');
});

test('tokenViewDefaultGroups matches each management view', () => {
  expect(tokenViewDefaultGroups('chart')).toEqual(['TokenType', 'Revoked']);
  expect(tokenViewDefaultGroups('kanban')).toEqual(['Revoked']);
  expect(tokenViewDefaultGroups('list')).toEqual([]);
});

test('ensureTokenViewGroups overwrites prior appliedGroups on the shared store', () => {
  const store = {
    state: { queryState: { appliedGroups: ['Revoked'], keyword: 'keep-me' } },
  };
  ensureTokenViewGroups(store, 'chart');
  expect(store.state.queryState.appliedGroups).toEqual(['TokenType', 'Revoked']);
  expect(store.state.queryState.keyword).toBe('keep-me');

  ensureTokenViewGroups(store, 'kanban');
  expect(store.state.queryState.appliedGroups).toEqual(['Revoked']);

  ensureTokenViewGroups(store, 'list');
  expect(store.state.queryState.appliedGroups).toEqual([]);
});

test('ensureTokenViewGroups creates queryState when missing', () => {
  const store: { state?: { queryState?: { appliedGroups?: unknown } } } = {};
  ensureTokenViewGroups(store, 'chart');
  expect(store.state?.queryState?.appliedGroups).toEqual(['TokenType', 'Revoked']);
});

test('formatTokenRevokedChartLabel maps boolean and field=value keys', () => {
  const labels = { revoked: 'Revoked', notRevoked: 'Not Revoked' };
  expect(formatTokenRevokedChartLabel('true', labels)).toBe('Revoked');
  expect(formatTokenRevokedChartLabel('false', labels)).toBe('Not Revoked');
  expect(formatTokenRevokedChartLabel('Revoked=true', labels)).toBe('Revoked');
  expect(formatTokenRevokedChartLabel('Revoked=false', labels)).toBe('Not Revoked');
  expect(formatTokenRevokedChartLabel('access', labels)).toBe('access');
});
