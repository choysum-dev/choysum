// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';

// Avoid importing menus.ts (Element Plus icons) under QuickJS FE unit.
const menuTitle = createTranslate('base', { scope: 'web/menu/menus' })._lt('Master Data');

test('base menu/route _lt titles: TermReference factory pins Master Data src', () => {
  expect(menuTitle.src).toBe('Master Data');
  expect(menuTitle.module).toBe('base');
});

test('base menu/route _lt titles: TermReference factory pins Company List src', () => {
  const companyListTitle = createTranslate('base', { scope: 'web/route/routes' })._lt('Company List');
  expect(companyListTitle.src).toBe('Company List');
  expect(companyListTitle.module).toBe('base');
});
