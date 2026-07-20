// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { createTranslate } from '@/web/web/i18n';
import { authMenus } from './menus';
import { authRoutes } from '../route/routes';

const menuTitle = createTranslate('auth', { scope: 'web/menu/menus' })._lt('Access Control');
const loginTitle = createTranslate('auth', { scope: 'web/route/routes' })._lt('Login');

describe('auth menu/route _lt titles', () => {
  it('pins TermReference titles on auth menus', () => {
    const root = authMenus[0] as any;
    expect(root.title).toBe('Access Control');
    expect(root.titleText).toEqual(menuTitle);
    expect(root.children?.some((child: any) => child.titleText?.src === 'User List')).toBe(true);
  });

  it('pins TermReference titles on auth routes', () => {
    const login = authRoutes.find((route: any) => route.name === 'login') as any;
    expect(login).toBeTruthy();
    expect(login.meta?.pageTitle).toBe('Login');
    expect(login.meta?.pageTitleText).toEqual(loginTitle);
  });
});
