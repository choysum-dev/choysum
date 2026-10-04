// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { menus } from '../menu/menus';
import routes from './routes';
import { MODULE_BOARD_PATH } from './resolveDefaultLandPath';
import { createPinia, setActivePinia } from 'pinia';

test('web routes: root and catch-all redirect via default land path', () => {
  setActivePinia(createPinia());
  const root = routes.find(route => route.name === 'Root') as any;
  expect(root).toBeTruthy();
  expect(typeof root.redirect).toBe('function');
  // Without menu injection, runtime land falls back to Module Board.
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

test('web routes: registers Error page and no Home route/menu', () => {
  const layout = routes.find(route => route.name === 'Layout') as any;
  const error = layout?.children?.find((route: any) => route.name === 'Error');
  expect(error).toBeTruthy();
  expect(error.path).toBe('error/:code(\\d+)');
  expect(error.meta?.requiresAuth).toBe(false);
  expect(error.meta?.isAuthPage).toBe(true);

  const home = layout?.children?.find((route: any) => route.name === 'Home');
  expect(home).toBeUndefined();
  expect(menus).toEqual([]);
});

test('AppLayout enables sidebar menu chrome with header and footer', () => {
  const appLayout = routes.find(route => route.name === 'AppLayout') as any;
  expect(appLayout?.props?.showSidebar).toBe(true);
  expect(appLayout?.props?.showFooter).toBe(true);
  expect(appLayout?.props?.showHeader).toBe(true);

  const layout = routes.find(route => route.name === 'Layout') as any;
  expect(layout?.props?.showSidebar).toBe(true);
  expect(layout?.props?.showFooter).toBe(true);
});
