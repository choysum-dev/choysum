// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { getUserId, withUser } from '@/core/service/api/context';
import { listIanaTimezoneSelection } from '@/core/service/utils/datetime';
import { _lt } from '../i18n';
import Job from './job';
import { computeNextRunAt, assertTimezone } from './_cron';

/**
 * Immediate trigger command. Actor ids are the session user, not request fields.
 * With no session, the stored schedule SchedulerUserId is used via withUser.
 */
export type TriggerScheduleReq = {
  ScheduleId: string;
  PayloadOverride?: Record<string, unknown>;
};

/** Job created by an immediate schedule trigger. */
export type TriggerScheduleResp = {
  JobId: string;
};

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
  PayloadTemplateJson: Record<string, unknown>;

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

  /**
   * Persists NextRunAt from CronExpr and Timezone on Create / UpdateById.
   * Inactive schedules clear NextRunAt.
   */
  @Constraint<Schedule>(['Active', 'CronExpr', 'Timezone'])
  assignNextRunAt(): void {
    if (this.Active === false) {
      this.NextRunAt = null as unknown as Date;
      return;
    }
    const next = computeNextRunAt(this, new Date());
    if (next) this.NextRunAt = next;
  }

  /**
   * Triggers a schedule immediately and returns the created job id.
   * A session user wins. With no session, the stored SchedulerUserId is applied via withUser.
   * Client actor overrides are ignored.
   */
  static async TriggerSchedule(req: TriggerScheduleReq): Promise<TriggerScheduleResp> {
    const scheduleId = String(req?.ScheduleId || '').trim();
    if (!scheduleId) throw new Error('ScheduleId is required');
    const schedule = await this.Browse(scheduleId);
    const payload = (req?.PayloadOverride ?? schedule.PayloadTemplateJson ?? {}) as Record<string, unknown>;
    const timeoutMs = typeof schedule.TimeoutMs === 'number' && schedule.TimeoutMs > 0 ? schedule.TimeoutMs : 0;
    const sessionUserId = String(getUserId() || '').trim();
    const storedUserId = String(schedule.SchedulerUserId || '').trim();
    const enqueue = async (): Promise<TriggerScheduleResp> => {
      const job = await Job.EnqueueJob({
        TargetApp: schedule.TargetApp,
        FullMethod: schedule.FullMethod,
        Payload: payload,
        RunAfter: new Date(),
        MaxAttempts: 0,
        TimeoutMs: timeoutMs,
      });
      await this.UpdateById(scheduleId, { LastTriggeredAt: new Date(), LastRunAt: new Date() });
      return { JobId: job.Id };
    };
    if (sessionUserId) return enqueue();
    if (!storedUserId) throw new Error('authenticated user is required to trigger schedule');
    return withUser(storedUserId, enqueue);
  }
}
