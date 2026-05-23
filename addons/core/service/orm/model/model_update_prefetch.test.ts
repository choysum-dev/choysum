// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizePrefetchedRows } from './model_update_prefetch';

test('normalizePrefetchedRows returns empty array for undefined input', () => {
  expect(normalizePrefetchedRows(undefined)).toEqual([]);
});

test('normalizePrefetchedRows returns original rows when provided', () => {
  const rows = [{ Id: '1' }, { Id: '2' }];
  expect(normalizePrefetchedRows(rows)).toBe(rows);
});
