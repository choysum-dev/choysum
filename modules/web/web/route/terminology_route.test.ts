// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { terminologyListComponent, terminologyRoutes } from './terminologyRoutes';

const translationsTitle = createTranslate('web', { scope: 'web/route/terminologyRoutes' })._lt('Translations');

test('terminologyListComponent is exported for domain list routes', () => {
  expect(typeof terminologyListComponent).toBe('function');
});

test('terminology list routes: web hosts only core + web for UI_VAL_006 menu match', () => {
  expect(terminologyRoutes.map((r) => r.name)).toEqual([
    'TranslationTermList_core',
    'TranslationTermList_web',
    'TranslationTermDetail',
  ]);

  const coreList = terminologyRoutes.find((r) => r.name === 'TranslationTermList_core') as any;
  expect(coreList).toBeTruthy();
  expect(coreList.path).toBe('web/terminology/core');
  expect(coreList.meta?.resourceId).toBe('core.route.terminology_list');
  expect(coreList.meta?.pageTitleText).toEqual(translationsTitle);
  expect(coreList.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(coreList.meta?.terminologyApplication).toBe('core');
  expect(coreList.component).toBe(terminologyListComponent);

  const webList = terminologyRoutes.find((r) => r.name === 'TranslationTermList_web') as any;
  expect(webList?.path).toBe('web/terminology/web');
  expect(webList?.meta?.resourceId).toBe('web.route.terminology_list');
  expect(webList?.meta?.terminologyApplication).toBe('web');
});

test('terminology detail route: application + id, admin defaultRoles', () => {
  const route = terminologyRoutes.find((r) => r.name === 'TranslationTermDetail') as any;
  expect(route).toBeTruthy();
  expect(route.path).toBe('web/terminology/:application/:id');
  expect(route.meta?.resourceId).toBe('web.route.terminology_detail');
  expect(route.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(route.meta?.resource?.actions).toEqual(['web.action.translation_term_edit']);
});
