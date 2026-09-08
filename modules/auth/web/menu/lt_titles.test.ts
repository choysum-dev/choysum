// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';

// Avoid importing menus.ts (Element Plus icons) under QuickJS FE unit.
const menuTitle = createTranslate('auth', { scope: 'web/menu/menus' })._lt('Access Control');

test('auth menu/route _lt titles: TermReference factory pins Access Control src', () => {
  expect(menuTitle.src).toBe('Access Control');
  expect(menuTitle.module).toBe('auth');
});

test('auth menu/route _lt titles: TermReference factory pins Login src', () => {
  const loginTitle = createTranslate('auth', { scope: 'web/route/routes' })._lt('Login');
  expect(loginTitle.src).toBe('Login');
});
