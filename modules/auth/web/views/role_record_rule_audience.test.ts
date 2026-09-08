// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isGrantEveryoneWarning } from './role_record_rule_audience';

test('isGrantEveryoneWarning: is false without a draft', () => {
  expect(isGrantEveryoneWarning(null)).toBe(false);
  expect(isGrantEveryoneWarning(undefined)).toBe(false);
});

test('isGrantEveryoneWarning: is false when Kind is not grant', () => {
  expect(isGrantEveryoneWarning({ Kind: 'restrict', RoleId: null })).toBe(false);
  expect(isGrantEveryoneWarning({ Kind: 'RESTRICT', RoleId: '' })).toBe(false);
});

test('isGrantEveryoneWarning: treats missing Kind as grant and empty Role as everyone', () => {
  expect(isGrantEveryoneWarning({})).toBe(true);
  expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: null })).toBe(true);
  expect(isGrantEveryoneWarning({ Kind: 'Grant', RoleId: '' })).toBe(true);
});

test('isGrantEveryoneWarning: handles RoleId object and string forms', () => {
  expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: { Id: '' } })).toBe(true);
  expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: {} })).toBe(true);
  expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: { Id: 'role-1' } })).toBe(false);
  expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: 'role-1' })).toBe(false);
  expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: '   ' })).toBe(true);
});
