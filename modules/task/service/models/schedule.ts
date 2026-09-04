// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import type { QueryCondition, SearchOptions, OrderBy } from '@/core/service/api/query';
import type { FieldSelection } from '@/core/service/api/selection';
import { normalizeOffset } from '@/core/service/utils/normalization';
import { toDate, listIanaTimezoneSelection } from '@/core/service/utils/datetime';
import { _lt } from '../i18n';
import Job from './job';
import { clampLimit } from './_limit';
import { computeNextRunAt, assertTimezone, applyNextRunPreview } from './_cron';

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
  if (params.name) and.push(['Name', 'ilike', `%${params.name}%`]);
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
@Model('Schedule', { application: 'task' })
export default class Schedule extends BaseModel {
  /** Whether the schedule is active. */
  @Field({
    type: 'boolean',
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'task.model.Schedule.fields' }),
  })
  Active: boolean;

  /** Display name of the schedule. */
  @Field({
    type: 'varchar',
    size: 200,
    notNull: true,
    translate: true,
    index: 'trigram',
    string: _lt('Name', { scope: 'task.model.Schedule.fields' }),
  })
  Name: string;

  /** Target application that owns triggered jobs. */
  @Field({
    type: 'varchar',
    size: 100,
    index: true,
    notNull: true,
    string: _lt('Target App', { scope: 'task.model.Schedule.fields' }),
  })
  TargetApp: string;

  /** Fully-qualified method invoked by triggered jobs. */
  @Field({
    type: 'varchar',
    size: 255,
    index: true,
    notNull: true,
    string: _lt('Full Method', { scope: 'task.model.Schedule.fields' }),
    help: _lt('gRPC full method path invoked when the schedule fires.', {
      scope: 'task.model.Schedule.fields',
    }),
  })
  FullMethod: string;

  /** Payload template applied to triggered jobs. */
  @Field({
    type: 'jsonobject',
    string: _lt('Payload Template', { scope: 'task.model.Schedule.fields' }),
  })
  PayloadTemplateJson: Record<string, any>;

  /** User who owns the schedule configuration. */
  @Field({
    type: 'varchar',
    size: 20,
    index: true,
    notNull: true,
    string: _lt('Scheduler User', { scope: 'task.model.Schedule.fields' }),
  })
  SchedulerUserId: string;

  /** User recorded as the trigger actor. */
  @Field({
    type: 'varchar',
    size: 20,
    index: true,
    notNull: true,
    string: _lt('Triggered By User', { scope: 'task.model.Schedule.fields' }),
  })
  TriggeredByUserId: string;

  /** Five-field cron expression. */
  @Field({
    type: 'varchar',
    size: 100,
    index: true,
    notNull: true,
    string: _lt('Cron Expression', { scope: 'task.model.Schedule.fields' }),
    help: _lt('Five-field cron (minute hour dom month dow).', {
      scope: 'task.model.Schedule.fields',
    }),
  })
  CronExpr: string;

  /** IANA timezone used to evaluate the cron expression. */
  @Field({
    type: 'selection',
    selection: () => listIanaTimezoneSelection(),
    size: 100,
    notNull: true,
    string: _lt('Timezone', { scope: 'task.model.Schedule.fields' }),
    help: _lt('IANA zone used to evaluate the cron expression.', {
      scope: 'task.model.Schedule.fields',
    }),
  })
  Timezone: string;

  /** Timeout budget applied to triggered jobs. */
  @Field({
    type: 'int',
    default: () => 0,
    string: _lt('Timeout Ms', { scope: 'task.model.Schedule.fields' }),
    help: _lt('0 uses the platform default (no timeout unless configured).', {
      scope: 'task.model.Schedule.fields',
    }),
  })
  TimeoutMs: number;

  /** Next computed run time preview. */
  @Field({
    type: 'datetime',
    index: true,
    string: _lt('Next Run At', { scope: 'task.model.Schedule.fields' }),
  })
  NextRunAt: Date;

  /** Time when the schedule last ran. */
  @Field({
    type: 'datetime',
    index: true,
    string: _lt('Last Run At', { scope: 'task.model.Schedule.fields' }),
  })
  LastRunAt: Date;

  /** Time when the schedule last triggered a job. */
  @Field({
    type: 'datetime',
    index: true,
    string: _lt('Last Triggered At', { scope: 'task.model.Schedule.fields' }),
  })
  LastTriggeredAt: Date;

  /** Validate and normalize Timezone on generic Create / UpdateById paths. */
  @Constraint<Schedule>(['Timezone'])
  validateTimezoneConstraint(): void {
    this.Timezone = assertTimezone(this.Timezone);
  }

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
    const tz = assertTimezone(timezone);
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
    assertTimezone(values.Timezone ?? existing.Timezone);
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
