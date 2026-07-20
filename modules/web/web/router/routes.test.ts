// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { createTranslate } from '@/web/web/i18n';
import { menus } from '../menu/menus';
import routes from './routes';

const homeRouteTitle = createTranslate('web', { scope: 'web/route/routes' })._lt('Home');
const homeMenuTitle = createTranslate('web', { scope: 'web/menu/menus' })._lt('Home');

describe('web home resource declarations', () => {
  it('declares home as a protected route resource', () => {
    const layoutRoute = routes.find(route => route.name === 'Layout') as any;
    const homeRoute = layoutRoute?.children?.find((route: any) => route.name === 'Home');

    expect(homeRoute).toBeTruthy();
    expect(homeRoute.meta?.resourceId).toBe('web.route.home');
    expect(homeRoute.meta?.pageTitle).toBe('Home');
    expect(homeRoute.meta?.pageTitleText).toEqual(homeRouteTitle);
    expect(homeRoute.meta?.requiresAuth).toBe(true);
    expect(homeRoute.meta?.routeSequence).toBe(1);
  });

  it('declares home as a menu resource', () => {
    const homeMenu = menus[0] as any;

    expect(homeMenu.id).toBe('web.menu.home');
    expect(homeMenu.title).toBe('Home');
    expect(homeMenu.titleText).toEqual(homeMenuTitle);
    expect(homeMenu.path).toBe('/home');
    expect(homeMenu.order).toBe(1);
    expect(homeMenu.meta?.resourceId).toBe('web.menu.home');
  });
});
