// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sortForEncoding, encodeStableJson } from '@/core/service/utils/serialization';

test('core.serialization sortForEncoding sorts object keys', () => {
  const input = { zebra: 1, apple: 2, mango: { cherry: 3, banana: 4 } };
  const result = sortForEncoding(input);
  const keys = Object.keys(result);
  expect(keys).toEqual(['apple', 'mango', 'zebra']);
  const innerKeys = Object.keys(result.mango);
  expect(innerKeys).toEqual(['banana', 'cherry']);
});

test('core.serialization sortForEncoding handles arrays and primitives', () => {
  expect(sortForEncoding([3, 1, 2])).toEqual([3, 1, 2]);
  expect(sortForEncoding('hello')).toBe('hello');
  expect(sortForEncoding(42)).toBe(42);
  expect(sortForEncoding(null)).toBe(null);
  expect(sortForEncoding(undefined)).toBe(undefined);
});

test('core.serialization encodeStableJson produces deterministic output', () => {
  const a = encodeStableJson({ b: 2, a: 1 });
  const b = encodeStableJson({ a: 1, b: 2 });
  expect(a).toBe(b);
});

test('core.serialization encodeStableJson handles primitives', () => {
  expect(encodeStableJson(1)).toBe('1');
  expect(encodeStableJson('hello')).toBe('"hello"');
});
