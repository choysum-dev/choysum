// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { normalizeScopeKey } from './scopeKey';

describe('normalizeScopeKey', () => {
  it('returns empty for blank input', () => {
    expect(normalizeScopeKey('')).toBe('');
    expect(normalizeScopeKey(null)).toBe('');
    expect(normalizeScopeKey(undefined)).toBe('');
    expect(normalizeScopeKey('   ')).toBe('');
  });

  it('strips query and hash, collapses slashes, trims trailing slash', () => {
    expect(normalizeScopeKey('/web/users/?q=1#hash')).toBe('/web/users');
    expect(normalizeScopeKey('web/users///')).toBe('/web/users');
    expect(normalizeScopeKey('\\web\\users\\')).toBe('/web/users');
    expect(normalizeScopeKey('/')).toBe('/');
  });

  it('replaces numeric and opaque id segments with :id', () => {
    expect(normalizeScopeKey('/web/partners/42/edit')).toBe('/web/partners/:id/edit');
    expect(normalizeScopeKey('/web/partners/abc123def456ghi7/form')).toBe('/web/partners/:id/form');
    expect(normalizeScopeKey('/web/partners/short/form')).toBe('/web/partners/short/form');
  });
});
