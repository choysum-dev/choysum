// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';

// Avoid importing menus.ts (Element Plus icons) under QuickJS FE unit.
const menuTitle = createTranslate('partner', { scope: 'web/menu/menus' })._lt('Partner Management');

test('partner menu/route _lt titles: TermReference factory pins Partner Management src', () => {
  expect(menuTitle.src).toBe('Partner Management');
  expect(menuTitle.module).toBe('partner');
});

test('partner menu/route _lt titles: TermReference factory pins Partner List src', () => {
  const listTitle = createTranslate('partner', { scope: 'web/route/routes' })._lt('Partner List');
  expect(listTitle.src).toBe('Partner List');
});
