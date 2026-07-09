// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeRefId } from '@/core/service/utils/normalization';
import { normalizeCompanyScopeKey } from '@/base/service/models/_refs';

// ---------------------------------------------------------------------------
// normalizeRefId
// ---------------------------------------------------------------------------

test('base._refs: normalizeRefId undefined → null', () => {
  expect(normalizeRefId(undefined)).toBeNull();
});

test('base._refs: normalizeRefId null → null', () => {
  expect(normalizeRefId(null)).toBeNull();
});

test('base._refs: normalizeRefId object with Id', () => {
  expect(normalizeRefId({ Id: '  ABC  ' })).toBe('ABC');
});

test('base._refs: normalizeRefId object with id', () => {
  expect(normalizeRefId({ id: 'xyz' })).toBe('xyz');
});

test('base._refs: normalizeRefId object without Id/id → null', () => {
  expect(normalizeRefId({ Name: 'X' })).toBeNull();
});

test('base._refs: normalizeRefId plain string', () => {
  expect(normalizeRefId('  hello  ')).toBe('hello');
});

test('base._refs: normalizeRefId empty string → null', () => {
  expect(normalizeRefId('   ')).toBeNull();
});

test('base._refs: normalizeRefId number → string', () => {
  expect(normalizeRefId(42)).toBe('42');
});

test('base._refs: normalizeRefId zero → "0"', () => {
  expect(normalizeRefId(0)).toBe('0');
});

// ---------------------------------------------------------------------------
// normalizeCompanyScopeKey
// ---------------------------------------------------------------------------

test('base._refs: normalizeCompanyScopeKey with valid id', () => {
  expect(normalizeCompanyScopeKey({ Id: ' C01 ' })).toBe('C01');
});

test('base._refs: normalizeCompanyScopeKey null → __GLOBAL__', () => {
  expect(normalizeCompanyScopeKey(null)).toBe('__GLOBAL__');
});

test('base._refs: normalizeCompanyScopeKey undefined → __GLOBAL__', () => {
  expect(normalizeCompanyScopeKey(undefined)).toBe('__GLOBAL__');
});

test('base._refs: normalizeCompanyScopeKey empty → __GLOBAL__', () => {
  expect(normalizeCompanyScopeKey('')).toBe('__GLOBAL__');
});
