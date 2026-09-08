// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { partnerListMenuTitle, partnerRootMenuTitle } from '../menu/titles';
import { partnerRoutes } from '../route/routes';

const listRouteTitle = createTranslate('partner', { scope: 'web/route/routes' })._lt('Partner List');

test('partner menu titles: menus.ts registers shared TermReferences', () => {
  expect(partnerRootMenuTitle.src).toBe('Partner Management');
  expect(partnerRootMenuTitle.module).toBe('partner');
  expect(partnerListMenuTitle.src).toBe('Partner List');
});

test('partner route titles: partnerRoutes pins Partner List pageTitleText', () => {
  const list = partnerRoutes.find(route => route.name === 'PartnerList') as any;
  expect(list).toBeTruthy();
  expect(list.meta?.pageTitle).toBe('Partner List');
  expect(list.meta?.pageTitleText).toEqual(listRouteTitle);
});
