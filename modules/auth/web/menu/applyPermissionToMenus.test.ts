// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';
import { createTermReference } from '@/core/service/i18n';
import type { PermissionState } from '@/auth/web/permission';
import { applyPermissionToMenus } from './applyPermissionToMenus';

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v));
}

function emptyState(): PermissionState {
  return { byCompany: {}, permStateVersion: 1 };
}

function wildcardMenus(): PermissionState {
  return {
    permStateVersion: 1,
    byCompany: {
      '*': { ui: { routes: [], menus: ['*'], actions: [] } },
    },
  };
}

function menusAllowing(...ids: string[]): PermissionState {
  return {
    permStateVersion: 1,
    byCompany: {
      '*': { ui: { routes: [], menus: ids, actions: [] } },
    },
  };
}

const ctx = { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] };

test('applyPermissionToMenus: hides unauthorized menu item by default', () => {
  const menus = [
    {
      id: 'auth.menu.user_list',
      title: 'Users',
    },
  ] as unknown as MenuItem[];

  applyPermissionToMenus(menus, emptyState(), ctx);

  expect(Boolean((menus[0] as any).hidden)).toBe(true);
  expect(Boolean((menus[0] as any).disabled)).toBe(false);
});

test('applyPermissionToMenus: disables unauthorized menu in disable mode', () => {
  const menus = [
    {
      id: 'auth.menu.user_list',
      title: 'Users',
      meta: { permissionMode: 'disable' },
    },
  ] as unknown as MenuItem[];

  applyPermissionToMenus(menus, emptyState(), ctx);

  expect(Boolean((menus[0] as any).hidden)).toBe(false);
  expect(Boolean((menus[0] as any).disabled)).toBe(true);
});

test('applyPermissionToMenus: hides non-clickable parent when all children are hidden', () => {
  const src = [
    {
      id: 'auth.menu.root',
      title: 'Root',
      children: [
        { id: 'auth.menu.child_a', title: 'A' },
        { id: 'auth.menu.child_b', title: 'B' },
      ],
    },
  ] as unknown as MenuItem[];

  const menus = clone(src) as MenuItem[];
  applyPermissionToMenus(menus, menusAllowing('auth.menu.root'), ctx);

  expect(Boolean((menus[0] as any).hidden)).toBe(true);
  expect(Boolean((menus[0].children?.[0] as any).hidden)).toBe(true);
  expect(Boolean((menus[0].children?.[1] as any).hidden)).toBe(true);
});

test('applyPermissionToMenus: preserves term reference metadata while applying permissions recursively', () => {
  const reference = createTermReference('auth', 'Users', { scope: 'auth.menu.users' });
  const menus = clone([
    {
      id: 'auth.menu.root',
      title: 'Root',
      children: [{ id: 'auth.menu.users', title: 'Users', titleText: reference }],
    },
  ]) as MenuItem[];

  applyPermissionToMenus(menus, wildcardMenus(), {});

  expect(menus[0].children?.[0].titleText).toEqual(reference);
});
