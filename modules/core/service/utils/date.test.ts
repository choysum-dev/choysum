// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { parseISODate, toDate } from '@/core/service/utils/date';

// ---------------------------------------------------------------------------
// parseISODate
// ---------------------------------------------------------------------------

test('parseISODate returns current date for undefined', () => {
  const before = Date.now();
  const result = parseISODate(undefined);
  const after = Date.now();
  expect(result.getTime()).toBeGreaterThanOrEqual(before - 1000);
  expect(result.getTime()).toBeLessThanOrEqual(after + 1000);
});

test('parseISODate returns current date for empty string', () => {
  const before = Date.now();
  const result = parseISODate('');
  const after = Date.now();
  expect(result.getTime()).toBeGreaterThanOrEqual(before - 1000);
  expect(result.getTime()).toBeLessThanOrEqual(after + 1000);
});

test('parseISODate parses a valid ISO string', () => {
  const result = parseISODate('2024-01-15T10:30:00.000Z');
  expect(result.toISOString()).toBe('2024-01-15T10:30:00.000Z');
});

test('parseISODate returns current date for invalid string', () => {
  const before = Date.now();
  const result = parseISODate('not-a-date');
  const after = Date.now();
  // Should fall back to now, so the result is close to current time
  expect(result.getTime()).toBeGreaterThanOrEqual(before - 1000);
  expect(result.getTime()).toBeLessThanOrEqual(after + 1000);
});

// ---------------------------------------------------------------------------
// toDate
// ---------------------------------------------------------------------------

test('toDate returns the Date instance for a valid Date', () => {
  const d = new Date('2024-06-15T12:00:00Z');
  expect(toDate(d)).toBe(d);
});

test('toDate returns undefined for an invalid Date instance', () => {
  const d = new Date('invalid');
  expect(toDate(d)).toBe(undefined);
});

test('toDate parses a valid ISO string', () => {
  const result = toDate('2024-06-15T12:00:00Z');
  expect(result instanceof Date).toBe(true);
  expect(result!.toISOString()).toBe('2024-06-15T12:00:00.000Z');
});

test('toDate returns undefined for an invalid string', () => {
  expect(toDate('not-a-date')).toBe(undefined);
});

test('toDate returns undefined for null', () => {
  expect(toDate(null)).toBe(undefined);
});

test('toDate returns undefined for undefined', () => {
  expect(toDate(undefined)).toBe(undefined);
});
