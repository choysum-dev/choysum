// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { createTranslate } from '@/web/web/i18n';
import { baseMenus } from './menus';
import { baseRoutes, companyRoutes } from '../route/routes';

const rootTitle = createTranslate('base', { scope: 'web/menu/menus' })._lt('Master Data');
const companyListTitle = createTranslate('base', { scope: 'web/route/routes' })._lt('Company List');

describe('base menu/route _lt titles', () => {
  it('pins TermReference titles on base menus', () => {
    const root = baseMenus[0] as any;
    expect(root.title).toBe('Master Data');
    expect(root.titleText).toEqual(rootTitle);
    expect(root.children?.some((child: any) => child.titleText?.src === 'Company Management')).toBe(true);
  });

  it('pins TermReference titles on base routes', () => {
    expect(baseRoutes.length).toBeGreaterThan(0);
    const company = companyRoutes.find((route: any) => route.name === 'CompanyList') as any;
    expect(company).toBeTruthy();
    expect(company.meta?.pageTitle).toBe('Company List');
    expect(company.meta?.pageTitleText).toEqual(companyListTitle);
  });
});
