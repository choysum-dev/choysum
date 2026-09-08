// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/web/web/i18n';
import { menus } from '../menu/menus';
import { routes } from '../router/routes';
import { useBreadcrumbStore } from '../stores/breadcrumbStore';

test('web shell _lt bindings: pins TermReference titles on web menus and routes', () => {
  const homeMenuTitle = createTranslate('web', { scope: 'web/menu/menus' })._lt('Home');
  const homeRouteTitle = createTranslate('web', { scope: 'web/route/routes' })._lt('Home');
  const root = menus[0] as any;
  expect(root.title).toBe('Home');
  expect(root.titleText).toEqual(homeMenuTitle);

  expect(routes.length).toBeGreaterThan(0);
  const layoutRoute = routes.find(route => route.name === 'Layout') as any;
  const homeRoute = layoutRoute?.children?.find((route: any) => route.name === 'Home');
  expect(homeRoute).toBeTruthy();
  expect(homeRoute.meta?.pageTitle).toBe('Home');
  expect(homeRoute.meta?.pageTitleText).toEqual(homeRouteTitle);
});

test('web shell _lt bindings: breadcrumbStore factory-default _lt titles', () => {
  const expectedPage = createTranslate('web', { scope: 'web/stores/breadcrumbStore' })._lt('Page');
  const expectedDetails = createTranslate('web', { scope: 'web/stores/breadcrumbStore' })._lt('Details');
  expect(useBreadcrumbStore).toBeTypeOf('function');
  expect(expectedPage.src).toBe('Page');
  expect(expectedDetails.src).toBe('Details');
  expect(expectedPage).toEqual(
    createTranslate('web', { scope: 'web/stores/breadcrumbStore' })._lt('Page'),
  );
  expect(expectedDetails).toEqual(
    createTranslate('web', { scope: 'web/stores/breadcrumbStore' })._lt('Details'),
  );
});
