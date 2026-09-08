// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { getResourceDeclarationFromMeta } from '@/core/web/resource';
import { languageRoutes } from './routes';

const terminologyTitle = createTranslate('base', { scope: 'web/route/routes' })._lt('Terminology Editor');

test('terminology editor route: declares role-gated meta without loading the Vue component', () => {
  const route = languageRoutes.find(r => r.name === 'TerminologyEditor') as any;
  expect(route).toBeTruthy();
  expect(route.path).toBe('base/terminology');
  expect(route.meta?.resourceId).toBe('base.route.terminology_editor');
  expect(route.meta?.pageTitleText).toEqual(terminologyTitle);
  expect(route.meta?.requiresAuth).toBe(true);
  expect(getResourceDeclarationFromMeta(route.meta)?.defaultRoles).toEqual(['terminology.editor']);

  expect(typeof route.component).toBe('function');
});
