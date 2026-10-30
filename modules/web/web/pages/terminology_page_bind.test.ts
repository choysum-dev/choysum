// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  resolveTerminologyDetailApplication,
  resolveTerminologyListApplication,
} from './terminology_page_bind';

test('resolveTerminologyListApplication prefers meta then params then path', () => {
  expect(
    resolveTerminologyListApplication({
      meta: { terminologyApplication: 'auth' },
      params: { application: 'base' },
      path: '/web/terminology/partner',
    })
  ).toBe('auth');
  expect(
    resolveTerminologyListApplication({
      meta: {},
      params: { application: 'base' },
      path: '/web/terminology/partner',
    })
  ).toBe('base');
  expect(
    resolveTerminologyListApplication({
      meta: {},
      params: {},
      path: '/web/terminology/partner',
    })
  ).toBe('partner');
  expect(resolveTerminologyListApplication({ meta: {}, params: {}, path: '/other' })).toBe('');
  expect(resolveTerminologyListApplication({ meta: { terminologyApplication: '  ' }, params: {}, path: '' })).toBe('');
});

test('resolveTerminologyDetailApplication trims params.application', () => {
  expect(resolveTerminologyDetailApplication({ params: { application: 'web' } })).toBe('web');
  expect(resolveTerminologyDetailApplication({ params: { application: '  ' } })).toBe('');
  expect(resolveTerminologyDetailApplication({ params: {} })).toBe('');
  expect(resolveTerminologyDetailApplication({})).toBe('');
});
