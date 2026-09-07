// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isGrantEveryoneWarning } from './role_record_rule_audience';

test('Access Rules field binding: grant-everyone helper stays wired', () => {
  expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: null })).toBe(true);
  expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: 'r1' })).toBe(false);
  expect(isGrantEveryoneWarning({ Kind: 'restrict', RoleId: null })).toBe(false);
});
