// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';

// Avoid importing menus.ts (icon registry side effects) under QuickJS FE unit.
const menuTitle = createTranslate('meta', { scope: 'web/menu/menus' })._lt('Modules');
const catalogTitle = createTranslate('meta', { scope: 'web/menu/menus' })._lt('Catalog');

test('meta menu/route _lt titles: TermReference factory pins Modules menu src', () => {
  expect(menuTitle.src).toBe('Modules');
  expect(menuTitle.module).toBe('meta');
});

test('meta menu/route _lt titles: TermReference factory pins Catalog menu src', () => {
  expect(catalogTitle.src).toBe('Catalog');
  expect(catalogTitle.module).toBe('meta');
});

test('meta menu/route _lt titles: TermReference factory pins Catalog route src', () => {
  const routeTitle = createTranslate('meta', { scope: 'web/route/routes' })._lt('Catalog');
  expect(routeTitle.src).toBe('Catalog');
  expect(routeTitle.module).toBe('meta');
});

test('meta menu/route _lt titles: TermReference factory pins Module List route src', () => {
  const routeTitle = createTranslate('meta', { scope: 'web/route/routes' })._lt('Module List');
  expect(routeTitle.src).toBe('Module List');
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
