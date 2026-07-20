// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { createTranslate } from '@/web/web/i18n';
import { partnerMenus } from '../menu/menus';
import { partnerRoutes } from '../route/routes';

const rootTitle = createTranslate('partner', { scope: 'web/menu/menus' })._lt('Partner Management');
const listTitle = createTranslate('partner', { scope: 'web/route/routes' })._lt('Partner List');

describe('partner menu/route _lt titles', () => {
  it('pins TermReference titles on partner menus', () => {
    const root = partnerMenus[0] as any;
    expect(root.title).toBe('Partner Management');
    expect(root.titleText).toEqual(rootTitle);
  });

  it('pins TermReference titles on partner routes', () => {
    const list = partnerRoutes.find((route: any) => route.name === 'PartnerList') as any;
    expect(list).toBeTruthy();
    expect(list.meta?.pageTitle).toBe('Partner List');
    expect(list.meta?.pageTitleText).toEqual(listTitle);
  });
});
