// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isIanaTimezone, parseTimezoneOffsetMinutes } from '@/core/service/utils/timezone';

test('core.timezone isIanaTimezone validates known zones', () => {
  expect(isIanaTimezone('Asia/Shanghai')).toBe(true);
  expect(isIanaTimezone('UTC')).toBe(true);
  expect(isIanaTimezone('America/New_York')).toBe(true);
  expect(isIanaTimezone('')).toBe(false);
  expect(isIanaTimezone()).toBe(false);
  expect(isIanaTimezone('Not/A_Zone')).toBe(false);
});

test('core.timezone parseTimezoneOffsetMinutes parses UTC variants', () => {
  expect(parseTimezoneOffsetMinutes('UTC')).toBe(0);
  expect(parseTimezoneOffsetMinutes('GMT')).toBe(0);
  expect(parseTimezoneOffsetMinutes('Z')).toBe(0);
});

test('core.timezone parseTimezoneOffsetMinutes parses positive offset', () => {
  expect(parseTimezoneOffsetMinutes('+08:00')).toBe(480);
  expect(parseTimezoneOffsetMinutes('+8')).toBe(480);
  expect(parseTimezoneOffsetMinutes('+05:30')).toBe(330);
});

test('core.timezone parseTimezoneOffsetMinutes parses negative offset', () => {
  expect(parseTimezoneOffsetMinutes('-05:00')).toBe(-300);
  expect(parseTimezoneOffsetMinutes('-12')).toBe(-720);
});

test('core.timezone parseTimezoneOffsetMinutes returns undefined for invalid', () => {
  expect(parseTimezoneOffsetMinutes('')).toBe(undefined);
  expect(parseTimezoneOffsetMinutes()).toBe(undefined);
  expect(parseTimezoneOffsetMinutes('invalid')).toBe(undefined);
  expect(parseTimezoneOffsetMinutes('+15:00')).toBe(undefined);
  expect(parseTimezoneOffsetMinutes('+08:60')).toBe(undefined);
});
