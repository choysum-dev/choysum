// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { clearExclusive } from './clear_exclusive';

test('clearExclusive nulls listed keys only', () => {
  const row = { MetaServiceId: 'svc', MetaModelId: 'mdl', LogicalModelName: 'Role', Mode: 'allow' };
  clearExclusive(row, ['MetaServiceId', 'MetaModelId']);
  expect(row.MetaServiceId).toBeNull();
  expect(row.MetaModelId).toBeNull();
  expect(row.LogicalModelName).toBe('Role');
  expect(row.Mode).toBe('allow');
});

test('clearExclusive accepts empty key list', () => {
  const row = { Id: 'x' };
  clearExclusive(row, []);
  expect(row.Id).toBe('x');
});
