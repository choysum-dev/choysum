// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildAppNavGroups, visibleMenuChildren } from './buildAppNavGroups';

describe('buildAppNavGroups', () => {
  test('returns empty for null root', () => {
    expect(buildAppNavGroups(null)).toEqual([]);
  });

  test('wraps a leaf-only root as a single ungrouped item', () => {
    const root = { id: 'solo', title: 'Solo', path: '/solo' } as any;
    expect(buildAppNavGroups(root)).toEqual([
      { key: 'ungrouped', label: '', items: [root] },
    ]);
  });

  test('returns empty when every declared child is hidden', () => {
    const root = {
      id: 'app',
      title: 'App',
      children: [{ id: 'h', title: 'Hidden', path: '/h', hidden: true }],
    } as any;
    expect(buildAppNavGroups(root)).toEqual([]);
  });

  test('keeps section roots as branch items with nested children', () => {
    const nested = [
      { id: 'u1', title: 'Unit', path: '/u1' },
      { id: 'hidden', title: 'Hidden', path: '/h', hidden: true },
    ];
    const grp = {
      id: 'grp',
      title: 'Units',
      children: nested,
    };
    const root = {
      id: 'base',
      title: 'Base',
      children: [{ id: 'loose', title: 'Company', path: '/c' }, grp],
    } as any;
    const groups = buildAppNavGroups(root);
    expect(groups).toHaveLength(1);
    expect(groups[0]!.key).toBe('base');
    expect(groups[0]!.label).toBe('');
    expect(groups[0]!.items.map((i) => i.id)).toEqual(['loose', 'grp']);
    expect(groups[0]!.items[1]).toBe(grp);
  });

  test('visibleMenuChildren skips hidden', () => {
    expect(
      visibleMenuChildren({
        id: 'a',
        children: [
          { id: '1', title: 'A' },
          { id: '2', title: 'B', hidden: true },
        ],
      } as any).map((c) => c.id),
    ).toEqual(['1']);
  });
});
