// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import type { QueryCondition, SearchOptions, OrderBy } from '@/core/service/api/query';
import type { FieldSelection } from '@/core/service/api/selection';
import { normalizeOffset } from '@/core/service/utils/normalization';
import moment from 'moment-timezone';
import Job from './job';

/**
 * Filter and pagination options for paged schedule listing.
 */
type ListSchedulesParams = {
  active?: boolean;
  name?: string;
  targetApp?: string;
  fullMethod?: string;
  cronExpr?: string;
  timezone?: string;
  createdAtGte?: string | number | Date;
  createdAtLt?: string | number | Date;
  limit?: number;
  offset?: number;
  orderBy?: OrderBy<Schedule> | OrderBy<Schedule>[];
  fields?: FieldSelection<Schedule>;
};

/**
 * Parsed cron fields used while computing preview run times.
 */
type CronFields = {
  minutes: Record<number, true>;
  hours: Record<number, true>;
  dom: Record<number, true>;
  month: Record<number, true>;
  dow: Record<number, true>;
};

/**
 * Parses a loose date input into a Date instance.
 */
function parseDate(input?: string | number | Date): Date | undefined {
  if (input == null) return undefined;
  if (input instanceof Date) return input;
  if (typeof input === 'number') return new Date(input);
  const parsed = new Date(input);
  if (Number.isNaN(parsed.getTime())) return undefined;
  return parsed;
}

/**
 * Builds a search condition from paged schedule list parameters.
 */
function buildScheduleCondition(params: ListSchedulesParams): QueryCondition<Schedule> | [] {
  const and: any[] = [];
  if (typeof params.active === 'boolean') and.push(['Active', '=', params.active]);
  if (params.name) and.push(['Name', 'contains', params.name]);
  if (params.targetApp) and.push(['TargetApp', '=', params.targetApp]);
  if (params.fullMethod) and.push(['FullMethod', '=', params.fullMethod]);
  if (params.cronExpr) and.push(['CronExpr', '=', params.cronExpr]);
  if (params.timezone) and.push(['Timezone', '=', params.timezone]);

  const createdAtGte = parseDate(params.createdAtGte);
  const createdAtLt = parseDate(params.createdAtLt);
  if (createdAtGte) and.push(['CreatedAt', '>=', createdAtGte]);
  if (createdAtLt) and.push(['CreatedAt', '<', createdAtLt]);

  if (and.length === 0) return [];
  return { And: and } as any;
}

/**
 * Clamps a requested page size to the supported range.
 */
function clampLimit(limit?: number, fallback: number = 50, max: number = 500): number {
  const val = typeof limit === 'number' ? limit : fallback;
  if (val <= 0) return fallback;
  return Math.min(val, max);
}


/**
 * Parses a fixed UTC offset string into minutes.
 */
function parseTimezoneOffsetMinutes(tz?: string): number | undefined {
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
function isIanaTimezone(tz?: string): boolean {
  if (!tz) return false;
  return Boolean(moment.tz.zone(tz));
}

/**
 * Parses a single cron field into an allowed-value set.
 */
function parseCronField(input: string, min: number, max: number): Record<number, true> | null {
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
function parseCronExpr(expr: string): CronFields | null {
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
function nextCronTime(from: Date, fields: CronFields): Date | undefined {
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
function nextCronMoment(from: moment.Moment, fields: CronFields): moment.Moment | undefined {
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
function computeNextRunAt(schedule: Schedule, baseTime?: Date): Date | undefined {
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
function normalizeTimezone(value?: string): string {
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
function applyNextRunPreview(schedule: Schedule, baseTime?: Date): Schedule {
  if (!schedule.NextRunAt) {
    const computed = computeNextRunAt(schedule, baseTime);
    if (computed) schedule.NextRunAt = computed;
  }
  return schedule;
}

/**
 * Persistent schedule definition for creating task jobs on a cron cadence.
 */
@Model('Schedule', { application: 'task', companyScoped: false })
export default class Schedule extends BaseModel {
  /** Whether the schedule is active. */
  @Field({ type: 'boolean', column: { default: () => true, index: true } })
  Active: boolean;

  /** Display name of the schedule. */
  @Field({ type: 'varchar', column: { size: 200, index: true, notNull: true } })
  Name: string;

  /** Target application that owns triggered jobs. */
  @Field({ type: 'varchar', column: { size: 100, index: true, notNull: true } })
  TargetApp: string;

  /** Fully-qualified method invoked by triggered jobs. */
  @Field({ type: 'varchar', column: { size: 255, index: true, notNull: true } })
  FullMethod: string;

  /** Payload template applied to triggered jobs. */
  @Field({ type: 'jsonobject' })
  PayloadTemplateJson: Record<string, any>;

  /** User who owns the schedule configuration. */
  @Field({ type: 'varchar', column: { size: 20, index: true, notNull: true } })
  SchedulerUserId: string;

  /** User recorded as the trigger actor. */
  @Field({ type: 'varchar', column: { size: 20, index: true, notNull: true } })
  TriggeredByUserId: string;

  /** Five-field cron expression. */
  @Field({ type: 'varchar', column: { size: 100, index: true, notNull: true } })
  CronExpr: string;

  /** IANA timezone used to evaluate the cron expression. */
  @Field({ type: 'varchar', column: { size: 100, notNull: true } })
  Timezone: string;

  /** Timeout budget applied to triggered jobs. */
  @Field({ type: 'int', column: { default: () => 0 } })
  TimeoutMs: number;

  /** Next computed run time preview. */
  @Field({ type: 'datetime', column: { index: true } })
  NextRunAt: Date;

  /** Time when the schedule last ran. */
  @Field({ type: 'datetime', column: { index: true } })
  LastRunAt: Date;

  /** Time when the schedule last triggered a job. */
  @Field({ type: 'datetime', column: { index: true } })
  LastTriggeredAt: Date;

  /** Creates a persisted schedule with an initial next-run preview. */
  static async CreateSchedule(
    name: string,
    targetApp: string,
    fullMethod: string,
    payloadTemplate: Record<string, any>,
    schedulerUserId: string,
    triggeredByUserId: string,
    cronExpr: string,
    timezone: string,
    timeoutMs: number = 0
  ): Promise<Schedule> {
    const tz = normalizeTimezone(timezone);
    const timeoutValue = Number.isFinite(timeoutMs) && timeoutMs > 0 ? Math.floor(timeoutMs) : 0;
    const now = new Date();
    const nextRunAt = computeNextRunAt(
      {
        CronExpr: cronExpr,
        Timezone: tz,
      } as Schedule,
      now
    );
    const created = await this.Create({
      Active: true,
      Name: name,
      TargetApp: targetApp,
      FullMethod: fullMethod,
      PayloadTemplateJson: payloadTemplate ?? {},
      SchedulerUserId: schedulerUserId,
      TriggeredByUserId: triggeredByUserId,
      CronExpr: cronExpr,
      Timezone: tz,
      TimeoutMs: timeoutValue,
      NextRunAt: nextRunAt,
    });
    return applyNextRunPreview(created, now);
  }

  /** Updates a schedule and recomputes its next-run preview when needed. */
  static async UpdateSchedule(scheduleId: string, values: Partial<Schedule>): Promise<Schedule> {
    const existing = await this.Browse(scheduleId);
    normalizeTimezone(values.Timezone ?? existing.Timezone);
    if (typeof values.TimeoutMs === 'number') {
      values.TimeoutMs = Number.isFinite(values.TimeoutMs) && values.TimeoutMs > 0 ? Math.floor(values.TimeoutMs) : 0;
    }
    const merged: Schedule = Object.assign(existing, values);
    const now = new Date();
    if (values.Active === false) {
      values.NextRunAt = null as any;
    } else if (values.CronExpr || values.Timezone || !existing.NextRunAt) {
      values.NextRunAt = computeNextRunAt(merged, now) as any;
    }
    const updated = await (this as any).UpdateById(scheduleId as any, values as any);
    return applyNextRunPreview(updated, now);
  }

  /** Deletes a schedule by identifier. */
  static async DeleteSchedule(scheduleId: string): Promise<number> {
    return await this.DeleteById(scheduleId);
  }

  /** Triggers a schedule immediately and returns the created job id. */
  static async TriggerSchedule(
    scheduleId: string,
    payloadOverride?: Record<string, any>,
    schedulerUserIdOverride?: string,
    triggeredByUserId?: string
  ): Promise<{ jobId: string }> {
    const schedule = await this.Browse(scheduleId);
    const payload = payloadOverride ?? schedule.PayloadTemplateJson ?? {};
    const schedulerUserId = schedulerUserIdOverride ?? schedule.SchedulerUserId;
    const triggeredBy = triggeredByUserId ?? schedule.TriggeredByUserId;
    const timeoutMs = typeof schedule.TimeoutMs === 'number' && schedule.TimeoutMs > 0 ? schedule.TimeoutMs : 0;
    const job = await Job.EnqueueJob(schedule.TargetApp, schedule.FullMethod, payload, schedulerUserId, triggeredBy, new Date(), 0, timeoutMs);
    await (this as any).UpdateById(scheduleId as any, { LastTriggeredAt: new Date(), LastRunAt: new Date() } as any);
    return { jobId: job.Id };
  }

  /** Lists schedules using a raw query condition. */
  static async ListSchedules(condition: QueryCondition<Schedule> | [] = [], options?: SearchOptions<Schedule>): Promise<Schedule[]> {
    const items = await this.Search(condition, options);
    return items.map(item => applyNextRunPreview(item));
  }

  /** Lists schedules with filter, pagination, and total-count metadata. */
  static async ListSchedulesPaged(params: ListSchedulesParams = {}): Promise<{ items: Schedule[]; total: number; limit: number; offset: number }> {
    const condition = buildScheduleCondition(params);
    const limit = clampLimit(params.limit, 50, 500);
    const offset = normalizeOffset(params.offset);
    const orderBy = params.orderBy ?? ({ field: 'CreatedAt', order: 'desc' } as OrderBy<Schedule>);
    const items = await this.Search(condition, {
      limit,
      offset,
      orderBy,
      fields: params.fields,
    });
    const total = Number(await this.Count(condition as any)) || 0;
    return { items: items.map(item => applyNextRunPreview(item)), total, limit, offset };
  }
}
