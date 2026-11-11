// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  clearCompanyScopeView,
  publishCompanyScopeView,
  useCompanyScopeViewKey,
} from './companyScopeView';

test('publishCompanyScopeView: drives the remount key', () => {
  clearCompanyScopeView();
  const key = useCompanyScopeViewKey();
  expect(key.value).toBe('|');

  publishCompanyScopeView('a', ['b', 'a']);
  expect(key.value).toBe('a|a,b');

  publishCompanyScopeView('b', ['b']);
  expect(key.value).toBe('b|b');

  clearCompanyScopeView();
  expect(key.value).toBe('|');
});
