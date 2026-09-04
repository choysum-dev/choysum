// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import moment from 'moment-timezone';
import { createTranslate } from '@/core/service/i18n';
import { isIanaTimezone, parseTimezoneOffsetMinutes } from '@/core/service/utils/datetime';

const { _t } = createTranslate('task');

/**
 * Parsed cron fields used while computing preview run times.
 */
export type CronFields = {
  minutes: Record<number, true>;
  hours: Record<number, true>;
  dom: Record<number, true>;
  domAny: boolean;
  month: Record<number, true>;
  dow: Record<number, true>;
  dowAny: boolean;
};

/**
 * Parses a single cron field into an allowed-value set.
 */
export function parseCronField(input: string, min: number, max: number): Record<number, true> | null {
  const set: Record<number, true> = {};
  if (input === '*') {
    for (let i = min; i <= max; i += 1) set[i] = true;
    return set;
  }
  const parts = input.split(',');
  for (const raw of parts) {
    const part = raw.trim();
    if (!part) continue;
    if (part.startsWith('*/')) {
      const step = Number(part.slice(2));
      if (!Number.isInteger(step) || step <= 0) return null;
      for (let i = min; i <= max; i += step) set[i] = true;
      continue;
    }
    if (part.includes('-')) {
      const [a, b] = part.split('-', 2);
      const start = Number(a);
      const end = Number(b);
      if (!Number.isInteger(start) || !Number.isInteger(end)) return null;
      if (start < min || end > max || start > end) return null;
      for (let i = start; i <= end; i += 1) set[i] = true;
      continue;
    }
    const val = Number(part);
    if (!Number.isInteger(val) || val < min || val > max) return null;
    set[val] = true;
  }
  return set;
}

/**
 * Parses a five-field cron expression into lookup sets.
 */
export function parseCronExpr(expr: string): CronFields | null {
  const parts = expr.trim().split(/\s+/g);
  if (parts.length !== 5) return null;
  const domRaw = parts[2];
  const dowRaw = parts[4];
  const minutes = parseCronField(parts[0], 0, 59);
  const hours = parseCronField(parts[1], 0, 23);
  const dom = parseCronField(domRaw, 1, 31);
  const month = parseCronField(parts[3], 1, 12);
  const dow = parseCronField(dowRaw, 0, 6);
  if (!minutes || !hours || !dom || !month || !dow) return null;
  return { minutes, hours, dom, domAny: domRaw === '*', month, dow, dowAny: dowRaw === '*' };
}

function cronDayMatches(fields: CronFields, dom: number, dow: number): boolean {
  if (fields.domAny && fields.dowAny) return true;
  if (fields.domAny) return Boolean(fields.dow[dow]);
  if (fields.dowAny) return Boolean(fields.dom[dom]);
  return Boolean(fields.dom[dom] || fields.dow[dow]);
}

/**
 * Computes the next matching cron time in local Date space.
 */
export function nextCronTime(from: Date, fields: CronFields): Date | undefined {
  const cur = new Date(from.getTime());
  cur.setSeconds(0, 0);
  cur.setMinutes(cur.getMinutes() + 1);
  const limit = new Date(from.getTime());
  limit.setFullYear(limit.getFullYear() + 1);
  while (cur.getTime() <= limit.getTime()) {
    const month = cur.getMonth() + 1;
    if (!fields.month[month]) {
      cur.setMonth(cur.getMonth() + 1, 1);
      cur.setHours(0, 0, 0, 0);
      continue;
    }
    const dom = cur.getDate();
    const dow = cur.getDay();
    if (!cronDayMatches(fields, dom, dow)) {
      cur.setDate(cur.getDate() + 1);
      cur.setHours(0, 0, 0, 0);
      continue;
    }
    const hours = cur.getHours();
    if (!fields.hours[hours]) {
      cur.setHours(cur.getHours() + 1, 0, 0, 0);
      continue;
    }
    const minutes = cur.getMinutes();
    if (!fields.minutes[minutes]) {
      cur.setMinutes(cur.getMinutes() + 1, 0, 0);
      continue;
    }
    return new Date(cur.getTime());
  }
  return undefined;
}

/**
 * Computes the next matching cron time in a timezone-aware moment instance.
 */
export function nextCronMoment(from: moment.Moment, fields: CronFields): moment.Moment | undefined {
  const cur = from.clone().second(0).millisecond(0).add(1, 'minute');
  const limit = from.clone().add(1, 'year');
  while (cur.isSameOrBefore(limit)) {
    const month = cur.month() + 1;
    if (!fields.month[month]) {
      cur.add(1, 'month').date(1).hour(0).minute(0).second(0).millisecond(0);
      continue;
    }
    const dom = cur.date();
    const dow = cur.day();
    if (!cronDayMatches(fields, dom, dow)) {
      cur.add(1, 'day').hour(0).minute(0).second(0).millisecond(0);
      continue;
    }
    const hours = cur.hour();
    if (!fields.hours[hours]) {
      cur.add(1, 'hour').minute(0).second(0).millisecond(0);
      continue;
    }
    const minutes = cur.minute();
    if (!fields.minutes[minutes]) {
      cur.add(1, 'minute');
      continue;
    }
    return cur.clone();
  }
  return undefined;
}

/**
 * Computes the next run time preview for a schedule.
 */
export function computeNextRunAt(schedule: { CronExpr?: string; Timezone?: string }, baseTime?: Date): Date | undefined {
  const expr = (schedule.CronExpr ?? '').trim();
  if (!expr) return undefined;
  const fields = parseCronExpr(expr);
  if (!fields) return undefined;

  const base = baseTime ?? new Date();
  const tz = schedule.Timezone?.trim();
  if (tz && isIanaTimezone(tz)) {
    const baseTz = moment.tz(base, tz);
    const nextTz = nextCronMoment(baseTz, fields);
    return nextTz ? nextTz.toDate() : undefined;
  }
  const offsetMinutes = parseTimezoneOffsetMinutes(tz);
  if (typeof offsetMinutes === 'number') {
    const localBase = new Date(base.getTime() + offsetMinutes * 60000);
    const nextLocal = nextCronTime(localBase, fields);
    if (!nextLocal) return undefined;
    return new Date(nextLocal.getTime() - offsetMinutes * 60000);
  }
  return nextCronTime(base, fields);
}

/**
 * Asserts an IANA timezone string.
 */
export function assertTimezone(value?: string): string {
  const tz = (value ?? '').trim();
  if (!tz) {
    throw new Error(_t('timezone is required', { scope: 'service/models/_cron' }));
  }
  if (!isIanaTimezone(tz)) {
    throw new Error(_t('invalid timezone: %s', { scope: 'service/models/_cron' }, tz));
  }
  return tz;
}

/**
 * Fills the computed next-run preview when the stored value is absent.
 */
export function applyNextRunPreview<T extends { Active?: boolean; CronExpr?: string; Timezone?: string; NextRunAt?: Date | null }>(schedule: T, baseTime?: Date): T {
  if (schedule.Active === false) {
    schedule.NextRunAt = null;
    return schedule;
  }
  if (!schedule.NextRunAt) {
    const computed = computeNextRunAt(schedule, baseTime);
    if (computed) schedule.NextRunAt = computed;
  }
  return schedule;
}
