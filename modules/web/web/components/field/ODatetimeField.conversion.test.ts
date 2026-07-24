// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Conversion helpers used by ODatetimeField (toView/fromView) — cover the FE hub paths
 * without mounting Element Plus (framework prefers pure-logic Vitest for coverage).
 */
import { describe, expect, it } from 'vitest';
import { formatUtcIso, userWallDateToUtc, utcToUserWallDate } from '@/web/web/utils/datetime';

describe('ODatetimeField conversion contract', () => {
  it('toView-equivalent: UTC ISO → user wall Date components', () => {
    const wall = utcToUserWallDate('2024-11-03T05:00:00.000Z', 'America/New_York');
    expect(wall).not.toBeNull();
    // Fall-back morning: 01:00 EST
    expect(wall!.getFullYear()).toBe(2024);
    expect(wall!.getMonth()).toBe(10);
    expect(wall!.getDate()).toBe(3);
    expect(wall!.getHours()).toBe(1);
  });

  it('fromView-equivalent: wall Date → UTC ISO storage', () => {
    const wall = utcToUserWallDate('2024-06-30T16:00:00.000Z', 'Asia/Shanghai')!;
    const utc = userWallDateToUtc(wall, 'Asia/Shanghai')!;
    expect(formatUtcIso(utc, 'YYYY-MM-DD[T]HH:mm:ss.SSSZ')).toBe('2024-06-30T16:00:00.000Z');
  });

  it('round-trips spring-forward New York midnight', () => {
    const iso = '2024-03-10T05:00:00.000Z';
    const wall = utcToUserWallDate(iso, 'America/New_York')!;
    expect(userWallDateToUtc(wall, 'America/New_York')!.toISOString()).toBe(iso);
  });
});
