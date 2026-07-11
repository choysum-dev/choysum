// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import moment from 'moment-timezone';

/**
 * Parsed cron fields used while computing preview run times.
 */
export type CronFields = {
  minutes: Record<number, true>;
  hours: Record<number, true>;
  dom: Record<number, true>;
  month: Record<number, true>;
  dow: Record<number, true>;
};

/**
 * Parses a fixed UTC offset string into minutes.
 */
export function parseTimezoneOffsetMinutes(tz?: string): number | undefined {
  if (!tz) return undefined;
  const raw = tz.trim();
  if (!raw) return undefined;
  if (raw === 'UTC' || raw === 'GMT' || raw === 'Z') return 0;

  const match = raw.match(/^([+-])(\d{1,2})(?::?(\d{2}))?$/);
  if (!match) return undefined;
  const sign = match[1] === '-' ? -1 : 1;
  const hours = Number(match[2]);
  const minutes = Number(match[3] ?? '0');
  if (Number.isNaN(hours) || Number.isNaN(minutes)) return undefined;
  if (hours > 14 || minutes >= 60) return undefined;
  return sign * (hours * 60 + minutes);
}

/**
 * Reports whether a timezone string is a valid IANA timezone.
 */
export function isIanaTimezone(tz?: string): boolean {
  if (!tz) return false;
  return Boolean(moment.tz.zone(tz));
}

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
      if (!Number.isFinite(step) || step <= 0) return null;
      for (let i = min; i <= max; i += step) set[i] = true;
      continue;
    }
    if (part.includes('-')) {
      const [a, b] = part.split('-', 2);
      const start = Number(a);
      const end = Number(b);
      if (!Number.isFinite(start) || !Number.isFinite(end)) return null;
      if (start < min || end > max || start > end) return null;
      for (let i = start; i <= end; i += 1) set[i] = true;
      continue;
    }
    const val = Number(part);
    if (!Number.isFinite(val) || val < min || val > max) return null;
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
  const minutes = parseCronField(parts[0], 0, 59);
  const hours = parseCronField(parts[1], 0, 23);
  const dom = parseCronField(parts[2], 1, 31);
  const month = parseCronField(parts[3], 1, 12);
  const dow = parseCronField(parts[4], 0, 6);
  if (!minutes || !hours || !dom || !month || !dow) return null;
  return { minutes, hours, dom, month, dow };
}

/**
 * Computes the next matching cron time in local Date space.
 */
export function nextCronTime(from: Date, fields: CronFields): Date | undefined {
  const cur = new Date(from.getTime());
  cur.setSeconds(0, 0);
  cur.setMinutes(cur.getMinutes() + 1);
  for (let i = 0; i < 525600; i += 1) {
    const month = cur.getMonth() + 1;
    const dom = cur.getDate();
    const dow = cur.getDay();
    const hours = cur.getHours();
    const minutes = cur.getMinutes();
    if (fields.month[month] && fields.dom[dom] && fields.dow[dow] && fields.hours[hours] && fields.minutes[minutes]) {
      return new Date(cur.getTime());
    }
    cur.setMinutes(cur.getMinutes() + 1);
  }
  return undefined;
}

/**
 * Computes the next matching cron time in a timezone-aware moment instance.
 */
export function nextCronMoment(from: moment.Moment, fields: CronFields): moment.Moment | undefined {
  const cur = from.clone().second(0).millisecond(0).add(1, 'minute');
  for (let i = 0; i < 525600; i += 1) {
    const month = cur.month() + 1;
    const dom = cur.date();
    const dow = cur.day();
    const hours = cur.hour();
    const minutes = cur.minute();
    if (fields.month[month] && fields.dom[dom] && fields.dow[dow] && fields.hours[hours] && fields.minutes[minutes]) {
      return cur.clone();
    }
    cur.add(1, 'minute');
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
 * Validates and normalizes an IANA timezone.
 */
export function normalizeTimezone(value?: string): string {
  const tz = (value ?? '').trim();
  if (!tz) {
    throw new Error('timezone is required');
  }
  if (!isIanaTimezone(tz)) {
    throw new Error(`invalid timezone: ${tz}`);
  }
  return tz;
}

/**
 * Fills the computed next-run preview when the stored value is absent.
 */
export function applyNextRunPreview<T extends { CronExpr?: string; Timezone?: string; NextRunAt?: Date }>(schedule: T, baseTime?: Date): T {
  if (!schedule.NextRunAt) {
    const computed = computeNextRunAt(schedule, baseTime);
    if (computed) schedule.NextRunAt = computed;
  }
  return schedule;
}
