// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  MODULE_BOARD_PATH,
  findFirstNavigableMenuPath,
  resolveDefaultLandPath,
} from './resolveDefaultLandPath';
import type { MenuItem } from '@/core/web/menu';

describe('resolveDefaultLandPath', () => {
  test('returns first DFS navigable leaf by order', () => {
    const menus: MenuItem[] = [
      {
        id: 'root-b',
        title: 'B',
        order: 20,
        children: [{ id: 'b1', title: 'B1', path: '/b', order: 1 }],
      },
      {
        id: 'root-a',
        title: 'A',
        order: 10,
        children: [
          { id: 'a-hidden', title: 'Hidden', path: '/hidden', hidden: true, order: 1 },
          { id: 'a1', title: 'A1', path: '/a', order: 2 },
        ],
      },
    ];
    expect(findFirstNavigableMenuPath(menus)).toBe('/a');
    expect(resolveDefaultLandPath({ menus })).toBe('/a');
  });

  test('skips disabled / external / filtered paths then falls back to Module Board', () => {
    const menus: MenuItem[] = [
      { id: 'ext', title: 'Ext', path: 'https://example.com', externalLink: true, order: 1 },
      { id: 'dis', title: 'Dis', path: '/disabled', disabled: true, order: 2 },
      { id: 'deny', title: 'Deny', path: '/deny', order: 3 },
    ];
    expect(
      resolveDefaultLandPath({
        menus,
        canNavigate: (p) => p !== '/deny',
      }),
    ).toBe(MODULE_BOARD_PATH);
  });

  test('menuLeafPaths alternative and empty menus use Module Board', () => {
    expect(resolveDefaultLandPath({ menuLeafPaths: ['/x', '/y'] })).toBe('/x');
    expect(
      resolveDefaultLandPath({
        menuLeafPaths: ['/x', '/y'],
        canNavigate: (p) => p === '/y',
      }),
    ).toBe('/y');
    expect(resolveDefaultLandPath()).toBe(MODULE_BOARD_PATH);
    expect(resolveDefaultLandPath({ menus: [] })).toBe(MODULE_BOARD_PATH);
  });

  test('still returns Module Board when canNavigate rejects the board', () => {
    expect(
      resolveDefaultLandPath({
        menus: [],
        canNavigate: () => false,
      }),
    ).toBe(MODULE_BOARD_PATH);
  });

  test('breaks order ties by id and continues past empty child subtrees', () => {
    const menus: MenuItem[] = [
      {
        id: 'a-empty',
        title: 'Empty parent',
        order: 1,
        children: [{ id: 'a-hidden', title: 'Hidden', path: '/hidden', hidden: true, order: 1 }],
      },
      { id: 'm-leaf', title: 'M', path: '/m', order: 1 },
      { id: 'b-leaf', title: 'B', path: '/b', order: 1 },
    ];
    // Same order=1: id sort visits a-empty first (continue after empty kids), then b-leaf.
    expect(resolveDefaultLandPath({ menus })).toBe('/b');
  });

  test('guards against cyclic menu trees', () => {
    const cycleChild: MenuItem = { id: 'cycle', title: 'Cycle', path: '/cycle', order: 1 };
    const parent: MenuItem = {
      id: 'parent',
      title: 'Parent',
      order: 1,
      children: [cycleChild],
    };
    // Mutual cycle: child lists parent again.
    cycleChild.children = [parent];
    expect(() => resolveDefaultLandPath({ menus: [parent] })).not.toThrow();
    // Cycle nodes are parents-with-children, so no leaf; fall back to Module Board.
    expect(resolveDefaultLandPath({ menus: [parent] })).toBe(MODULE_BOARD_PATH);
  });

  test('tolerates nullish menu entries during sort', () => {
    const menus = [
      null,
      { id: 'ok', title: 'Ok', path: '/ok', order: 1 },
      undefined,
    ] as unknown as MenuItem[];
    expect(resolveDefaultLandPath({ menus })).toBe('/ok');
  });
});
