// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isGrantEveryoneWarning } from './role_record_rule_audience';

test('Access Rules admin coverage: grant-everyone helper', () => {
  expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: null })).toBe(true);
  expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: 'role-1' })).toBe(false);
  expect(isGrantEveryoneWarning({ Kind: 'restrict', RoleId: '' })).toBe(false);
});
