// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import type { DialectName } from '../repository_dialect';
import type { TemporalGranularity } from '../types';

function applyTimezone(dialect: DialectName, column: unknown, timezone?: string): unknown {
  if (!timezone) return column;
  const tzLit = sql.raw(`'${String(timezone).replace(/'/g, "''")}'`);

  switch (dialect) {
    case 'postgres':
      return sql`(${column}) AT TIME ZONE ${tzLit}`;
    case 'mysql':
      return sql`CONVERT_TZ(${column}, @@session.time_zone, ${tzLit})`;
    case 'sqlite':
    case 'mssql':
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
