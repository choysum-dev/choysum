// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { menus } from '../menu/menus';
import routes from './routes';
import { MODULE_BOARD_PATH } from './resolveDefaultLandPath';
import { createPinia, setActivePinia } from 'pinia';
import { createMenuPlugin } from '@/core/web/menu';

test('web routes: root and catch-all redirect via default land path', () => {
  setActivePinia(createPinia());
  createMenuPlugin().install({
    config: { globalProperties: {} },
    provide() {},
  } as any);
  const root = routes.find(route => route.name === 'Root') as any;
  expect(root).toBeTruthy();
  expect(typeof root.redirect).toBe('function');
  // Without menus, runtime land falls back to Module Board.
  expect(root.redirect()).toBe(MODULE_BOARD_PATH);

  const catchAll = routes.find(route => route.name === 'CatchAll') as any;
  expect(typeof catchAll.redirect).toBe('function');
  // Unknown path → Module Board.
  expect(catchAll.redirect({ path: '/unknown-xyz' })).toBe(MODULE_BOARD_PATH);
  // Land path that rematches catch-all → Error (loop guard).
  expect(catchAll.redirect({ path: MODULE_BOARD_PATH })).toEqual({
    name: 'Error',
    params: { code: '404' },
  });
});

test('web routes: registers Error page under GuestLayout and no Home route/menu', () => {
  const guest = routes.find(route => route.name === 'GuestLayout') as any;
  const error = guest?.children?.find((route: any) => route.name === 'Error');
  expect(error).toBeTruthy();
  expect(error.path).toBe('error/:code(\\d+)');
  expect(error.meta?.requiresAuth).toBe(false);
  expect(error.meta?.isAuthPage).toBeUndefined();

  const home = guest?.children?.find((route: any) => route.name === 'Home');
  expect(home).toBeUndefined();
  expect(menus).toEqual([]);
});

test('web routes: GuestLayout and AppLayout use separate shells with expected chrome props', () => {
  const appLayout = routes.find(route => route.name === 'AppLayout') as any;
  expect(appLayout?.props?.showSidebar).toBe(true);
  expect(appLayout?.props?.showFooter).toBe(true);
  expect(appLayout?.props?.showHeader).toBe(true);
  expect(String(appLayout?.component?.toString?.() ?? appLayout?.component)).toContain('ChoyAppShell');

  const guest = routes.find(route => route.name === 'GuestLayout') as any;
  expect(guest?.props?.showFooter).toBe(true);
  expect(guest?.props?.showHeader).toBe(true);
  expect(guest?.props?.showSidebar).toBeUndefined();
  expect(String(guest?.component?.toString?.() ?? guest?.component)).toContain('ChoyGuestShell');
});
