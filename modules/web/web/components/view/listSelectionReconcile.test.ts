// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { listSelectionRowKey, reconcileListSelection } from './listSelectionReconcile';

describe('listSelectionReconcile', () => {
  test('listSelectionRowKey reads key / __rowKey / Id / wrappers', () => {
    expect(listSelectionRowKey(null)).toBeUndefined();
    expect(listSelectionRowKey('x')).toBeUndefined();
    expect(listSelectionRowKey({ key: 'k1' })).toBe('k1');
    expect(listSelectionRowKey({ __rowKey: 'rk' })).toBe('rk');
    expect(listSelectionRowKey({ Id: 'id1' })).toBe('id1');
    expect(listSelectionRowKey({ type: 'record', record: { Id: 'r1' } })).toBe('r1');
    expect(listSelectionRowKey({ kind: 'record', payload: { Id: 'p1' } })).toBe('p1');
    expect(listSelectionRowKey({ payload: { Id: 'fallback' } })).toBe('fallback');
  });

  test('reconcileListSelection replaces matching rows and keeps misses', () => {
    expect(reconcileListSelection([], [{ Id: 'a' }])).toEqual([]);
    expect(reconcileListSelection([{ Id: 'a' }], [])).toEqual([{ Id: 'a' }]);
    const selected = [
      { Id: 'keep-me', Name: 'Before' },
      { Id: 'gone', Name: 'Stale' },
    ];
    const fresh = [{ Id: 'keep-me', Name: 'After' }];
    expect(reconcileListSelection(selected, fresh)).toEqual([
      { Id: 'keep-me', Name: 'After' },
      { Id: 'gone', Name: 'Stale' },
    ]);
    expect(
      reconcileListSelection(
        [{ kind: 'record', key: 'k', payload: { Id: 'k', Name: 'Old' } }],
        [{ kind: 'record', key: 'k', payload: { Id: 'k', Name: 'New' } }],
      )[0],
    ).toEqual({ kind: 'record', key: 'k', payload: { Id: 'k', Name: 'New' } });
  });
});
