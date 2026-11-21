// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { keepAliveScopeKey, stickyLeafMenuId } from './keepAliveScopeKey';

test('keepAliveScopeKey joins company and leaf menu', () => {
  expect(keepAliveScopeKey('a|b', 'base.menu.company')).toBe('a|b::base.menu.company');
  expect(keepAliveScopeKey('', '')).toBe('::');
  expect(keepAliveScopeKey('co', '  menu  ')).toBe('co::menu');
  expect(keepAliveScopeKey('co', null)).toBe('co::');
  expect(keepAliveScopeKey(undefined, undefined)).toBe('::');
});

test('stickyLeafMenuId keeps previous on null/blank', () => {
  expect(stickyLeafMenuId('', 'leaf.a')).toBe('leaf.a');
  expect(stickyLeafMenuId('leaf.a', null)).toBe('leaf.a');
  expect(stickyLeafMenuId('leaf.a', '')).toBe('leaf.a');
  expect(stickyLeafMenuId('leaf.a', '  ')).toBe('leaf.a');
  expect(stickyLeafMenuId('leaf.a', 'leaf.b')).toBe('leaf.b');
});
