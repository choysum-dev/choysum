// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import type { MenuItem } from '@/core/web/menu';
import { createTextDescriptor } from '@/core/service/i18n';

const canMenuMock = vi.fn<(resourceId: string | undefined, state: any, ctx: any) => boolean>(() => true);

vi.mock('@/auth/web/permission', () => {
  return {
    canMenu: canMenuMock,
  };
});

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v));
}

describe('applyPermissionToMenus', () => {
  beforeEach(() => {
    canMenuMock.mockReset();
  });

  it('hides unauthorized menu item by default', async () => {
    canMenuMock.mockReturnValue(false);
    const { applyPermissionToMenus } = await import('./applyPermissionToMenus');

    const menus = [
      {
        id: 'auth.menu.user_list',
        title: 'Users',
      },
    ] as unknown as MenuItem[];

    applyPermissionToMenus(menus, { byCompany: {}, permStateVersion: 1 } as any, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] });

    expect(Boolean((menus[0] as any).hidden)).toBe(true);
    expect(Boolean((menus[0] as any).disabled)).toBe(false);
  });

  it('disables unauthorized menu in disable mode', async () => {
    canMenuMock.mockReturnValue(false);
    const { applyPermissionToMenus } = await import('./applyPermissionToMenus');

    const menus = [
      {
        id: 'auth.menu.user_list',
        title: 'Users',
        meta: { permissionMode: 'disable' },
      },
    ] as unknown as MenuItem[];

    applyPermissionToMenus(menus, { byCompany: {}, permStateVersion: 1 } as any, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] });

    expect(Boolean((menus[0] as any).hidden)).toBe(false);
    expect(Boolean((menus[0] as any).disabled)).toBe(true);
  });

  it('hides non-clickable parent when all children are hidden', async () => {
    canMenuMock.mockImplementation((id?: string) => id === 'auth.menu.root');
    const { applyPermissionToMenus } = await import('./applyPermissionToMenus');

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
    applyPermissionToMenus(menus, { byCompany: {}, permStateVersion: 1 } as any, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] });

    expect(Boolean((menus[0] as any).hidden)).toBe(true);
    expect(Boolean((menus[0].children?.[0] as any).hidden)).toBe(true);
    expect(Boolean((menus[0].children?.[1] as any).hidden)).toBe(true);
  });

  it('preserves descriptor metadata while applying permissions recursively', async () => {
    canMenuMock.mockReturnValue(true);
    const { applyPermissionToMenus } = await import('./applyPermissionToMenus');
    const descriptor = createTextDescriptor('auth', 'Users', { scope: 'auth.menu.users' });
    const menus = clone([
      {
        id: 'auth.menu.root',
        title: 'Root',
        children: [{ id: 'auth.menu.users', title: 'Users', titleText: descriptor }],
      },
    ]) as MenuItem[];

    applyPermissionToMenus(menus, { byCompany: {}, permStateVersion: 1 } as any, {});

    expect(menus[0].children?.[0].titleText).toEqual(descriptor);
  });
});
