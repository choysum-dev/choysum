// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';

/**
 * Runtime execution snapshot for a background job.
 */
@Model('JobExecution', { application: 'task', tableName: 'task_job_execution', autoMigrate: false })
export default class JobExecution extends BaseModel {
  /** Related job identifier. */
  @Field({
    type: 'varchar',
    size: 100,
    primaryKey: true,
    notNull: true,
    string: _lt('Job', { scope: 'task.model.JobExecution.fields' }),
  })
  JobId: string;

  /** Current execution status. */
  @Field({
    type: 'varchar',
    size: 32,
    index: true,
    string: _lt('Status', { scope: 'task.model.JobExecution.fields' }),
  })
  Status: string;

  /** Current lease owner for the executing worker. */
  @Field({
    type: 'varchar',
    size: 128,
    copy: false,
    string: _lt('Lease Owner', { scope: 'task.model.JobExecution.fields' }),
  })
  LeaseOwner: string;

  /** Lease expiration time for the current worker. */
  @Field({
    type: 'datetime',
    index: true,
    copy: false,
    string: _lt('Lease Until', { scope: 'task.model.JobExecution.fields' }),
  })
  LeaseUntil: Date;

  /** Attempt number for the execution. */
  @Field({
    type: 'int',
    default: () => 0,
    string: _lt('Attempt', { scope: 'task.model.JobExecution.fields' }),
  })
  Attempt: number;

  /** User who scheduled the job. */
  @Field({
    type: 'varchar',
    size: 20,
    index: true,
    string: _lt('Scheduler User', { scope: 'task.model.JobExecution.fields' }),
  })
  SchedulerUserId: string;

  /** User who triggered the job. */
  @Field({
    type: 'varchar',
    size: 20,
    index: true,
    string: _lt('Triggered By User', { scope: 'task.model.JobExecution.fields' }),
  })
  TriggeredByUserId: string;

  /** Fully-qualified method executed by the job. */
  @Field({
    type: 'varchar',
    size: 255,
    index: true,
    string: _lt('Full Method', { scope: 'task.model.JobExecution.fields' }),
  })
  FullMethod: string;

  /** Stored job payload. */
  @Field({
    type: 'jsonobject',
    string: _lt('Payload', { scope: 'task.model.JobExecution.fields' }),
  })
  PayloadJson: Record<string, any>;

  /** Stored job result payload. */
  @Field({
    type: 'jsonobject',
    string: _lt('Result', { scope: 'task.model.JobExecution.fields' }),
  })
  ResultJson: Record<string, any>;

  /** Stored execution error payload. */
  @Field({
    type: 'jsonobject',
    string: _lt('Error', { scope: 'task.model.JobExecution.fields' }),
  })
  ErrorJson: Record<string, any>;

  /** Time when the execution was cancelled. */
  @Field({
    type: 'datetime',
    index: true,
    string: _lt('Cancelled At', { scope: 'task.model.JobExecution.fields' }),
  })
  CancelledAt: Date;

  /** Time when the execution started. */
  @Field({
    type: 'datetime',
    index: true,
    string: _lt('Started At', { scope: 'task.model.JobExecution.fields' }),
  })
  StartedAt: Date;

  /** Time when the execution finished. */
  @Field({
    type: 'datetime',
    index: true,
    string: _lt('Finished At', { scope: 'task.model.JobExecution.fields' }),
  })
  FinishedAt: Date;
}
