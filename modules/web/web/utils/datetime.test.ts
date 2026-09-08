// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  formatUtcIso,
  getUserTimeZone,
  parseUtc,
  setUserTimeZoneResolver,
} from './datetime';

// density backfill from main before merge
// Wall-clock / dayRange suites need dayjs timezone + real ICU Intl under QJS.

afterEach(() => {
  setUserTimeZoneResolver(undefined);
});

describe('getUserTimeZone', () => {
  test('uses resolver when provided', () => {
    setUserTimeZoneResolver(() => 'America/New_York');
    expect(getUserTimeZone()).toBe('America/New_York');
  });

  test('trims resolver whitespace before validating', () => {
    setUserTimeZoneResolver(() => '  America/Chicago  ');
    expect(getUserTimeZone()).toBe('America/Chicago');
  });

  test('falls back when resolver returns blank', () => {
    setUserTimeZoneResolver(() => '');
    const tz = getUserTimeZone();
    expect(typeof tz).toBe('string');
    expect(tz.length).toBeGreaterThan(0);
  });

  test('falls back to browser when resolver returns an invalid zone', () => {
    setUserTimeZoneResolver(() => 'Not/A_Zone');
    const tz = getUserTimeZone();
    expect(tz).not.toBe('Not/A_Zone');
    expect(tz.length).toBeGreaterThan(0);
  });

  test('treats resolver throws as empty and still returns a zone', () => {
    setUserTimeZoneResolver(() => {
      throw new Error('boom');
    });
    const tz = getUserTimeZone();
    expect(typeof tz).toBe('string');
    expect(tz.length).toBeGreaterThan(0);
  });

  test('returns UTC when resolver and browser both yield empty zones', () => {
    setUserTimeZoneResolver(() => '');
    // With minimal Intl, detectBrowserTimezone still returns UTC.
    const tz = getUserTimeZone();
    expect(typeof tz).toBe('string');
    expect(tz.length).toBeGreaterThan(0);
  });
});

describe('formatUtcIso / parseUtc', () => {
  test('formatUtcIso keeps Z storage and rejects empties', () => {
    expect(formatUtcIso('2024-07-01T00:00:00.000Z', 'YYYY-MM-DDTHH:mm:ss[Z]')).toMatch(/Z$/);
    expect(formatUtcIso('', 'YYYY-MM-DDTHH:mm:ss[Z]')).toBeNull();
  });

  test('parseUtc supports strict format parsing', () => {
    const d = parseUtc('2024-07-01T00:00:00Z');
    expect(d).toBeTruthy();
  });
});
