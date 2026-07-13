// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';

/**
 * Runtime execution snapshot for a background job.
 */
@Model('JobExecution', { application: 'task', companyScoped: false, tableName: 'task_job_execution', autoMigrate: false })
export default class JobExecution extends BaseModel {
  /** Related job identifier. */
  @Field({ type: 'varchar', size: 100, primaryKey: true, notNull: true})
  JobId: string;

  /** Current execution status. */
  @Field({ type: 'varchar', size: 32, index: true})
  Status: string;

  /** Current lease owner for the executing worker. */
  @Field({ type: 'varchar', size: 128})
  LeaseOwner: string;

  /** Lease expiration time for the current worker. */
  @Field({ type: 'datetime', index: true})
  LeaseUntil: Date;

  /** Attempt number for the execution. */
  @Field({ type: 'int', default: () => 0})
  Attempt: number;

  /** User who scheduled the job. */
  @Field({ type: 'varchar', size: 20, index: true})
  SchedulerUserId: string;

  /** User who triggered the job. */
  @Field({ type: 'varchar', size: 20, index: true})
  TriggeredByUserId: string;

  /** Fully-qualified method executed by the job. */
  @Field({ type: 'varchar', size: 255, index: true})
  FullMethod: string;

  /** Stored job payload. */
  @Field({ type: 'jsonobject' })
  PayloadJson: Record<string, any>;

  /** Stored job result payload. */
  @Field({ type: 'jsonobject' })
  ResultJson: Record<string, any>;

  /** Stored execution error payload. */
  @Field({ type: 'jsonobject' })
  ErrorJson: Record<string, any>;

  /** Time when the execution was cancelled. */
  @Field({ type: 'datetime', index: true})
  CancelledAt: Date;

  /** Time when the execution started. */
  @Field({ type: 'datetime', index: true})
  StartedAt: Date;

  /** Time when the execution finished. */
  @Field({ type: 'datetime', index: true})
  FinishedAt: Date;
}
