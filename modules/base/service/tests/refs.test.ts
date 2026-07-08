// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { asRefId, normalizeCompanyScopeKey } from '@/base/service/models/_refs';

// ---------------------------------------------------------------------------
// asRefId
// ---------------------------------------------------------------------------

test('base._refs: asRefId undefined → undefined', () => {
  expect(asRefId(undefined)).toBeUndefined();
});

test('base._refs: asRefId null → null', () => {
  expect(asRefId(null)).toBeNull();
});

test('base._refs: asRefId object with Id', () => {
  expect(asRefId({ Id: '  ABC  ' })).toBe('ABC');
});

test('base._refs: asRefId object with id', () => {
  expect(asRefId({ id: 'xyz' })).toBe('xyz');
});

test('base._refs: asRefId object without Id/id → null', () => {
  expect(asRefId({ Name: 'X' })).toBeNull();
});

test('base._refs: asRefId plain string', () => {
  expect(asRefId('  hello  ')).toBe('hello');
});

test('base._refs: asRefId empty string → null', () => {
  expect(asRefId('   ')).toBeNull();
});

test('base._refs: asRefId number → string', () => {
  expect(asRefId(42)).toBe('42');
});

test('base._refs: asRefId zero → "0"', () => {
  expect(asRefId(0)).toBe('0');
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
