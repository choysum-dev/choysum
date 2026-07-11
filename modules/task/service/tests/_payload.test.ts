// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  isSensitiveKey,
  maskSensitive,
  sortForEncoding,
  encodeStableJson,
  byteLength,
  truncatePreview,
  sanitizePayload,
  PAYLOAD_MAX_BYTES,
  MASK_VALUE,
} from '@/task/service/models/_payload';

test('task._payload isSensitiveKey matches hint fragments', () => {
  expect(isSensitiveKey('password')).toBe(true);
  expect(isSensitiveKey('user_password')).toBe(true);
  expect(isSensitiveKey('access_token')).toBe(true);
  expect(isSensitiveKey('Authorization')).toBe(true);
  expect(isSensitiveKey('email')).toBe(false);
  expect(isSensitiveKey('name')).toBe(false);
});

test('task._payload maskSensitive recursively masks values', () => {
  const input = {
    email: 'a@b.com',
    password: 'secret123',
    profile: { access_token: 'abc', name: 'x' },
    tokens: [{ token: 't1' }, { token: 't2' }],
  };
  const result = maskSensitive(input);
  expect(result.email).toBe('a@b.com');
  expect(result.password).toBe(MASK_VALUE);
  expect(result.profile.access_token).toBe(MASK_VALUE);
  expect(result.profile.name).toBe('x');
  expect(result.tokens).toBe(MASK_VALUE);
});

test('task._payload maskSensitive handles primitives', () => {
  expect(maskSensitive('hello')).toBe('hello');
  expect(maskSensitive(42)).toBe(42);
  expect(maskSensitive(null)).toBe(null);
  expect(maskSensitive(undefined)).toBe(undefined);
});

test('task._payload maskSensitive handles empty objects', () => {
  expect(maskSensitive({})).toEqual({});
  expect(maskSensitive([])).toEqual([]);
});

test('task._payload sortForEncoding sorts object keys', () => {
  const input = { zebra: 1, apple: 2, mango: { cherry: 3, banana: 4 } };
  const result = sortForEncoding(input);
  const keys = Object.keys(result);
  expect(keys).toEqual(['apple', 'mango', 'zebra']);
  const innerKeys = Object.keys(result.mango);
  expect(innerKeys).toEqual(['banana', 'cherry']);
});

test('task._payload sortForEncoding handles arrays and primitives', () => {
  expect(sortForEncoding([3, 1, 2])).toEqual([3, 1, 2]);
  expect(sortForEncoding('hello')).toBe('hello');
});

test('task._payload encodeStableJson produces deterministic output', () => {
  const a = encodeStableJson({ b: 2, a: 1 });
  const b = encodeStableJson({ a: 1, b: 2 });
  expect(a).toBe(b);
});

test('task._payload byteLength computes utf-8 byte count', () => {
  expect(byteLength('hello')).toBe(5);
  expect(byteLength('中文')).toBe(6);
  expect(byteLength('')).toBe(0);
});

test('task._payload truncatePreview cuts at byte boundary', () => {
  const input = 'hello世界';
  const result = truncatePreview(input, 5);
  expect(result).toBe('hello');
});

test('task._payload sanitizePayload returns masked payload', () => {
  const result = sanitizePayload({ password: 's', email: 'a@b.com' });
  expect(result.password).toBe(MASK_VALUE);
  expect(result.email).toBe('a@b.com');
});

test('task._payload sanitizePayload truncates oversized payload', () => {
  const big = { blob: 'x'.repeat(20000) };
  const result = sanitizePayload(big);
  expect(result._truncated).toBe(true);
  expect(typeof result._preview).toBe('string');
  expect((result._preview as string).length).toBeGreaterThan(0);
  expect((result._preview as string).length).toBeLessThanOrEqual(PAYLOAD_MAX_BYTES);
});

test('task._payload sanitizePayload handles empty payload', () => {
  const result = sanitizePayload({} as any);
  expect(result).toEqual({});
});
