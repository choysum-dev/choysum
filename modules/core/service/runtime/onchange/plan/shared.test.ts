// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { chunk, isObject, uniq } from './shared';

test('onchange plan shared isObject and uniq behave as expected', () => {
  expect(isObject({ a: 1 })).toBe(true);
  expect(isObject(null)).toBe(false);
  expect(isObject(1 as any)).toBe(false);

  expect(uniq([1, 2, 2, 3, 1])).toEqual([1, 2, 3]);
});

test('onchange plan shared chunk handles non-array and empty inputs', () => {
  expect(chunk('bad-input' as any, 2)).toEqual([]);
  expect(chunk([], 2)).toEqual([]);
});

test('onchange plan shared chunk handles non-positive size and normal slicing', () => {
  expect(chunk([1, 2, 3], 0)).toEqual([[1, 2, 3]]);
  expect(chunk([1, 2, 3], -1)).toEqual([[1, 2, 3]]);
  expect(chunk([1, 2, 3, 4, 5], 2)).toEqual([[1, 2], [3, 4], [5]]);
});
