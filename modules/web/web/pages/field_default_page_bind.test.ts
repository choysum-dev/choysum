// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  resolveFieldDefaultDetailApplication,
  resolveFieldDefaultListApplication,
} from './field_default_page_bind';

test('resolveFieldDefaultListApplication prefers meta then param then path', () => {
  expect(
    resolveFieldDefaultListApplication({
      meta: { fieldDefaultApplication: 'auth' },
      params: { application: 'web' },
      path: '/web/field-defaults/base',
    })
  ).toBe('auth');
  expect(
    resolveFieldDefaultListApplication({
      meta: {},
      params: { application: 'partner' },
      path: '/web/field-defaults/base',
    })
  ).toBe('partner');
  expect(
    resolveFieldDefaultListApplication({
      meta: {},
      params: {},
      path: '/web/field-defaults/sandbox',
    })
  ).toBe('sandbox');
  expect(resolveFieldDefaultListApplication({ meta: {}, params: {}, path: '/x' })).toBe('');
});

test('resolveFieldDefaultDetailApplication reads params.application', () => {
  expect(resolveFieldDefaultDetailApplication({ params: { application: 'meta' } })).toBe('meta');
  expect(resolveFieldDefaultDetailApplication({ params: { application: '  ' } })).toBe('');
});
