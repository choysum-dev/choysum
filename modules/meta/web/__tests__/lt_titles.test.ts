// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';

// Avoid importing menus.ts (Element Plus icons) under QuickJS FE unit.
const menuTitle = createTranslate('meta', { scope: 'web/menu/menus' })._lt('Module Management');

test('meta menu/route _lt titles: TermReference factory pins Module Management menu src', () => {
  expect(menuTitle.src).toBe('Module Management');
  expect(menuTitle.module).toBe('meta');
});

test('meta menu/route _lt titles: TermReference factory pins Module Management route src', () => {
  const routeTitle = createTranslate('meta', { scope: 'web/route/routes' })._lt('Module Management');
  expect(routeTitle.src).toBe('Module Management');
  expect(routeTitle.module).toBe('meta');
});

test('meta menu/route _lt titles: TermReference factory pins External IDs titles', () => {
  const menuTitle = createTranslate('meta', { scope: 'web/menu/menus' })._lt('External IDs');
  expect(menuTitle.src).toBe('External IDs');
  expect(menuTitle.module).toBe('meta');

  const listTitle = createTranslate('meta', { scope: 'web/route/routes' })._lt('External IDs');
  expect(listTitle.src).toBe('External IDs');

  const detailTitle = createTranslate('meta', { scope: 'web/route/routes' })._lt('External ID');
  expect(detailTitle.src).toBe('External ID');
});
