// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from 'vitest';
import {
  formatUtcInTimeZone,
  formatUtcIso,
  getUserTimeZone,
  parseUtc,
  setUserTimeZoneResolver,
  userWallDateToUtc,
  utcToUserWallDate,
  dayRange,
} from './datetime';

afterEach(() => {
  setUserTimeZoneResolver(undefined);
});

describe('getUserTimeZone', () => {
  it('uses resolver when provided', () => {
    setUserTimeZoneResolver(() => 'America/New_York');
    expect(getUserTimeZone()).toBe('America/New_York');
  });

  it('falls back to browser when resolver returns an invalid zone', () => {
    setUserTimeZoneResolver(() => 'Not/A_Zone');
    const tz = getUserTimeZone();
    expect(tz).not.toBe('Not/A_Zone');
    expect(typeof tz).toBe('string');
    expect(tz.length).toBeGreaterThan(0);
  });

  it('treats resolver throws as empty and still returns a zone', () => {
    setUserTimeZoneResolver(() => {
      throw new Error('store unavailable');
    });
    expect(typeof getUserTimeZone()).toBe('string');
    expect(getUserTimeZone().length).toBeGreaterThan(0);
  });
});

describe('utc ↔ user wall', () => {
  it('maps a fixed UTC instant to Asia/Shanghai wall components', () => {
    const wall = utcToUserWallDate('2024-06-30T16:00:00.000Z', 'Asia/Shanghai');
    expect(wall).not.toBeNull();
    expect(wall!.getFullYear()).toBe(2024);
    expect(wall!.getMonth()).toBe(6); // July
    expect(wall!.getDate()).toBe(1);
    expect(wall!.getHours()).toBe(0);
    expect(wall!.getMinutes()).toBe(0);
  });

  it('maps America/New_York wall back to the same UTC instant', () => {
    const utcIso = '2024-03-10T05:00:00.000Z'; // New York spring-forward midnight
    const wall = utcToUserWallDate(utcIso, 'America/New_York');
    const back = userWallDateToUtc(wall!, 'America/New_York');
    expect(back!.toISOString()).toBe(utcIso);
  });

  it('returns null for empty / invalid wall conversions', () => {
    expect(utcToUserWallDate(null, 'UTC')).toBeNull();
    expect(utcToUserWallDate('', 'UTC')).toBeNull();
    expect(userWallDateToUtc(null, 'UTC')).toBeNull();
    expect(userWallDateToUtc(new Date('invalid'), 'UTC')).toBeNull();
  });

  it('formats UTC in user timezone for display', () => {
    expect(formatUtcInTimeZone('2024-06-30T16:00:00.000Z', 'YYYY-MM-DD HH:mm:ss', 'Asia/Shanghai')).toBe(
      '2024-07-01 00:00:00'
    );
    expect(formatUtcInTimeZone('2024-06-30T16:00:00.000Z', 'YYYY-MM-DD HH:mm:ss', 'America/New_York')).toBe(
      '2024-06-30 12:00:00'
    );
    expect(formatUtcInTimeZone(null, 'YYYY-MM-DD', 'UTC')).toBe('');
    expect(formatUtcInTimeZone(Date.parse('2024-06-30T16:00:00.000Z'), 'YYYY-MM-DD HH:mm:ss', 'UTC')).toBe(
      '2024-06-30 16:00:00'
    );
  });

  it('formatUtcIso keeps Z storage and rejects empties', () => {
    expect(formatUtcIso('2024-07-01T00:00:00.000Z', 'YYYY-MM-DD[T]HH:mm:ss.SSSZ')).toBe('2024-07-01T00:00:00.000Z');
    expect(formatUtcIso(null, 'YYYY-MM-DD[T]HH:mm:ssZ')).toBeNull();
    expect(formatUtcIso('', 'YYYY-MM-DD[T]HH:mm:ssZ')).toBeNull();
  });

  it('parseUtc supports strict format parsing', () => {
    expect(parseUtc('2024-07-01T12:00:00Z').isValid()).toBe(true);
    expect(parseUtc('2024-07-01 12:00:00', 'YYYY-MM-DD HH:mm:ss', true).isValid()).toBe(true);
  });

  it('dayRange matches Asia/Shanghai half-open UTC bounds', () => {
    const { start, end } = dayRange('2024-07-01', 'Asia/Shanghai');
    expect(start.toISOString()).toBe('2024-06-30T16:00:00.000Z');
    expect(end.toISOString()).toBe('2024-07-01T16:00:00.000Z');
  });

  it('dayRange handles America/New_York spring-forward 23h day', () => {
    const { start, end } = dayRange('2024-03-10', 'America/New_York');
    expect(end.getTime() - start.getTime()).toBe(23 * 60 * 60 * 1000);
  });

  it('dayRange accepts Date input and rejects invalid values', () => {
    // 10:00Z = 18:00 Asia/Shanghai on Jul 1 → calendar day 2024-07-01
    const { start } = dayRange(new Date('2024-07-01T10:00:00.000Z'), 'Asia/Shanghai');
    expect(start.toISOString()).toBe('2024-06-30T16:00:00.000Z');
    expect(() => dayRange('bad', 'UTC')).toThrow(/Invalid date/);
    expect(() => dayRange(new Date('invalid'), 'UTC')).toThrow(/Invalid date/);
  });
});
