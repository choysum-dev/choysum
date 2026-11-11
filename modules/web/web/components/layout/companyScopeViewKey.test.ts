// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { companyScopeViewKey } from './companyScopeViewKey';

test('companyScopeViewKey: sorts enabled ids and ignores order', () => {
  expect(companyScopeViewKey('a', ['b', 'a'])).toBe('a|a,b');
  expect(companyScopeViewKey('a', ['a', 'b'])).toBe('a|a,b');
});

test('companyScopeViewKey: includes active when missing from enabled', () => {
  expect(companyScopeViewKey('a', ['b'])).toBe('a|a,b');
});

test('companyScopeViewKey: empty scope is stable', () => {
  expect(companyScopeViewKey('', null)).toBe('|');
  expect(companyScopeViewKey(undefined, undefined)).toBe('|');
});
