// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';
import { collectHomeShortcuts } from './homeShortcuts';

const labelOf = (item: MenuItem) => String(item.title || item.path || item.id || '');

function item(partial: Partial<MenuItem> & { id: string }): MenuItem {
  return partial as MenuItem;
}

describe('collectHomeShortcuts', () => {
  test('collects internal paths and skips hidden, disabled, and external links', () => {
    const menus: MenuItem[] = [
      item({ id: 'a', title: 'Alpha', path: '/a' }),
      item({ id: 'hidden', title: 'Hidden', path: '/h', hidden: true }),
      item({ id: 'disabled', title: 'Disabled', path: '/d', disabled: true }),
      item({ id: 'ext', title: 'Docs', path: 'https://example.com', externalLink: true }),
      item({
        id: 'group',
        title: 'Group',
        disabled: true,
        children: [item({ id: 'child', title: 'Child', path: '/child' })],
      }),
      item({
        id: 'ok-group',
        title: 'OK',
        children: [item({ id: 'b', title: 'Beta', path: '/b' })],
      }),
    ];

    expect(collectHomeShortcuts(menus, 9, labelOf)).toEqual([
      { id: 'a', label: 'Alpha', path: '/a' },
      { id: 'b', label: 'Beta', path: '/b' },
    ]);
  });

  test('dedupes shared group/child paths and respects the limit', () => {
    const menus: MenuItem[] = [
      item({
        id: 'app',
        title: 'App',
        path: '/app',
        children: [
          item({ id: 'app-home', title: 'App Home', path: '/app' }),
          item({ id: 'app-list', title: 'List', path: '/app/list' }),
          item({ id: 'app-form', title: 'Form', path: '/app/form' }),
        ],
      }),
    ];

    expect(collectHomeShortcuts(menus, 2, labelOf)).toEqual([
      { id: 'app', label: 'App', path: '/app' },
      { id: 'app-list', label: 'List', path: '/app/list' },
    ]);
  });

  test('falls back to path as id when menu id is missing', () => {
    const menus: MenuItem[] = [item({ id: '', title: 'Bare', path: '/bare' })];
    expect(collectHomeShortcuts(menus, 1, labelOf)).toEqual([
      { id: '/bare', label: 'Bare', path: '/bare' },
    ]);
  });

  test('skips absolute URLs without externalLink and guards against cycles', () => {
    const cyclicChild = item({ id: 'c', title: 'C', path: '/c' });
    const cyclicParent = item({
      id: 'p',
      title: 'P',
      path: '/p',
      children: [cyclicChild],
    });
    (cyclicChild as any).children = [cyclicParent];

    const menus: MenuItem[] = [
      item({ id: 'abs', title: 'Abs', path: 'https://example.com/x' }),
      cyclicParent,
    ];

    expect(collectHomeShortcuts(menus, 9, labelOf)).toEqual([
      { id: 'p', label: 'P', path: '/p' },
      { id: 'c', label: 'C', path: '/c' },
    ]);
  });
});
