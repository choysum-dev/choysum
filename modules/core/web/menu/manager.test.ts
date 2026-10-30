// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MenuManager } from './manager';

test('MenuManager: tracks nested menus by id, path, parent and export shape', () => {
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

test('MenuManager: queues children until parent is registered', () => {
  const manager = new MenuManager();

  manager.addMenu('web.menu.settings', {
    id: 'auth.menu.settings_app',
    title: 'auth',
    order: 20,
    children: [
      {
        id: 'auth.menu.terminology',
        title: 'Translations',
        path: '/web/terminology/auth',
        order: 10,
      },
    ],
  });
  expect(manager.getMenu('auth.menu.settings_app')).toBeUndefined();

  manager.addMenu({
    id: 'web.menu.settings',
    title: 'Settings',
    order: 110,
    children: [
      {
        id: 'web.menu.settings_app_core',
        title: 'core',
        order: 10,
        children: [
          {
            id: 'web.menu.terminology_core',
            title: 'Translations',
            path: '/web/terminology/core',
            order: 10,
          },
        ],
      },
    ],
  });

  expect(manager.getMenu('auth.menu.terminology')?.path).toBe('/web/terminology/auth');
  expect(manager.getMenuChildren('web.menu.settings').map((item) => item.id).sort()).toEqual([
    'auth.menu.settings_app',
    'web.menu.settings_app_core',
  ]);
});

test('MenuManager: replaces paths and removes descendants recursively', () => {
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

test('MenuManager: clearMenus drops pending children', () => {
  const manager = new MenuManager();
  manager.addMenu('missing.parent', {
    id: 'queued.child',
    title: 'Queued',
    path: '/queued',
  });
  manager.clearMenus();
  manager.addMenu({
    id: 'missing.parent',
    title: 'Parent',
  });
  expect(manager.getMenu('queued.child')).toBeUndefined();
});

test('MenuManager: flushes pending for nested configured descendants', () => {
  const manager = new MenuManager();
  manager.addMenu('web.menu.settings_app_core', {
    id: 'auth.menu.terminology',
    title: 'Translations',
    path: '/web/terminology/auth',
  });
  manager.addMenu({
    id: 'web.menu.settings',
    title: 'Settings',
    children: [
      {
        id: 'web.menu.settings_app_core',
        title: 'core',
        children: [
          {
            id: 'web.menu.terminology_core',
            title: 'Translations',
            path: '/web/terminology/core',
          },
        ],
      },
    ],
  });
  expect(manager.getMenu('auth.menu.terminology')?.path).toBe('/web/terminology/auth');
  expect(manager.getMenuParent('auth.menu.terminology')?.id).toBe('web.menu.settings_app_core');
});

test('MenuManager: removeMenu clears pending queue for that id', () => {
  const manager = new MenuManager();
  manager.addMenu('doomed.parent', {
    id: 'orphan.child',
    title: 'Orphan',
    path: '/orphan',
  });
  manager.addMenu({ id: 'doomed.parent', title: 'Doomed' });
  expect(manager.getMenu('orphan.child')).toBeTruthy();
  manager.addMenu('doomed.parent', {
    id: 'late.child',
    title: 'Late',
    path: '/late',
  });
  expect(manager.removeMenu('doomed.parent')).toBe(true);
  manager.addMenu({ id: 'doomed.parent', title: 'Reborn' });
  expect(manager.getMenu('late.child')).toBeUndefined();
});

test('MenuManager: clearMenus recursively drops nested pending queues', () => {
  const manager = new MenuManager();
  manager.addMenu('parent.a', { id: 'child.b', title: 'B' });
  manager.addMenu('child.b', { id: 'grand.c', title: 'C', path: '/c' });
  manager.clearMenus();
  manager.addMenu({ id: 'parent.a', title: 'A' });
  manager.addMenu({ id: 'child.b', title: 'B2' });
  expect(manager.getMenu('grand.c')).toBeUndefined();
  expect(manager.getMenu('child.b')?.title).toBe('B2');
});

test('MenuManager: multiple pending children flush in order when parent arrives', () => {
  const manager = new MenuManager();
  manager.addMenu('parent.x', { id: 'child.1', title: 'One', path: '/1', order: 20 });
  manager.addMenu('parent.x', { id: 'child.2', title: 'Two', path: '/2', order: 10 });
  expect(manager.getMenu('child.1')).toBeUndefined();
  manager.addMenu({ id: 'parent.x', title: 'Parent' });
  expect(manager.getMenuChildren('parent.x').map(item => item.id)).toEqual(['child.2', 'child.1']);
});

test('MenuManager: loadMenusFromConfig clears stale pending before applying config', () => {
  const manager = new MenuManager();
  manager.addMenu('ghost.parent', { id: 'ghost.child', title: 'Ghost', path: '/ghost' });
  manager.loadMenusFromConfig([{ id: 'ghost.parent', title: 'Real', path: '/real' }]);
  expect(manager.getMenu('ghost.child')).toBeUndefined();
  expect(manager.getMenuByPath('/real')?.id).toBe('ghost.parent');
});
