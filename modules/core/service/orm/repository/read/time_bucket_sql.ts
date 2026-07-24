// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import moment from 'moment-timezone';
import type { DialectName } from '../repository_dialect';
import type { TemporalGranularity } from '../types';

/** Keep DST CASE expressions bounded (IANA histories can be huge). */
const SQLITE_TZ_CASE_FROM_YEAR = 1990;
const SQLITE_TZ_CASE_TO_YEAR = 2040;

function isUtcTimezone(timezone: string): boolean {
  const normalized = timezone.trim().toUpperCase();
  return normalized === 'UTC' || normalized === 'ETC/UTC' || normalized === 'GMT' || normalized === 'Z';
}

/**
 * Resolve a fixed UTC offset (minutes) for zones without DST.
 * Returns null when the zone observes DST.
 * Throws on invalid IANA ids.
 */
export function resolveFixedUtcOffsetMinutes(timezone: string): number | null {
  const tz = String(timezone || '').trim();
  if (!tz || !moment.tz.zone(tz)) {
    throw new Error(`Invalid IANA timezone for time bucketing: ${timezone}`);
  }
  if (isUtcTimezone(tz)) return 0;

  const jan = moment.tz([2024, 0, 1], tz).utcOffset();
  const jul = moment.tz([2024, 6, 1], tz).utcOffset();
  if (jan !== jul) return null;
  return jan;
}

export function formatSqliteUtcOffsetModifier(offsetMinutes: number): string {
  const sign = offsetMinutes >= 0 ? '+' : '-';
  const abs = Math.abs(offsetMinutes);
  const hh = String(Math.floor(abs / 60)).padStart(2, '0');
  const mm = String(abs % 60).padStart(2, '0');
  return `${sign}${hh}:${mm}`;
}

/**
 * Zone offset segments for SQL CASE generation.
 * `utcOffsetMinutes` matches `moment#utcOffset()` (e.g. EST = -300).
 * Segment covers `[prevUntilMs, untilMs)`.
 */
export type ZoneOffsetSegment = {
  untilMs: number;
  utcOffsetMinutes: number;
};

export function listZoneOffsetSegments(
  timezone: string,
  fromYear = SQLITE_TZ_CASE_FROM_YEAR,
  toYear = SQLITE_TZ_CASE_TO_YEAR
): ZoneOffsetSegment[] {
  const tz = String(timezone || '').trim();
  const zone = moment.tz.zone(tz);
  if (!zone) {
    throw new Error(`Invalid IANA timezone for time bucketing: ${timezone}`);
  }
  if (isUtcTimezone(tz)) {
    return [{ untilMs: Number.POSITIVE_INFINITY, utcOffsetMinutes: 0 }];
  }

  const fromMs = Date.UTC(fromYear, 0, 1);
  const toMs = Date.UTC(toYear, 0, 1);
  const segments: ZoneOffsetSegment[] = [];

  for (let i = 0; i < zone.untils.length; i++) {
    const untilRaw = zone.untils[i];
    const untilMs = untilRaw === Infinity ? Number.POSITIVE_INFINITY : untilRaw;
    const prevMs = i === 0 ? Number.NEGATIVE_INFINITY : zone.untils[i - 1];
    if (untilMs <= fromMs) continue;
    if (prevMs >= toMs) break;
    // moment-timezone zone.offsets use Date#getTimezoneOffset sign (west positive).
    const utcOffsetMinutes = -zone.offsets[i];
    segments.push({
      untilMs: untilMs === Number.POSITIVE_INFINITY ? toMs + 1 : Math.min(untilMs, toMs + 1),
      utcOffsetMinutes,
    });
    if (untilMs > toMs) break;
  }

  if (!segments.length) {
    segments.push({ untilMs: toMs + 1, utcOffsetMinutes: moment.tz(fromMs, tz).utcOffset() });
  }
  return segments;
}

/**
 * Apply the same UTC→wall adjustment SQLite/MSSQL SQL would apply (for fixture parity).
 */
export function applySqlTimezoneAdjustment(instant: Date | string, timezone: string): Date {
  const d = instant instanceof Date ? new Date(instant) : new Date(instant);
  if (Number.isNaN(d.getTime())) return d;
  const tz = String(timezone || '').trim();
  if (!tz || isUtcTimezone(tz)) return d;

  const fixed = resolveFixedUtcOffsetMinutes(tz);
  const offsetMinutes = fixed != null ? fixed : moment.utc(d.getTime()).tz(tz).utcOffset();
  // datetime(col, '+08:00') adds the offset to the stored UTC wall → local wall as naive UTC label.
  return new Date(d.getTime() + offsetMinutes * 60_000);
}

function buildSqliteOffsetAdjustedColumn(column: unknown, timezone: string): unknown {
  const fixed = resolveFixedUtcOffsetMinutes(timezone);
  if (fixed != null) {
    if (fixed === 0) return column;
    const modLit = sql.raw(`'${formatSqliteUtcOffsetModifier(fixed)}'`);
    return sql`datetime(${column}, ${modLit})`;
  }

  const segments = listZoneOffsetSegments(timezone);
  const elseMod = formatSqliteUtcOffsetModifier(segments[segments.length - 1]?.utcOffsetMinutes ?? 0);
  let acc: unknown = sql`datetime(${column}, ${sql.raw(`'${elseMod}'`)})`;
  for (let i = segments.length - 2; i >= 0; i--) {
    const untilIso = moment.utc(segments[i].untilMs).format('YYYY-MM-DD HH:mm:ss');
    const mod = formatSqliteUtcOffsetModifier(segments[i].utcOffsetMinutes);
    acc = sql`CASE WHEN datetime(${column}) < datetime(${sql.raw(`'${untilIso}'`)}) THEN datetime(${column}, ${sql.raw(`'${mod}'`)}) ELSE ${acc} END`;
  }
  return acc;
}

function buildMssqlOffsetAdjustedColumn(column: unknown, timezone: string): unknown {
  const fixed = resolveFixedUtcOffsetMinutes(timezone);
  if (fixed != null) {
    if (fixed === 0) return column;
    return sql`DATEADD(minute, ${sql.raw(String(fixed))}, ${column})`;
  }

  const segments = listZoneOffsetSegments(timezone);
  const elseOffset = segments[segments.length - 1]?.utcOffsetMinutes ?? 0;
  let acc: unknown = sql`DATEADD(minute, ${sql.raw(String(elseOffset))}, ${column})`;
  for (let i = segments.length - 2; i >= 0; i--) {
    const untilIso = moment.utc(segments[i].untilMs).format('YYYY-MM-DDTHH:mm:ss');
    acc = sql`CASE WHEN ${column} < ${sql.raw(`'${untilIso}'`)} THEN DATEADD(minute, ${sql.raw(String(segments[i].utcOffsetMinutes))}, ${column}) ELSE ${acc} END`;
  }
  return acc;
}

function applyTimezone(dialect: DialectName, column: unknown, timezone?: string): unknown {
  if (!timezone) return column;
  const tz = String(timezone).trim();
  if (!tz) return column;

  const tzLit = sql.raw(`'${tz.replace(/'/g, "''")}'`);

  switch (dialect) {
    case 'postgres':
      return sql`(${column}) AT TIME ZONE ${tzLit}`;
    case 'mysql':
      // Requires mysql timezone tables; CONVERT_TZ returns NULL when tables are missing.
      return sql`CONVERT_TZ(${column}, @@session.time_zone, ${tzLit})`;
    case 'sqlite': {
      if (isUtcTimezone(tz)) return column;
      if (!moment.tz.zone(tz)) {
        throw new Error(`Invalid IANA timezone for time bucketing: ${timezone}`);
      }
      return buildSqliteOffsetAdjustedColumn(column, tz);
    }
    case 'mssql': {
      if (isUtcTimezone(tz)) return column;
      if (!moment.tz.zone(tz)) {
        throw new Error(`Invalid IANA timezone for time bucketing: ${timezone}`);
      }
      return buildMssqlOffsetAdjustedColumn(column, tz);
    }
    default:
      return column;
  }
}

export function buildTimeBucketExpr(dialect: DialectName, column: unknown, granularity: TemporalGranularity, timezone?: string): unknown {
  const ts = applyTimezone(dialect, column, timezone);

  switch (dialect) {
    case 'postgres': {
      switch (granularity) {
        case 'year':
          return sql`DATE_TRUNC('year', ${ts})`;
        case 'quarter':
          return sql`DATE_TRUNC('quarter', ${ts})`;
        case 'month':
          return sql`DATE_TRUNC('month', ${ts})`;
        case 'week':
          return sql`DATE_TRUNC('week', ${ts})`;
        case 'day':
          return sql`DATE_TRUNC('day', ${ts})`;
        default:
          throw new Error(`Unsupported granularity for postgres: ${granularity}`);
      }
    }

    case 'mysql': {
      switch (granularity) {
        case 'year':
          return sql`STR_TO_DATE(DATE_FORMAT(${ts}, '%Y-01-01 00:00:00'), '%Y-%m-%d %H:%i:%s')`;
        case 'quarter':
          return sql`STR_TO_DATE(CONCAT(YEAR(${ts}), '-', LPAD(1 + 3 * (QUARTER(${ts}) - 1), 2, '0'), '-01 00:00:00'), '%Y-%m-%d %H:%i:%s')`;
        case 'month':
          return sql`STR_TO_DATE(DATE_FORMAT(${ts}, '%Y-%m-01 00:00:00'), '%Y-%m-%d %H:%i:%s')`;
        case 'week':
          return sql`DATE_SUB(DATE(${ts}), INTERVAL WEEKDAY(${ts}) DAY)`;
        case 'day':
          return sql`STR_TO_DATE(DATE_FORMAT(${ts}, '%Y-%m-%d 00:00:00'), '%Y-%m-%d %H:%i:%s')`;
        default:
          throw new Error(`Unsupported granularity for mysql: ${granularity}`);
      }
    }

    case 'sqlite': {
      switch (granularity) {
        case 'year':
          return sql`DATE(${ts}, 'start of year')`;
        case 'quarter':
          return sql`
            DATE(
              STRFTIME('%Y-%m-01', ${ts}),
              CASE
                WHEN CAST(STRFTIME('%m', ${ts}) AS INTEGER) BETWEEN 1 AND 3 THEN 'start of month'
                WHEN CAST(STRFTIME('%m', ${ts}) AS INTEGER) BETWEEN 4 AND 6 THEN '+3 months', 'start of month', '-3 months'
                WHEN CAST(STRFTIME('%m', ${ts}) AS INTEGER) BETWEEN 7 AND 9 THEN '+6 months', 'start of month', '-6 months'
                ELSE '+9 months', 'start of month', '-9 months'
              END
            )
          `;
        case 'month':
          return sql`DATE(${ts}, 'start of month')`;
        case 'week':
          return sql`DATE(${ts}, 'weekday 1', '-7 days')`;
        case 'day':
          return sql`DATE(${ts}, 'start of day')`;
        default:
          throw new Error(`Unsupported granularity for sqlite: ${granularity}`);
      }
    }

    case 'mssql': {
      switch (granularity) {
        case 'year':
          return sql`DATEFROMPARTS(YEAR(${ts}), 1, 1)`;
        case 'quarter':
          return sql`DATEFROMPARTS(YEAR(${ts}), 1 + 3 * (DATEPART(QUARTER, ${ts}) - 1), 1)`;
        case 'month':
          return sql`DATEFROMPARTS(YEAR(${ts}), MONTH(${ts}), 1)`;
        case 'week':
          return sql`DATEADD(week, DATEDIFF(week, 0, ${ts}), 0)`;
        case 'day':
          return sql`CAST(${ts} as date)`;
        default:
          throw new Error(`Unsupported granularity for mssql: ${granularity}`);
      }
    }

    default:
      throw new Error(`Unsupported dialect: ${dialect}`);
  }
}
