// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import type { QueryCondition, SearchOptions, OrderBy } from '@/core/service/api/query';
import type { FieldSelection } from '@/core/service/api/selection';
import { normalizeOffset } from '@/core/service/utils/normalization';
import { toDate } from '@/core/service/utils/date';
import Job from './job';
import { clampLimit } from './_helpers';
import { computeNextRunAt, normalizeTimezone, applyNextRunPreview } from './_cron';

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

  const createdAtGte = toDate(params.createdAtGte);
  const createdAtLt = toDate(params.createdAtLt);
  if (createdAtGte) and.push(['CreatedAt', '>=', createdAtGte]);
  if (createdAtLt) and.push(['CreatedAt', '<', createdAtLt]);

  if (and.length === 0) return [];
  return { And: and } as any;
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
