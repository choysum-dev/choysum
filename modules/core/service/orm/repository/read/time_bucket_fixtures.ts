// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Cross-dialect time-bucket fixtures (D11 / scenario #7).
 * Expected keys are calendar labels at UTC midnight from coerceToBucketStart (runtime / PG semantic).
 */

import type { TemporalGranularity } from '../types';
import type { DialectName } from '../repository_dialect';
import { coerceToBucketStart } from './time_bucket_runtime';
import { applySqlTimezoneAdjustment } from './time_bucket_sql';

export type TimeBucketFixture = {
  name: string;
  instant: string;
  timezone: string;
  granularity: TemporalGranularity;
  /** Expected bucket start ISO (calendar label at UTC midnight). */
  expected: string;
};

/** Shared fixtures: Shanghai (fixed offset) + New York (DST spring/fall). */
export const TIME_BUCKET_CROSS_DIALECT_FIXTURES: TimeBucketFixture[] = [
  {
    name: 'Shanghai cross-midnight → next local day',
    instant: '2026-05-17T23:30:00.000Z',
    timezone: 'Asia/Shanghai',
    granularity: 'day',
    expected: '2026-05-18T00:00:00.000Z',
  },
  {
    name: 'Shanghai same civil day',
    instant: '2026-05-17T10:00:00.000Z',
    timezone: 'Asia/Shanghai',
    granularity: 'day',
    expected: '2026-05-17T00:00:00.000Z',
  },
  {
    name: 'Shanghai month bucket',
    instant: '2026-02-17T18:00:00.000Z',
    timezone: 'Asia/Shanghai',
    granularity: 'month',
    expected: '2026-02-01T00:00:00.000Z',
  },
  {
    name: 'New York before spring-forward (EST)',
    // 2024-03-10 06:30Z = 01:30 EST (spring-forward at 07:00Z)
    instant: '2024-03-10T06:30:00.000Z',
    timezone: 'America/New_York',
    granularity: 'day',
    expected: '2024-03-10T00:00:00.000Z',
  },
  {
    name: 'New York after spring-forward (EDT)',
    instant: '2024-03-10T07:30:00.000Z',
    timezone: 'America/New_York',
    granularity: 'day',
    expected: '2024-03-10T00:00:00.000Z',
  },
  {
    name: 'New York fall-back morning (second 01:xx EST)',
    // 2024-11-03 06:30Z = 01:30 EST after fall-back (05:00Z)
    instant: '2024-11-03T06:30:00.000Z',
    timezone: 'America/New_York',
    granularity: 'day',
    expected: '2024-11-03T00:00:00.000Z',
  },
  {
    name: 'New York week bucket (ISO week start Monday)',
    instant: '2024-03-13T15:00:00.000Z',
    timezone: 'America/New_York',
    granularity: 'week',
    expected: '2024-03-11T00:00:00.000Z',
  },
  {
    name: 'UTC identity day',
    instant: '2024-07-01T15:30:00.000Z',
    timezone: 'UTC',
    granularity: 'day',
    expected: '2024-07-01T00:00:00.000Z',
  },
];

/**
 * Simulate dialect day/month/week bucket starts after SQL timezone adjustment.
 * Postgres/MySQL: trust runtime (AT TIME ZONE / CONVERT_TZ ≡ IANA).
 * SQLite/MSSQL: apply the same offset/CASE adjustment then UTC-calendar truncate.
 */
export function simulateDialectBucketStart(
  dialect: DialectName,
  instant: string,
  granularity: TemporalGranularity,
  timezone: string
): Date {
  if (dialect === 'postgres' || dialect === 'mysql') {
    return coerceToBucketStart(instant, granularity, timezone);
  }

  if (dialect === 'sqlite' || dialect === 'mssql') {
    if (!timezone || timezone.toUpperCase() === 'UTC') {
      return coerceToBucketStart(instant, granularity);
    }
    // Match SQL: shift to local wall as naive UTC components, then truncate like DATE()/CAST.
    const adjusted = applySqlTimezoneAdjustment(instant, timezone);
    return coerceToBucketStart(adjusted.toISOString(), granularity);
  }

  return coerceToBucketStart(instant, granularity, timezone);
}
