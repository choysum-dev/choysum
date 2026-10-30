// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  resolvePropertyDefinitionDetailApplication,
  resolvePropertyDefinitionListApplication,
} from './property_definition_page_bind';

test('resolvePropertyDefinitionListApplication prefers meta then param then path', () => {
  expect(
    resolvePropertyDefinitionListApplication({
      meta: { propertyDefinitionApplication: 'auth' },
      params: { application: 'web' },
      path: '/web/property-definitions/base',
    })
  ).toBe('auth');
  expect(
    resolvePropertyDefinitionListApplication({
      meta: {},
      params: { application: 'partner' },
      path: '/web/property-definitions/base',
    })
  ).toBe('partner');
  expect(
    resolvePropertyDefinitionListApplication({
      meta: {},
      params: {},
      path: '/web/property-definitions/sandbox',
    })
  ).toBe('sandbox');
  expect(resolvePropertyDefinitionListApplication({ meta: {}, params: {}, path: '/x' })).toBe('');
});

test('resolvePropertyDefinitionDetailApplication reads params.application', () => {
  expect(resolvePropertyDefinitionDetailApplication({ params: { application: 'meta' } })).toBe(
    'meta'
  );
  expect(resolvePropertyDefinitionDetailApplication({ params: { application: '  ' } })).toBe('');
});
