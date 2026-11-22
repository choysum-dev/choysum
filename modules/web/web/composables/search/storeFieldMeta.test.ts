// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildFilterConditionFieldMeta, readStoreFieldMeta } from './storeFieldMeta';

test('readStoreFieldMeta returns field entry or empty object', () => {
  expect(readStoreFieldMeta(undefined, 'Name')).toEqual({});
  expect(readStoreFieldMeta({ metadata: {} }, 'Name')).toEqual({});
  expect(readStoreFieldMeta({ metadata: { fields: {} } }, '')).toEqual({});
  expect(
    readStoreFieldMeta(
      { metadata: { fields: { Amount: { type: 'monetary', string: 'Amount' } } } },
      'Amount'
    )
  ).toEqual({ type: 'monetary', string: 'Amount' });
  expect(
    readStoreFieldMeta({ metadata: { fields: { Name: { type: 'varchar' } } } }, 'Missing')
  ).toEqual({});
});

test('buildFilterConditionFieldMeta overlays type fallback and clears readonly', () => {
  expect(buildFilterConditionFieldMeta(undefined, 'Name', 'varchar')).toEqual({
    type: 'varchar',
    isReadonly: false,
  });
  expect(
    buildFilterConditionFieldMeta(
      { metadata: { fields: { Amount: { type: 'monetary', string: '金额', isReadonly: true } } } },
      'Amount',
      'varchar'
    )
  ).toEqual({
    type: 'monetary',
    string: '金额',
    isReadonly: false,
  });
});
