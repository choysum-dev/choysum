// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from 'vitest';
import {
  formatUtcInTimeZone,
  formatUtcIso,
  getUserTimeZone,
  setUserTimeZoneResolver,
  userWallDateToUtc,
  utcToUserWallDate,
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

  it('formats UTC in user timezone for display', () => {
    expect(formatUtcInTimeZone('2024-06-30T16:00:00.000Z', 'YYYY-MM-DD HH:mm:ss', 'Asia/Shanghai')).toBe(
      '2024-07-01 00:00:00'
    );
    expect(formatUtcInTimeZone('2024-06-30T16:00:00.000Z', 'YYYY-MM-DD HH:mm:ss', 'America/New_York')).toBe(
      '2024-06-30 12:00:00'
    );
  });

  it('formatUtcIso keeps Z storage', () => {
    expect(formatUtcIso('2024-07-01T00:00:00.000Z', 'YYYY-MM-DD[T]HH:mm:ss.SSSZ')).toBe('2024-07-01T00:00:00.000Z');
  });

  it('dayRange matches Asia/Shanghai half-open UTC bounds', async () => {
    const { dayRange } = await import('./datetime');
    const { start, end } = dayRange('2024-07-01', 'Asia/Shanghai');
    expect(start.toISOString()).toBe('2024-06-30T16:00:00.000Z');
    expect(end.toISOString()).toBe('2024-07-01T16:00:00.000Z');
  });

  it('dayRange handles America/New_York spring-forward 23h day', async () => {
    const { dayRange } = await import('./datetime');
    const { start, end } = dayRange('2024-03-10', 'America/New_York');
    expect(end.getTime() - start.getTime()).toBe(23 * 60 * 60 * 1000);
  });
});
