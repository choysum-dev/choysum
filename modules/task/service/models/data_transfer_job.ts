// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import { _lt } from '../i18n';
import Job from './job';
import { executeExport, executeImport } from './data_transfer_job_worker';

export const DATA_TRANSFER_JOB_EXECUTE_IMPORT_FULL_METHOD = 'task.DataTransferJob/ExecuteImport';
export const DATA_TRANSFER_JOB_EXECUTE_EXPORT_FULL_METHOD = 'task.DataTransferJob/ExecuteExport';

const ALLOWED_PROFILES = new Set(['initdata', 'terminology', 'record']);
const ALLOWED_EXPORT_PROFILES = new Set(['record']);
const ALLOWED_POLICIES = new Set(['atomic', 'stop_keep', 'best_effort']);

export type EnqueueRecordImportInput = {
  targetModel: string;
  sourceRef: string;
  companyId?: string;
  policy?: string;
  profile?: string;
  specSnapshot: Record<string, unknown>;
};

export type EnqueueRecordImportResult = {
  dataTransferJobId: string;
  taskJobId: string;
};

export type EnqueueRecordExportInput = {
  targetModel: string;
  sourceRef: string;
  companyId?: string;
  profile?: string;
  specSnapshot: Record<string, unknown>;
};

export type EnqueueRecordExportResult = {
  dataTransferJobId: string;
  taskJobId: string;
};

function assertSelection(value: string | undefined, allowed: Set<string>, label: string): string {
  const normalized = String(value ?? '').trim();
  if (!normalized) {
    throw new Error(`data transfer ${label} is required`);
  }
  if (!allowed.has(normalized)) {
    throw new Error(`unsupported data transfer ${label} ${JSON.stringify(normalized)}`);
  }
  return normalized;
}

/**
 * Lean async data-transfer domain row (queue status lives on task.Job).
 * Direction distinguishes import vs export.
 */
@Model('DataTransferJob', { application: 'task', tableName: 'task_data_transfer_job' })
export default class DataTransferJob extends BaseModel {
  @Field({
    type: 'selection',
    selection: [
      { value: 'initdata', label: 'initdata' },
      { value: 'terminology', label: 'terminology' },
      { value: 'record', label: 'record' },
    ],
    size: 32,
    notNull: true,
    default: () => 'record',
    string: _lt('Profile', { scope: 'task.model.DataTransferJob.fields' }),
  })
  Profile: string;

  @Field({
    type: 'selection',
    selection: [
      { value: 'atomic', label: 'atomic' },
      { value: 'stop_keep', label: 'stop_keep' },
      { value: 'best_effort', label: 'best_effort' },
    ],
    size: 32,
    notNull: true,
    default: () => 'atomic',
    string: _lt('Policy', { scope: 'task.model.DataTransferJob.fields' }),
  })
  Policy: string;

  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Dry Run', { scope: 'task.model.DataTransferJob.fields' }),
  })
  DryRun: boolean;

  @Field({
    type: 'varchar',
    size: 255,
    index: true,
    notNull: true,
    string: _lt('Target Model', { scope: 'task.model.DataTransferJob.fields' }),
  })
  TargetModel: string;

  @Field({
    type: 'varchar',
    size: 255,
    index: true,
    notNull: true,
    string: _lt('Source Ref', { scope: 'task.model.DataTransferJob.fields' }),
  })
  SourceRef: string;

  @Field({
    type: 'varchar',
    size: 20,
    unique: true,
    index: true,
    string: _lt('Task Job', { scope: 'task.model.DataTransferJob.fields' }),
  })
  TaskJobId: string;

  @Field({
    type: 'varchar',
    size: 20,
    index: true,
    string: _lt('Company', { scope: 'task.model.DataTransferJob.fields' }),
  })
  CompanyId: string;

  @Field({
    type: 'int',
    default: () => 0,
    string: _lt('Progress Done', { scope: 'task.model.DataTransferJob.fields' }),
  })
  ProgressDone: number;

  @Field({
    type: 'int',
    default: () => 0,
    string: _lt('Progress Total', { scope: 'task.model.DataTransferJob.fields' }),
  })
  ProgressTotal: number;

  @Field({
    type: 'jsonobject',
    string: _lt('Report', { scope: 'task.model.DataTransferJob.fields' }),
  })
  ReportJson: Record<string, any>;

  @Field({
    type: 'varchar',
    size: 255,
    string: _lt('Report Ref', { scope: 'task.model.DataTransferJob.fields' }),
  })
  ReportRef: string;

  @Field({
    type: 'jsonobject',
    notNull: true,
    string: _lt('Spec Snapshot', { scope: 'task.model.DataTransferJob.fields' }),
  })
  SpecSnapshotJson: Record<string, any>;

  @Field({
    type: 'selection',
    selection: [
      { value: 'import', label: 'import' },
      { value: 'export', label: 'export' },
    ],
    size: 16,
    notNull: true,
    default: () => 'import',
    string: _lt('Direction', { scope: 'task.model.DataTransferJob.fields' }),
  })
  Direction: string;

  /** Creates DataTransferJob (Direction=import) + task.Job and links them 1:1. */
  static async EnqueueRecordImport(input: EnqueueRecordImportInput): Promise<EnqueueRecordImportResult> {
    const userId = String(getUserId() || '').trim();
    if (!userId) {
      throw new Error('authenticated user is required to enqueue data transfer job');
    }
    const targetModel = String(input?.targetModel || '').trim();
    const sourceRef = String(input?.sourceRef || '').trim();
    if (!targetModel || !sourceRef) {
      throw new Error('targetModel and sourceRef are required');
    }
    const specSnapshot = input?.specSnapshot;
    if (!specSnapshot || typeof specSnapshot !== 'object') {
      throw new Error('specSnapshot is required');
    }
    const profile = assertSelection(String(input?.profile ?? '').trim() || 'record', ALLOWED_PROFILES, 'profile');
    const policy = assertSelection(String(input?.policy ?? '').trim() || 'atomic', ALLOWED_POLICIES, 'policy');

    const row = await this.Create({
      Profile: profile,
      Policy: policy,
      DryRun: false,
      TargetModel: targetModel,
      SourceRef: sourceRef,
      CompanyId: String(input?.companyId || '').trim() || undefined,
      SpecSnapshotJson: specSnapshot,
      Direction: 'import',
      ProgressDone: 0,
      ProgressTotal: 0,
    } as Partial<DataTransferJob>);

    const taskJob = await Job.EnqueueJob(
      'task',
      DATA_TRANSFER_JOB_EXECUTE_IMPORT_FULL_METHOD,
      { dataTransferJobId: row.Id },
      userId,
      userId
    );

    await (this as any).UpdateById(row.Id, { TaskJobId: taskJob.Id } as Partial<DataTransferJob>);

    return { dataTransferJobId: row.Id, taskJobId: taskJob.Id };
  }

  /** Creates DataTransferJob (Direction=export) + task.Job and links them 1:1. */
  static async EnqueueRecordExport(input: EnqueueRecordExportInput): Promise<EnqueueRecordExportResult> {
    const userId = String(getUserId() || '').trim();
    if (!userId) {
      throw new Error('authenticated user is required to enqueue data transfer job');
    }
    const targetModel = String(input?.targetModel || '').trim();
    const sourceRef = String(input?.sourceRef || '').trim();
    if (!targetModel || !sourceRef) {
      throw new Error('targetModel and sourceRef are required');
    }
    const specSnapshot = input?.specSnapshot;
    if (!specSnapshot || typeof specSnapshot !== 'object') {
      throw new Error('specSnapshot is required');
    }
    const profile = assertSelection(String(input?.profile ?? '').trim() || 'record', ALLOWED_EXPORT_PROFILES, 'profile');

    const row = await this.Create({
      Profile: profile,
      Policy: 'atomic',
      DryRun: false,
      TargetModel: targetModel,
      SourceRef: sourceRef,
      CompanyId: String(input?.companyId || '').trim() || undefined,
      SpecSnapshotJson: specSnapshot,
      Direction: 'export',
      ProgressDone: 0,
      ProgressTotal: 0,
    } as Partial<DataTransferJob>);

    let taskJob;
    try {
      taskJob = await Job.EnqueueJob(
        'task',
        DATA_TRANSFER_JOB_EXECUTE_EXPORT_FULL_METHOD,
        { dataTransferJobId: row.Id },
        userId,
        userId
      );
    } catch (err) {
      try {
        await (this as any).DeleteById(row.Id);
      } catch {
        // best-effort cleanup when enqueue fails after row creation
      }
      throw err;
    }

    await (this as any).UpdateById(row.Id, { TaskJobId: taskJob.Id } as Partial<DataTransferJob>);

    return { dataTransferJobId: row.Id, taskJobId: taskJob.Id };
  }

  /** Task worker target for queued record imports. */
  static async ExecuteImport(dataTransferJobId: string): Promise<Record<string, any>> {
    return await executeImport(dataTransferJobId);
  }

  /** Task worker target for queued record exports. */
  static async ExecuteExport(dataTransferJobId: string): Promise<Record<string, any>> {
    return await executeExport(dataTransferJobId);
  }

  /** Persists transfer report and progress on the domain row. */
  static async FinalizeReport(dataTransferJobId: string, report: Record<string, any>): Promise<void> {
    const id = String(dataTransferJobId || '').trim();
    if (!id) {
      throw new Error('dataTransferJobId is required');
    }
    const stats = (report?.stats ?? report?.Stats ?? {}) as Record<string, any>;
    const total = Number(stats.total ?? stats.Total ?? 0) || 0;
    const artifactRef = String(report?.artifact_ref ?? report?.artifactRef ?? '').trim();
    const values: Partial<DataTransferJob> = {
      ReportJson: report ?? {},
      ProgressDone: total,
      ProgressTotal: total,
    };
    if (artifactRef) {
      values.ReportRef = artifactRef;
    }
    await (this as any).UpdateById(id, values as Partial<DataTransferJob>);
  }
}
