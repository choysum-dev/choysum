// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { MenuManager } from './manager';

describe('MenuManager', () => {
  it('tracks nested menus by id, path, parent and export shape', () => {
    const manager = new MenuManager();

    manager.loadMenusFromConfig([
      {
        id: 'root',
        title: 'Root',
        order: 20,
        path: '/root',
        children: [
          {
            id: 'child-b',
            title: 'Child B',
            order: 20,
            path: '/root/b',
          },
          {
            id: 'child-a',
            title: 'Child A',
            order: 10,
            path: '/root/a',
          },
        ],
      },
    ]);

    expect(manager.getMenu('root')?.title).toBe('Root');
    expect(manager.getMenuByPath('/root/a')?.id).toBe('child-a');
    expect(manager.getMenuParent('child-a')?.id).toBe('root');
    expect(manager.getMenuChildren('root').map(item => item.id)).toEqual(['child-a', 'child-b']);

    const exported = manager.exportMenuConfig();
    expect(exported[0]?.children?.[0]).not.toHaveProperty('__parent');
    expect(exported[0]?.children?.map(item => item.id)).toEqual(['child-a', 'child-b']);
  });

  it('replaces paths and removes descendants recursively', () => {
    const manager = new MenuManager();

    manager.addMenu({
      id: 'root',
      title: 'Root',
      path: '/root',
      children: [
        {
          id: 'child',
          title: 'Child',
          path: '/root/child',
        },
      ],
    });

    expect(manager.getMenuByPath('/root/child')?.id).toBe('child');

    expect(
      manager.replaceMenu('root', {
        id: 'ignored',
        title: 'Root 2',
        path: '/root-2',
        children: [
          {
            id: 'child-2',
            title: 'Child 2',
            path: '/root-2/child-2',
          },
        ],
      })
    ).toBe(true);

    expect(manager.getMenuByPath('/root')).toBeUndefined();
    expect(manager.getMenuByPath('/root/child')).toBeUndefined();
    expect(manager.getMenuByPath('/root-2')?.id).toBe('root');
    expect(manager.getMenuByPath('/root-2/child-2')?.id).toBe('child-2');

    expect(manager.removeMenu('root')).toBe(true);
    expect(manager.getMenu('root')).toBeUndefined();
    expect(manager.getMenu('child-2')).toBeUndefined();
    expect(manager.getMenuByPath('/root-2/child-2')).toBeUndefined();
  });
});
