// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import {
  getInspectedUiResourceId,
  getInspectedUiResourceRequires,
  isInspectedUiResourceRow,
  normalizeUiResourceRequires,
  selectInspectedUiResource,
} from './role_ui_requires_explain';

describe('normalizeUiResourceRequires', () => {
  it('returns empty for nullish and blank strings', () => {
    expect(normalizeUiResourceRequires(null)).toEqual([]);
    expect(normalizeUiResourceRequires(undefined)).toEqual([]);
    expect(normalizeUiResourceRequires('')).toEqual([]);
    expect(normalizeUiResourceRequires('   ')).toEqual([]);
  });

  it('throws for non-array values after string parsing', () => {
    expect(() => normalizeUiResourceRequires(42)).toThrow('invalid_ui_resource_requires');
    expect(() => normalizeUiResourceRequires({ rpc: 'x' })).toThrow('invalid_ui_resource_requires');
  });

  it('parses arrays and JSON array strings; dedupes empties', () => {
    expect(normalizeUiResourceRequires(['rpc:/auth.User/Browse', 'rpc:/auth.User/Browse', ''])).toEqual(['rpc:/auth.User/Browse']);
    expect(() => normalizeUiResourceRequires(['rpc:/auth.User/Browse', null])).toThrow('invalid_ui_resource_requires');
    expect(normalizeUiResourceRequires('["rpc:/auth.User/Update"]')).toEqual(['rpc:/auth.User/Update']);
    expect(normalizeUiResourceRequires('rpc:/auth.User/Create')).toEqual(['rpc:/auth.User/Create']);
    expect(normalizeUiResourceRequires('not-json[')).toEqual(['not-json[']);
    // JSON non-array primitives stay opaque require tokens.
    expect(normalizeUiResourceRequires('123')).toEqual(['123']);
    expect(normalizeUiResourceRequires('true')).toEqual(['true']);
    expect(normalizeUiResourceRequires('{"a":1}')).toEqual(['{"a":1}']);
  });
});

describe('selectInspectedUiResource', () => {
  it('keeps objects and clears invalid rows', () => {
    const row = { Id: '1' };
    expect(selectInspectedUiResource(row)).toBe(row);
    expect(selectInspectedUiResource(null)).toBeNull();
    expect(selectInspectedUiResource(undefined)).toBeNull();
    expect(selectInspectedUiResource('x')).toBeNull();
    expect(selectInspectedUiResource(0)).toBeNull();
  });
});

describe('inspected row helpers', () => {
  it('reads id and Requires/requires', () => {
    expect(getInspectedUiResourceId(null)).toBe('');
    expect(getInspectedUiResourceId(undefined)).toBe('');
    expect(getInspectedUiResourceId({})).toBe('');
    expect(getInspectedUiResourceId({ Id: null })).toBe('');
    expect(getInspectedUiResourceId({ Id: undefined })).toBe('');
    expect(getInspectedUiResourceId({ Id: '  abc  ' })).toBe('abc');
    expect(getInspectedUiResourceRequires(null)).toEqual([]);
    expect(getInspectedUiResourceRequires({})).toEqual([]);
    expect(getInspectedUiResourceRequires({ Requires: null, requires: ['rpc:/a/b'] })).toEqual(['rpc:/a/b']);
    expect(getInspectedUiResourceRequires({ Requires: ['rpc:/a/b'] })).toEqual(['rpc:/a/b']);
    expect(getInspectedUiResourceRequires({ requires: '["rpc:/c/d"]' })).toEqual(['rpc:/c/d']);
  });
});

describe('isInspectedUiResourceRow', () => {
  it('matches only when both sides have the same non-empty id', () => {
    expect(isInspectedUiResourceRow('', { Id: 'a' })).toBe(false);
    expect(isInspectedUiResourceRow(null as any, { Id: 'a' })).toBe(false);
    expect(isInspectedUiResourceRow(undefined as any, { Id: 'a' })).toBe(false);
    expect(isInspectedUiResourceRow('a', null)).toBe(false);
    expect(isInspectedUiResourceRow('a', 'x')).toBe(false);
    expect(isInspectedUiResourceRow('a', { Id: null })).toBe(false);
    expect(isInspectedUiResourceRow('a', { Id: undefined })).toBe(false);
    expect(isInspectedUiResourceRow('a', { Id: '  ' })).toBe(false);
    expect(isInspectedUiResourceRow('a', { Id: 'b' })).toBe(false);
    expect(isInspectedUiResourceRow('a', { Id: 'a' })).toBe(true);
    expect(isInspectedUiResourceRow('  a  ', { Id: ' a ' })).toBe(true);
  });
});
