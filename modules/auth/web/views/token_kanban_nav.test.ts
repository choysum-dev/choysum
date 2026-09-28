// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveTokenDetailId } from './token_kanban_nav';

test('resolveTokenDetailId: returns trimmed payload Id', () => {
  expect(resolveTokenDetailId({ Id: '  tok-1  ' })).toBe('tok-1');
});

test('resolveTokenDetailId: ignores missing or blank Id', () => {
  expect(resolveTokenDetailId(undefined)).toBe('');
  expect(resolveTokenDetailId(null)).toBe('');
  expect(resolveTokenDetailId({})).toBe('');
  expect(resolveTokenDetailId({ Id: '   ' })).toBe('');
});
