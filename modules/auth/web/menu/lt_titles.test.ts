// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { createTranslate } from '@/web/web/i18n';
import { authMenus } from './menus';
import { authRoutes, appRoutes } from '../route/routes';

const menuTitle = createTranslate('auth', { scope: 'web/menu/menus' })._lt('Access Control');
const loginTitle = createTranslate('auth', { scope: 'web/route/routes' })._lt('Login');

describe('auth menu/route _lt titles', () => {
  it('pins TermReference titles on auth menus', () => {
    const root = authMenus[0] as any;
    expect(root.title).toBe('Access Control');
    expect(root.titleText).toEqual(menuTitle);
    expect(root.children?.some((child: any) => child.titleText?.src === 'User List')).toBe(true);
  });

  it('pins Access Rules group and flat leaf paths', () => {
    const root = authMenus[0] as any;
    const accessRules = root.children?.find((child: any) => child.titleText?.src === 'Access Rules');
    expect(accessRules).toBeTruthy();
    expect(accessRules.path).toBeUndefined();
    expect(accessRules.children?.map((c: any) => c.titleText?.src)).toEqual([
      'Record Rules',
      'Field Rules',
      'Method Access',
      'UI Resource Grants',
    ]);
    expect(accessRules.children?.map((c: any) => c.path)).toEqual([
      '/auth/record-rules',
      '/auth/field-rules',
      '/auth/method-accesses',
      '/auth/ui-resource-grants',
    ]);
  });

  it('pins TermReference titles on auth routes', () => {
    const login = authRoutes.find((route: any) => route.name === 'login') as any;
    expect(login).toBeTruthy();
    expect(login.meta?.pageTitle).toBe('Login');
    expect(login.meta?.pageTitleText).toEqual(loginTitle);
  });

  it('pins Access Rules list route titles', () => {
    const recordList = appRoutes.find((route: any) => route.name === 'RecordRuleList') as any;
    expect(recordList).toBeTruthy();
    expect(recordList.path).toBe('auth/record-rules');
    expect(recordList.meta?.pageTitle).toBe('Record Rule List');
  });
});
