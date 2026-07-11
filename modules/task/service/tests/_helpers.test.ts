// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { clampLimit } from '@/task/service/models/_helpers';

test('task._helpers clampLimit returns fallback for undefined', () => {
  expect(clampLimit(undefined, 50, 500)).toBe(50);
});

test('task._helpers clampLimit returns fallback for null', () => {
  expect(clampLimit(null as any, 50, 500)).toBe(50);
});

test('task._helpers clampLimit returns fallback for zero', () => {
  expect(clampLimit(0, 50, 500)).toBe(50);
});

test('task._helpers clampLimit returns fallback for negative', () => {
  expect(clampLimit(-1, 50, 500)).toBe(50);
});

test('task._helpers clampLimit returns value within range', () => {
  expect(clampLimit(100, 50, 500)).toBe(100);
});

test('task._helpers clampLimit caps at max', () => {
  expect(clampLimit(1000, 50, 500)).toBe(500);
});

test('task._helpers clampLimit returns exactly max', () => {
  expect(clampLimit(500, 50, 500)).toBe(500);
});

test('task._helpers clampLimit returns fallback for non-number', () => {
  expect(clampLimit('abc' as any, 50, 500)).toBe(50);
});

test('task._helpers clampLimit respects custom fallback and max', () => {
  expect(clampLimit(200, 20, 100)).toBe(100);
  expect(clampLimit(undefined, 20, 100)).toBe(20);
});
