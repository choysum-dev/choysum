// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import type { FieldSelection } from '@/core/service/api/selection';
import type { SearchOptions, QueryCondition, OrderBy } from '@/core/service/api/query';
import { normalizeOffset } from '@/core/service/utils/normalization';
import { toDate } from '@/core/service/utils/datetime';
import { getBackendEnvPositiveInt } from '@/core/service/runtime/env/backend_env';
import { clampLimit } from './_helpers';
import { sanitizePayload } from './_payload';

/**
 * Supported lifecycle states for queued jobs.
 */
type JobStatus = 'queued' | 'dispatching' | 'succeeded' | 'failed' | 'cancelled';

/**
 * Filter and pagination options for paged job listing.
 */
type ListJobsParams = {
  targetApp?: string;
  fullMethod?: string;
  statuses?: JobStatus[];
  schedulerUserId?: string;
  triggeredByUserId?: string;
  createdAtGte?: string | number | Date;
  createdAtLt?: string | number | Date;
  runAfterGte?: string | number | Date;
  runAfterLt?: string | number | Date;
  limit?: number;
  offset?: number;
  orderBy?: OrderBy<Job> | OrderBy<Job>[];
  fields?: FieldSelection<Job>;
};

/** Builds a search condition from paged job list parameters. */
function buildJobCondition(params: ListJobsParams): QueryCondition<Job> | [] {
  const and: any[] = [];
  if (params.targetApp) and.push(['TargetApp', '=', params.targetApp]);
  if (params.fullMethod) and.push(['FullMethod', '=', params.fullMethod]);
  if (params.statuses && params.statuses.length > 0) and.push(['Status', 'in', params.statuses]);
  if (params.schedulerUserId) and.push(['SchedulerUserId', '=', params.schedulerUserId]);
  if (params.triggeredByUserId) and.push(['TriggeredByUserId', '=', params.triggeredByUserId]);

  const createdAtGte = toDate(params.createdAtGte);
  const createdAtLt = toDate(params.createdAtLt);
  const runAfterGte = toDate(params.runAfterGte);
  const runAfterLt = toDate(params.runAfterLt);

  if (createdAtGte) and.push(['CreatedAt', '>=', createdAtGte]);
  if (createdAtLt) and.push(['CreatedAt', '<', createdAtLt]);
  if (runAfterGte) and.push(['RunAfter', '>=', runAfterGte]);
  if (runAfterLt) and.push(['RunAfter', '<', runAfterLt]);

  if (and.length === 0) return [];
  return { And: and } as any;
}

/**
 * Persistent queued background job definition.
 */
@Model('Job', { application: 'task', companyScoped: false })
export default class Job extends BaseModel {
  /** Target application that owns the job. */
  @Field({ type: 'varchar', column: { size: 100, index: true, notNull: true } })
  TargetApp: string;

  /** Fully-qualified method invoked by the job. */
  @Field({ type: 'varchar', column: { size: 255, index: true, notNull: true } })
  FullMethod: string;

  /** Sanitized job payload. */
  @Field({ type: 'jsonobject' })
  PayloadJson: Record<string, any>;

  /** User who scheduled the job. */
  @Field({ type: 'varchar', column: { size: 20, index: true, notNull: true } })
  SchedulerUserId: string;

  /** User who triggered the job. */
  @Field({ type: 'varchar', column: { size: 20, index: true, notNull: true } })
  TriggeredByUserId: string;

  /** Current queue status. */
  @Field({
    type: 'selection',
    selection: [
      { value: 'queued', label: 'queued' },
      { value: 'dispatching', label: 'dispatching' },
      { value: 'succeeded', label: 'succeeded' },
      { value: 'failed', label: 'failed' },
      { value: 'cancelled', label: 'cancelled' },
    ],
    column: { size: 20, index: true, notNull: true, default: () => 'queued' },
  })
  Status: JobStatus;

  /** Earliest time the job may run. */
  @Field({ type: 'datetime', column: { index: true, notNull: true } })
  RunAfter: Date;

  /** Current attempt count. */
  @Field({ type: 'int', column: { default: () => 0 } })
  Attempt: number;

  /** Maximum attempt count before the job stops retrying. */
  @Field({ type: 'int', column: { default: () => 1 } })
  MaxAttempts: number;

  /** Per-job timeout budget in milliseconds. */
  @Field({ type: 'int', column: { default: () => 0 } })
  TimeoutMs: number;

  /** Time when cancellation was requested. */
  @Field({ type: 'datetime', column: { index: true } })
  CancelRequestedAt: Date;

  /** Time when the job was cancelled. */
  @Field({ type: 'datetime', column: { index: true } })
  CancelledAt: Date;

  /** Time when the job finished. */
  @Field({ type: 'datetime', column: { index: true } })
  FinishedAt: Date;

  /** Last execution error payload. */
  @Field({ type: 'jsonobject' })
  LastErrorJson: Record<string, any>;

  /** Hash of the last execution error payload. */
  @Field({ type: 'varchar', column: { size: 128 } })
  LastErrorHash: string;

  /** Whether the last error payload was truncated. */
  @Field({ type: 'boolean', column: { default: () => false } })
  LastErrorTruncated: boolean;

  /** Last execution result payload. */
  @Field({ type: 'jsonobject' })
  ResultJson: Record<string, any>;

  /** Hash of the last execution result payload. */
  @Field({ type: 'varchar', column: { size: 128 } })
  ResultHash: string;

  /** Whether the last result payload was truncated. */
  @Field({ type: 'boolean', column: { default: () => false } })
  ResultTruncated: boolean;

  /** Creates a queued job with sanitized payload data. */
  static async EnqueueJob(
    targetApp: string,
    fullMethod: string,
    payload: Record<string, any> = {},
    schedulerUserId: string,
    triggeredByUserId: string,
    runAfter?: string | number | Date,
    maxAttempts: number = 1,
    timeoutMs: number = 0
  ): Promise<Job> {
    const runAfterAt = toDate(runAfter) ?? new Date();
    const payloadSanitized = sanitizePayload(payload ?? {});
    const defaultMaxAttempts = getBackendEnvPositiveInt('CHOYSUM_TASK_DEFAULT_MAX_ATTEMPTS', 1);
    return await this.Create({
      TargetApp: targetApp,
      FullMethod: fullMethod,
      PayloadJson: payloadSanitized,
      SchedulerUserId: schedulerUserId,
      TriggeredByUserId: triggeredByUserId,
      Status: 'queued',
      RunAfter: runAfterAt,
      Attempt: 0,
      MaxAttempts: maxAttempts > 0 ? maxAttempts : defaultMaxAttempts,
      TimeoutMs: timeoutMs >= 0 ? timeoutMs : 0,
    });
  }

  /** Loads a single job by identifier. */
  static async GetJob(jobId: string, fields?: FieldSelection<Job>): Promise<Job> {
    return await this.Browse(jobId, fields);
  }

  /** Lists jobs using a raw query condition. */
  static async ListJobs(condition: QueryCondition<Job> | [] = [], options?: SearchOptions<Job>): Promise<Job[]> {
    return await this.Search(condition, options);
  }

  /** Lists jobs with filter, pagination, and total-count metadata. */
  static async ListJobsPaged(params: ListJobsParams = {}): Promise<{ items: Job[]; total: number; limit: number; offset: number }> {
    const condition = buildJobCondition(params);
    const limit = clampLimit(params.limit, 50, 500);
    const offset = normalizeOffset(params.offset);
    const orderBy = params.orderBy ?? ({ field: 'CreatedAt', order: 'desc' } as OrderBy<Job>);

    const items = await this.Search(condition, {
      limit,
      offset,
      orderBy,
      fields: params.fields,
    });
    const total = Number(await this.Count(condition as any)) || 0;
    return { items, total, limit, offset };
  }

  /** Cancels a queued job or requests cancellation for a running one. */
  static async CancelJob(jobId: string, reason?: string): Promise<Job> {
    const now = new Date();
    const existing = await this.Browse(jobId, ['Id', 'Status'] as any);
    const values: Partial<Job> = {};
    if (existing?.Status === 'queued') {
      values.Status = 'cancelled';
      values.CancelledAt = now;
      values.FinishedAt = now;
    } else {
      values.CancelRequestedAt = now;
    }
    if (reason) {
      values.LastErrorJson = { reason };
    }
    return await (this as any).UpdateById(jobId as any, values as any);
  }
}
