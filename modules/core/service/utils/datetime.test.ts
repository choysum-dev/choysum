// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  parseISODate,
  toDate,
  isIanaTimezone,
  listIanaTimezoneSelection,
  parseTimezoneOffsetMinutes,
  dayRange,
  businessToday,
  businessYesterday,
  wallClockRangeToUtc,
} from '@/core/service/utils/datetime';
import { withContext } from '@/core/service/runtime/context';

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

// ---------------------------------------------------------------------------
// isIanaTimezone
// ---------------------------------------------------------------------------

test('isIanaTimezone validates known zones', () => {
  expect(isIanaTimezone('Asia/Shanghai')).toBe(true);
  expect(isIanaTimezone('UTC')).toBe(true);
  expect(isIanaTimezone('America/New_York')).toBe(true);
  expect(isIanaTimezone('')).toBe(false);
  expect(isIanaTimezone()).toBe(false);
  expect(isIanaTimezone('Not/A_Zone')).toBe(false);
});

// ---------------------------------------------------------------------------
// listIanaTimezoneSelection
// ---------------------------------------------------------------------------

test('listIanaTimezoneSelection returns IANA ids as bare selection items', () => {
  const items = listIanaTimezoneSelection();
  expect(items.length).toBeGreaterThan(100);
  expect(items.every(item => item.value && item.value === item.label)).toBe(true);
  expect(items.some(item => item.value === 'UTC')).toBe(true);
  expect(items.some(item => item.value === 'Asia/Shanghai')).toBe(true);
});

test('listIanaTimezoneSelection caches the mapped array', () => {
  const items = listIanaTimezoneSelection();
  expect(items).toBe(listIanaTimezoneSelection());
  expect(Object.isFrozen(items)).toBe(true);
  expect(Object.isFrozen(items[0])).toBe(true);
});

// ---------------------------------------------------------------------------
// parseTimezoneOffsetMinutes
// ---------------------------------------------------------------------------

test('parseTimezoneOffsetMinutes parses UTC variants', () => {
  expect(parseTimezoneOffsetMinutes('UTC')).toBe(0);
  expect(parseTimezoneOffsetMinutes('GMT')).toBe(0);
  expect(parseTimezoneOffsetMinutes('Z')).toBe(0);
});

test('parseTimezoneOffsetMinutes parses positive offset', () => {
  expect(parseTimezoneOffsetMinutes('+08:00')).toBe(480);
  expect(parseTimezoneOffsetMinutes('+8')).toBe(480);
  expect(parseTimezoneOffsetMinutes('+05:30')).toBe(330);
});

test('parseTimezoneOffsetMinutes parses negative offset', () => {
  expect(parseTimezoneOffsetMinutes('-05:00')).toBe(-300);
  expect(parseTimezoneOffsetMinutes('-12')).toBe(-720);
});

test('parseTimezoneOffsetMinutes returns undefined for invalid', () => {
  expect(parseTimezoneOffsetMinutes('')).toBe(undefined);
  expect(parseTimezoneOffsetMinutes()).toBe(undefined);
  expect(parseTimezoneOffsetMinutes('invalid')).toBe(undefined);
  expect(parseTimezoneOffsetMinutes('+15:00')).toBe(undefined);
  expect(parseTimezoneOffsetMinutes('+08:60')).toBe(undefined);
});

// ---------------------------------------------------------------------------
// dayRange / businessToday / wallClockRangeToUtc
// ---------------------------------------------------------------------------

test('dayRange maps Asia/Shanghai calendar day to UTC half-open bounds', () => {
  const { start, end } = dayRange('2024-07-01', 'Asia/Shanghai');
  expect(start.toISOString()).toBe('2024-06-30T16:00:00.000Z');
  expect(end.toISOString()).toBe('2024-07-01T16:00:00.000Z');
});

test('dayRange handles America/New_York spring-forward 23h day', () => {
  const { start, end } = dayRange('2024-03-10', 'America/New_York');
  expect(end.getTime() - start.getTime()).toBe(23 * 60 * 60 * 1000);
  expect(start.toISOString()).toBe('2024-03-10T05:00:00.000Z');
  expect(end.toISOString()).toBe('2024-03-11T04:00:00.000Z');
});

test('dayRange handles America/New_York fall-back 25h day', () => {
  const { start, end } = dayRange('2024-11-03', 'America/New_York');
  expect(end.getTime() - start.getTime()).toBe(25 * 60 * 60 * 1000);
  expect(start.toISOString()).toBe('2024-11-03T04:00:00.000Z');
  expect(end.toISOString()).toBe('2024-11-04T05:00:00.000Z');
});

test('dayRange rejects invalid timezone', () => {
  let error: unknown;
  try {
    dayRange('2024-07-01', 'Not/A_Zone');
  } catch (err) {
    error = err;
  }
  expect(error instanceof Error).toBe(true);
  expect(String((error as Error).message)).toContain('Invalid IANA');
});

test('businessToday with UTC matches fixed clock', () => {
  const now = new Date('2024-07-01T15:30:00.000Z');
  expect(businessToday('UTC', now)).toBe('2024-07-01');
  expect(businessYesterday('UTC', now)).toBe('2024-06-30');
});

test('businessToday uses companyTz from context when tz omitted', () => {
  const now = new Date('2024-07-01T15:30:00.000Z'); // still 2024-07-01 in Shanghai (+8)
  const day = withContext({ companyTz: 'Asia/Shanghai' } as any, () => businessToday(undefined, now));
  expect(day).toBe('2024-07-01');

  const late = new Date('2024-07-01T16:30:00.000Z'); // 2024-07-02 00:30 Shanghai
  const next = withContext({ companyTz: 'Asia/Shanghai' } as any, () => businessToday(undefined, late));
  expect(next).toBe('2024-07-02');
});

test('wallClockRangeToUtc converts half-open wall clocks before Search', () => {
  const { start, end } = wallClockRangeToUtc('2024-07-01 00:00:00', '2024-07-02 00:00:00', 'Asia/Shanghai');
  expect(start.toISOString()).toBe('2024-06-30T16:00:00.000Z');
  expect(end.toISOString()).toBe('2024-07-01T16:00:00.000Z');
});

test('wallClockRangeToUtc matches dayRange for full calendar days', () => {
  const day = dayRange('2024-03-10', 'America/New_York');
  const wall = wallClockRangeToUtc('2024-03-10 00:00:00', '2024-03-11 00:00:00', 'America/New_York');
  expect(wall.start.toISOString()).toBe(day.start.toISOString());
  expect(wall.end.toISOString()).toBe(day.end.toISOString());
});
