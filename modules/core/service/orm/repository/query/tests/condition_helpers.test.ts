// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { andAll, isEmptyCondition, toRepoCondition } from '..';

test('repository condition helpers reuse repository condition helpers for empty and nested And conditions', () => {
  expect(isEmptyCondition(undefined as any)).toBe(true);
  expect(isEmptyCondition({ And: [[] as any] } as any)).toBe(true);
  expect(toRepoCondition(undefined as any)).toEqual([]);

  const merged = andAll(
    [] as any,
    {
      And: [['Name', '=', 'demo'] as any, [] as any],
    } as any,
    ['Status', '=', 'draft'] as any
  );

  expect(merged).toEqual({
    And: [
      ['Name', '=', 'demo'],
      ['Status', '=', 'draft'],
    ],
  });
});
