// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  parseTimezoneOffsetMinutes,
  isIanaTimezone,
  parseCronField,
  parseCronExpr,
  nextCronTime,
  computeNextRunAt,
  normalizeTimezone,
  applyNextRunPreview,
} from '@/task/service/models/_cron';

test('task._cron parseTimezoneOffsetMinutes parses UTC', () => {
  expect(parseTimezoneOffsetMinutes('UTC')).toBe(0);
  expect(parseTimezoneOffsetMinutes('GMT')).toBe(0);
  expect(parseTimezoneOffsetMinutes('Z')).toBe(0);
});

test('task._cron parseTimezoneOffsetMinutes parses positive offset', () => {
  expect(parseTimezoneOffsetMinutes('+08:00')).toBe(480);
  expect(parseTimezoneOffsetMinutes('+8')).toBe(480);
});

test('task._cron parseTimezoneOffsetMinutes parses negative offset', () => {
  expect(parseTimezoneOffsetMinutes('-05:00')).toBe(-300);
});

test('task._cron parseTimezoneOffsetMinutes returns undefined for invalid', () => {
  expect(parseTimezoneOffsetMinutes('')).toBe(undefined);
  expect(parseTimezoneOffsetMinutes('invalid')).toBe(undefined);
  expect(parseTimezoneOffsetMinutes('+15:00')).toBe(undefined);
  expect(parseTimezoneOffsetMinutes('+08:60')).toBe(undefined);
});

test('task._cron isIanaTimezone validates known zones', () => {
  expect(isIanaTimezone('Asia/Shanghai')).toBe(true);
  expect(isIanaTimezone('UTC')).toBe(true);
  expect(isIanaTimezone('America/New_York')).toBe(true);
  expect(isIanaTimezone('')).toBe(false);
  expect(isIanaTimezone('Not/A_Zone')).toBe(false);
});

test('task._cron parseCronField parses wildcard', () => {
  const result = parseCronField('*', 0, 5);
  expect(result).toBeTruthy();
  if (result) {
    expect(result[0]).toBe(true);
    expect(result[5]).toBe(true);
    expect(result[6]).toBeFalsy();
  }
});

test('task._cron parseCronField parses step', () => {
  const result = parseCronField('*/2', 0, 5);
  expect(result).toBeTruthy();
  if (result) {
    expect(result[0]).toBe(true);
    expect(result[2]).toBe(true);
    expect(result[4]).toBe(true);
    expect(result[1]).toBeFalsy();
  }
});

test('task._cron parseCronField parses range', () => {
  const result = parseCronField('2-4', 0, 5);
  expect(result).toBeTruthy();
  if (result) {
    expect(result[2]).toBe(true);
    expect(result[3]).toBe(true);
    expect(result[4]).toBe(true);
    expect(result[1]).toBeFalsy();
  }
});

test('task._cron parseCronField parses list', () => {
  const result = parseCronField('1,3,5', 0, 5);
  expect(result).toBeTruthy();
  if (result) {
    expect(result[1]).toBe(true);
    expect(result[3]).toBe(true);
    expect(result[5]).toBe(true);
    expect(result[2]).toBeFalsy();
  }
});

test('task._cron parseCronField rejects out-of-range', () => {
  expect(parseCronField('99', 0, 5)).toBe(null);
  expect(parseCronField('60', 0, 59)).toBe(null);
});

test('task._cron parseCronExpr parses valid expression', () => {
  const result = parseCronExpr('0 0 * * *');
  expect(result).toBeTruthy();
  if (result) {
    expect(result.minutes[0]).toBe(true);
    expect(result.hours[0]).toBe(true);
  }
});

test('task._cron parseCronExpr rejects short expressions', () => {
  expect(parseCronExpr('0 0 * *')).toBe(null);
});

test('task._cron parseCronExpr rejects invalid fields', () => {
  expect(parseCronExpr('60 0 * * *')).toBe(null);
});

test('task._cron nextCronTime finds next match', () => {
  const fields = parseCronExpr('0 0 * * *');
  expect(fields).toBeTruthy();
  if (fields) {
    const base = new Date('2026-01-01T12:30:00Z');
    const next = nextCronTime(base, fields);
    expect(next).toBeTruthy();
    if (next) {
      // next midnight after 12:30 should be the following day
      expect(next.getTime()).toBeGreaterThan(base.getTime());
      // minute should be 0 (midnight)
      expect(next.getUTCMinutes()).toBe(0);
    }
  }
});

test('task._cron computeNextRunAt returns undefined for empty expr', () => {
  expect(computeNextRunAt({ CronExpr: '' })).toBe(undefined);
});

test('task._cron computeNextRunAt returns undefined for invalid expr', () => {
  expect(computeNextRunAt({ CronExpr: 'not valid' })).toBe(undefined);
});

test('task._cron computeNextRunAt computes next run for daily schedule', () => {
  const base = new Date('2026-01-01T00:00:00Z');
  const result = computeNextRunAt({ CronExpr: '0 8 * * *', Timezone: 'UTC' }, base);
  expect(result).toBeTruthy();
  if (result) {
    expect(result.getTime()).toBeGreaterThan(base.getTime());
    expect(result.getUTCMinutes()).toBe(0);
  }
});

test('task._cron normalizeTimezone validates IANA', () => {
  expect(normalizeTimezone('UTC')).toBe('UTC');
  expect(normalizeTimezone('Asia/Shanghai')).toBe('Asia/Shanghai');
  expect(() => normalizeTimezone('')).toThrow('timezone is required');
  expect(() => normalizeTimezone('Not/A_Zone')).toThrow('invalid timezone');
});

test('task._cron applyNextRunPreview fills missing NextRunAt', () => {
  const schedule = { CronExpr: '0 0 * * *', Timezone: 'UTC', NextRunAt: undefined as Date | undefined };
  const result = applyNextRunPreview(schedule);
  expect(result.NextRunAt).toBeTruthy();
});

test('task._cron applyNextRunPreview keeps existing NextRunAt', () => {
  const existing = new Date('2026-06-01T00:00:00Z');
  const schedule = { CronExpr: '0 0 * * *', Timezone: 'UTC', NextRunAt: existing };
  const result = applyNextRunPreview(schedule);
  expect(result.NextRunAt).toBe(existing);
});
