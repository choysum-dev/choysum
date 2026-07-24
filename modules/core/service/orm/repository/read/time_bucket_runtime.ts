// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { TemporalGranularity } from '../types';
import { getRuntimeIntlApi } from '@/core/utils/env';
import moment from 'moment-timezone';

export function coerceToBucketStart(input: Date | string, granularity: TemporalGranularity, timezone?: string): Date {
  const d = input instanceof Date ? new Date(input) : new Date(input);
  if (Number.isNaN(d.getTime())) return d;

  if (!timezone) {
    return coerceUtcBucketStart(d, granularity);
  }

  // Prefer Intl when present (existing tests mock it); else moment-timezone (QuickJS / D21).
  const parts = getZonedParts(d, timezone);
  if (parts) {
    return bucketFromYmd(parts.year, parts.month, parts.day, parts.weekdayIndexIso, granularity, d);
  }

  if (moment.tz.zone(timezone)) {
    const wall = moment.utc(d.getTime()).tz(timezone);
    if (wall.isValid()) {
      const weekdayIndexIso = wall.isoWeekday() as 1 | 2 | 3 | 4 | 5 | 6 | 7;
      return bucketFromYmd(wall.year(), wall.month() + 1, wall.date(), weekdayIndexIso, granularity, d);
    }
  }

  return coerceUtcBucketStart(d, granularity);
}

function bucketFromYmd(
  y: number,
  m: number,
  day: number,
  weekdayIndexIso: 1 | 2 | 3 | 4 | 5 | 6 | 7,
  granularity: TemporalGranularity,
  fallback: Date
): Date {
  switch (granularity) {
    case 'year':
      return new Date(Date.UTC(y, 0, 1, 0, 0, 0, 0));
    case 'quarter': {
      const qm = 1 + Math.floor((m - 1) / 3) * 3;
      return new Date(Date.UTC(y, qm - 1, 1, 0, 0, 0, 0));
    }
    case 'month':
      return new Date(Date.UTC(y, m - 1, 1, 0, 0, 0, 0));
    case 'week': {
      const base = new Date(Date.UTC(y, m - 1, day, 0, 0, 0, 0));
      base.setUTCDate(base.getUTCDate() - (weekdayIndexIso - 1));
      return base;
    }
    case 'day':
      return new Date(Date.UTC(y, m - 1, day, 0, 0, 0, 0));
    default:
      return fallback;
  }
}

function coerceUtcBucketStart(d: Date, granularity: TemporalGranularity): Date {
  const y = d.getUTCFullYear();
  const m = d.getUTCMonth();
  const day = d.getUTCDate();

  switch (granularity) {
    case 'year':
      return new Date(Date.UTC(y, 0, 1, 0, 0, 0, 0));
    case 'quarter': {
      const qStart = Math.floor(m / 3) * 3;
      return new Date(Date.UTC(y, qStart, 1, 0, 0, 0, 0));
    }
    case 'month':
      return new Date(Date.UTC(y, m, 1, 0, 0, 0, 0));
    case 'week': {
      const tmp = new Date(Date.UTC(y, m, day, 0, 0, 0, 0));
      const wd = tmp.getUTCDay() === 0 ? 7 : tmp.getUTCDay();
      tmp.setUTCDate(tmp.getUTCDate() - (wd - 1));
      return tmp;
    }
    case 'day':
      return new Date(Date.UTC(y, m, day, 0, 0, 0, 0));
    default:
      return d;
  }
}

export function nextBucket(bucketStart: Date, granularity: TemporalGranularity): Date {
  const d = new Date(bucketStart.getTime());
  switch (granularity) {
    case 'year':
      d.setUTCFullYear(d.getUTCFullYear() + 1);
      return d;
    case 'quarter':
      d.setUTCMonth(d.getUTCMonth() + 3);
      return d;
    case 'month':
      d.setUTCMonth(d.getUTCMonth() + 1);
      return d;
    case 'week':
      d.setUTCDate(d.getUTCDate() + 7);
      return d;
    case 'day':
      d.setUTCDate(d.getUTCDate() + 1);
      return d;
    default:
      return d;
  }
}

export function enumerateBuckets(params: {
  start: Date | string;
  end: Date | string;
  granularity: TemporalGranularity;
  timezone?: string;
  maxBuckets?: number;
}): Date[] {
  const { start, end, granularity, timezone, maxBuckets = 10000 } = params;

  const startAligned = coerceToBucketStart(start, granularity, timezone);
  const endDate = end instanceof Date ? new Date(end) : new Date(end);

  const out: Date[] = [];
  let cur = startAligned;
  let guard = 0;

  while (cur <= endDate) {
    if (guard >= maxBuckets) break;
    out.push(new Date(cur.getTime()));
    cur = nextBucket(cur, granularity);
    guard++;
  }

  return out;
}

function getZonedParts(
  d: Date,
  timezone: string
):
  | {
      year: number;
      month: number;
      day: number;
      weekdayIndexIso: 1 | 2 | 3 | 4 | 5 | 6 | 7;
    }
  | undefined {
  const intlApi = getRuntimeIntlApi();
  if (!intlApi || typeof intlApi.DateTimeFormat !== 'function') {
    return undefined;
  }

  let fmt: Intl.DateTimeFormat;
  try {
    fmt = new intlApi.DateTimeFormat('en-US', {
      timeZone: timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      weekday: 'long',
      hour12: false,
    });
  } catch {
    return undefined;
  }

  const parts = fmt.formatToParts(d);
  let year = 0;
  let month = 0;
  let day = 0;
  let weekdayName = 'Monday';

  for (const p of parts) {
    switch (p.type) {
      case 'year':
        year = Number(p.value);
        break;
      case 'month':
        month = Number(p.value);
        break;
      case 'day':
        day = Number(p.value);
        break;
      case 'weekday':
        weekdayName = p.value;
        break;
    }
  }

  const weekdayIndexIso = ((): 1 | 2 | 3 | 4 | 5 | 6 | 7 => {
    const map: Record<string, number> = {
      Monday: 1,
      Tuesday: 2,
      Wednesday: 3,
      Thursday: 4,
      Friday: 5,
      Saturday: 6,
      Sunday: 7,
    };
    const idx = map[weekdayName] || 1;
    return idx as 1 | 2 | 3 | 4 | 5 | 6 | 7;
  })();

  return { year, month, day, weekdayIndexIso };
}
