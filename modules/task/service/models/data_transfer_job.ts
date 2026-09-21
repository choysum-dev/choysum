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

/** Frozen import/export spec snapshot stored on the job and passed to the bridge. */
export type DataTransferSpecSnapshot = Record<string, unknown>;

/** Report written by FinalizeReport. */
export type DataTransferReport = Record<string, unknown>;

export type EnqueueRecordImportReq = {
  TargetModel: string;
  SourceRef: string;
  CompanyId?: string;
  Policy?: string;
  Profile?: string;
  SpecSnapshot: DataTransferSpecSnapshot;
};

export type EnqueueRecordImportResp = {
  DataTransferJobId: string;
  TaskJobId: string;
};

export type EnqueueRecordExportReq = {
  TargetModel: string;
  SourceRef: string;
  CompanyId?: string;
  Profile?: string;
  SpecSnapshot: DataTransferSpecSnapshot;
};

export type EnqueueRecordExportResp = {
  DataTransferJobId: string;
  TaskJobId: string;
};

export type FinalizeReportReq = {
  DataTransferJobId: string;
  Report?: DataTransferReport;
};

function assertSelection(value: string, allowed: Set<string>, label: string): string {
  if (!allowed.has(value)) {
    throw new Error(`unsupported data transfer ${label} ${JSON.stringify(value)}`);
  }
  return value;
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
  ReportJson: Record<string, unknown>;

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
  SpecSnapshotJson: Record<string, unknown>;

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
  static async EnqueueRecordImport(req: EnqueueRecordImportReq): Promise<EnqueueRecordImportResp> {
    const userId = String(getUserId() || '').trim();
    if (!userId) {
      throw new Error('authenticated user is required to enqueue data transfer job');
    }
    const targetModel = String(req?.TargetModel || '').trim();
    const sourceRef = String(req?.SourceRef || '').trim();
    if (!targetModel || !sourceRef) {
      throw new Error('TargetModel and SourceRef are required');
    }
    const specSnapshot = req?.SpecSnapshot;
    if (!specSnapshot || typeof specSnapshot !== 'object') {
      throw new Error('SpecSnapshot is required');
    }
    const profile = assertSelection(String(req?.Profile ?? '').trim() || 'record', ALLOWED_PROFILES, 'profile');
    const policy = assertSelection(String(req?.Policy ?? '').trim() || 'atomic', ALLOWED_POLICIES, 'policy');

    const row = await this.Create({
      Profile: profile,
      Policy: policy,
      DryRun: false,
      TargetModel: targetModel,
      SourceRef: sourceRef,
      CompanyId: String(req?.CompanyId || '').trim() || undefined,
      SpecSnapshotJson: specSnapshot,
      Direction: 'import',
      ProgressDone: 0,
      ProgressTotal: 0,
    } as Partial<DataTransferJob>);

    let taskJob;
    try {
      taskJob = await Job.EnqueueJob({
        TargetApp: 'task',
        FullMethod: DATA_TRANSFER_JOB_EXECUTE_IMPORT_FULL_METHOD,
        Payload: { dataTransferJobId: row.Id },
      });
    } catch (err) {
      try {
        await this.DeleteById(row.Id);
      } catch {
        // best-effort cleanup when enqueue fails after row creation
      }
      throw err;
    }

    await this.UpdateById(row.Id, { TaskJobId: taskJob.Id } as Partial<DataTransferJob>);

    return { DataTransferJobId: row.Id, TaskJobId: taskJob.Id };
  }

  /** Creates DataTransferJob (Direction=export) + task.Job and links them 1:1. */
  static async EnqueueRecordExport(req: EnqueueRecordExportReq): Promise<EnqueueRecordExportResp> {
    const userId = String(getUserId() || '').trim();
    if (!userId) {
      throw new Error('authenticated user is required to enqueue data transfer job');
    }
    const targetModel = String(req?.TargetModel || '').trim();
    const sourceRef = String(req?.SourceRef || '').trim();
    if (!targetModel || !sourceRef) {
      throw new Error('TargetModel and SourceRef are required');
    }
    const specSnapshot = req?.SpecSnapshot;
    if (!specSnapshot || typeof specSnapshot !== 'object') {
      throw new Error('SpecSnapshot is required');
    }
    const profile = assertSelection(String(req?.Profile ?? '').trim() || 'record', ALLOWED_EXPORT_PROFILES, 'profile');

    const row = await this.Create({
      Profile: profile,
      Policy: 'atomic',
      DryRun: false,
      TargetModel: targetModel,
      SourceRef: sourceRef,
      CompanyId: String(req?.CompanyId || '').trim() || undefined,
      SpecSnapshotJson: specSnapshot,
      Direction: 'export',
      ProgressDone: 0,
      ProgressTotal: 0,
    } as Partial<DataTransferJob>);

    let taskJob;
    try {
      taskJob = await Job.EnqueueJob({
        TargetApp: 'task',
        FullMethod: DATA_TRANSFER_JOB_EXECUTE_EXPORT_FULL_METHOD,
        Payload: { dataTransferJobId: row.Id },
      });
    } catch (err) {
      try {
        await this.DeleteById(row.Id);
      } catch {
        // best-effort cleanup when enqueue fails after row creation
      }
      throw err;
    }

    await this.UpdateById(row.Id, { TaskJobId: taskJob.Id } as Partial<DataTransferJob>);

    return { DataTransferJobId: row.Id, TaskJobId: taskJob.Id };
  }

  /** Task-worker FullMethod target for queued record imports (not interactive client API). */
  static async ExecuteImport(dataTransferJobId: string): Promise<Record<string, unknown>> {
    return await executeImport(dataTransferJobId);
  }

  /** Task-worker FullMethod target for queued record exports (not interactive client API). */
  static async ExecuteExport(dataTransferJobId: string): Promise<Record<string, unknown>> {
    return await executeExport(dataTransferJobId);
  }

  /** Persists transfer report and progress on the domain row. */
  static async FinalizeReport(req: FinalizeReportReq): Promise<void> {
    const id = String(req?.DataTransferJobId || '').trim();
    if (!id) {
      throw new Error('DataTransferJobId is required');
    }
    const report = req?.Report ?? {};
    const stats = (report?.stats ?? report?.Stats ?? {}) as Record<string, unknown>;
    const total = Number(stats.total ?? stats.Total ?? 0) || 0;
    const artifactRef = String(report?.artifact_ref ?? report?.artifactRef ?? '').trim();
    const values: Partial<DataTransferJob> = {
      ReportJson: report,
      ProgressDone: total,
      ProgressTotal: total,
    };
    if (artifactRef) {
      values.ReportRef = artifactRef;
    }
    await this.UpdateById(id, values as Partial<DataTransferJob>);
  }
}
