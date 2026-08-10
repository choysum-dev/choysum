// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeScopeKey } from '@/web/service/models/_scope_key';

test('normalizeScopeKey returns empty for blank input', () => {
  expect(normalizeScopeKey('')).toBe('');
  expect(normalizeScopeKey(null)).toBe('');
  expect(normalizeScopeKey(undefined)).toBe('');
  expect(normalizeScopeKey('   ')).toBe('');
});

test('normalizeScopeKey strips query and hash independently', () => {
  expect(normalizeScopeKey('/web/users?q=1')).toBe('/web/users');
  expect(normalizeScopeKey('/web/users#section')).toBe('/web/users');
  expect(normalizeScopeKey('/web/users/?q=1#hash')).toBe('/web/users');
});

test('normalizeScopeKey collapses slashes and keeps root', () => {
  expect(normalizeScopeKey('\\web\\users\\')).toBe('/web/users');
  expect(normalizeScopeKey('web/users///')).toBe('/web/users');
  expect(normalizeScopeKey('/')).toBe('/');
});

test('normalizeScopeKey replaces numeric and opaque segments', () => {
  expect(normalizeScopeKey('/web/partners/42/edit')).toBe('/web/partners/:id/edit');
  expect(normalizeScopeKey('/web/partners/abc123def456ghi7/form')).toBe('/web/partners/:id/form');
  expect(normalizeScopeKey('/web/partners/AbC_def-0123456789/x')).toBe('/web/partners/:id/x');
  expect(normalizeScopeKey('/web/partners/short/form')).toBe('/web/partners/short/form');
});
