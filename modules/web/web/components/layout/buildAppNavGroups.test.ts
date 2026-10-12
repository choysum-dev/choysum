// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildAppNavGroups, visibleMenuChildren } from './buildAppNavGroups';

const labelOf = (item: { title?: string }) => String(item.title || '');

describe('buildAppNavGroups', () => {
  test('returns empty for null root', () => {
    expect(buildAppNavGroups(null, labelOf)).toEqual([]);
  });

  test('wraps a leaf-only root as a single ungrouped item', () => {
    const root = { id: 'solo', title: 'Solo', path: '/solo' } as any;
    expect(buildAppNavGroups(root, labelOf)).toEqual([
      { key: 'ungrouped', label: '', items: [root] },
    ]);
  });

  test('splits section roots and loose leaves', () => {
    const root = {
      id: 'base',
      title: 'Base',
      children: [
        { id: 'loose', title: 'Company', path: '/c' },
        {
          id: 'grp',
          title: 'Units',
          children: [
            { id: 'u1', title: 'Unit', path: '/u1' },
            { id: 'hidden', title: 'Hidden', path: '/h', hidden: true },
          ],
        },
      ],
    } as any;
    const groups = buildAppNavGroups(root, labelOf);
    expect(groups.map((g) => ({ key: g.key, label: g.label, ids: g.items.map((i) => i.id) }))).toEqual(
      [
        { key: 'base-loose', label: '', ids: ['loose'] },
        { key: 'grp', label: 'Units', ids: ['u1'] },
      ],
    );
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
