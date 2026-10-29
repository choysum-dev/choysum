// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { terminologyRoutes } from './terminologyRoutes';

const translationsTitle = createTranslate('web', { scope: 'web/route/terminologyRoutes' })._lt('Translations');

test('terminology list routes: concrete per-app paths for UI_VAL_006 menu match', () => {
  const coreList = terminologyRoutes.find((r) => r.name === 'TranslationTermList_core') as any;
  expect(coreList).toBeTruthy();
  expect(coreList.path).toBe('web/terminology/core');
  expect(coreList.meta?.resourceId).toBe('web.route.terminology_core_list');
  expect(coreList.meta?.pageTitleText).toEqual(translationsTitle);
  expect(coreList.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(coreList.meta?.terminologyApplication).toBe('core');

  const authList = terminologyRoutes.find((r) => r.name === 'TranslationTermList_auth') as any;
  expect(authList?.path).toBe('web/terminology/auth');
});

test('terminology detail route: application + id, admin defaultRoles', () => {
  const route = terminologyRoutes.find((r) => r.name === 'TranslationTermDetail') as any;
  expect(route).toBeTruthy();
  expect(route.path).toBe('web/terminology/:application/:id');
  expect(route.meta?.resourceId).toBe('web.route.terminology_detail');
  expect(route.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(route.meta?.resource?.actions).toEqual(['web.action.translation_term_edit']);
});
