// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveUserMenuDisplayName, resolveUserMenuMeta } from './user_menu_display_name';

test('resolveUserMenuDisplayName: applies last_first template', () => {
  expect(
    resolveUserMenuDisplayName({
      firstName: '步科',
      lastName: '王',
      personNameFormat: '{LastName}{FirstName}',
    }),
  ).toBe('王步科');
});

test('resolveUserMenuDisplayName: applies first_last default when template omitted', () => {
  expect(
    resolveUserMenuDisplayName({
      firstName: '步科',
      lastName: '王',
    }),
  ).toBe('步科 王');
});

test('resolveUserMenuDisplayName: falls back to FullName then Username', () => {
  expect(resolveUserMenuDisplayName({ fullName: 'Ada Lovelace', username: 'ada' })).toBe('Ada Lovelace');
  expect(resolveUserMenuDisplayName({ username: 'ada', email: 'a@b.c' })).toBe('ada');
  expect(resolveUserMenuDisplayName({ email: 'a@b.c' })).toBe('a@b.c');
});

test('resolveUserMenuMeta: username and email on one line; skips primary dupes', () => {
  expect(resolveUserMenuMeta({ email: 'a@b.c', username: 'ada' })).toBe('ada · a@b.c');
  expect(resolveUserMenuMeta({ username: 'ada' })).toBe('ada');
  expect(resolveUserMenuMeta({ email: 'a@b.c' })).toBe('a@b.c');
  expect(resolveUserMenuMeta({ email: '  ', username: 'ada' })).toBe('ada');
  expect(resolveUserMenuMeta({ email: 'a@b.c', username: 'ada', primaryLabel: 'ada' })).toBe('a@b.c');
  expect(resolveUserMenuMeta({ username: 'ada', primaryLabel: 'ada' })).toBe('');
  expect(resolveUserMenuMeta({ email: 'ada', username: 'ada' })).toBe('ada');
  expect(resolveUserMenuMeta({})).toBe('');
});
