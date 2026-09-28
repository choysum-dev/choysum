// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveListRowRecordId } from './list_row_nav';

test('resolveListRowRecordId: reads Id from a raw row', () => {
  expect(resolveListRowRecordId({ Id: '  role-1  ' })).toBe('role-1');
});

test('resolveListRowRecordId: unwraps { row } payloads', () => {
  expect(resolveListRowRecordId({ row: { Id: 'tok-9' }, index: 0 })).toBe('tok-9');
});

test('resolveListRowRecordId: blank or missing Id fails closed', () => {
  expect(resolveListRowRecordId(null)).toBe('');
  expect(resolveListRowRecordId(undefined)).toBe('');
  expect(resolveListRowRecordId({})).toBe('');
  expect(resolveListRowRecordId({ Id: '   ' })).toBe('');
  expect(resolveListRowRecordId({ row: {} })).toBe('');
});
