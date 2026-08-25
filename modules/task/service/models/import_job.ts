// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import { _lt } from '../i18n';
import Job from './job';
import { executeImportJob } from './import_job_worker';

export const IMPORT_JOB_EXECUTE_FULL_METHOD = 'task.ImportJob/ExecuteImport';

const ALLOWED_PROFILES = new Set(['initdata', 'terminology', 'record']);
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
  importJobId: string;
  taskJobId: string;
};

function normalizeSelection(value: string | undefined, fallback: string, allowed: Set<string>, label: string): string {
  const normalized = String(value || '').trim() || fallback;
  if (!allowed.has(normalized)) {
    throw new Error(`unsupported import ${label} ${JSON.stringify(normalized)}`);
  }
  return normalized;
}

/**
 * Lean async import domain row (queue status lives on task.Job).
 */
@Model('ImportJob', { application: 'task', tableName: 'task_import_job' })
export default class ImportJob extends BaseModel {
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
    string: _lt('Profile', { scope: 'task.model.ImportJob.fields' }),
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
    string: _lt('Policy', { scope: 'task.model.ImportJob.fields' }),
  })
  Policy: string;

  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Dry Run', { scope: 'task.model.ImportJob.fields' }),
  })
  DryRun: boolean;

  @Field({
    type: 'varchar',
    size: 255,
    index: true,
    notNull: true,
    string: _lt('Target Model', { scope: 'task.model.ImportJob.fields' }),
  })
  TargetModel: string;

  @Field({
    type: 'varchar',
    size: 255,
    index: true,
    notNull: true,
    string: _lt('Source Ref', { scope: 'task.model.ImportJob.fields' }),
  })
  SourceRef: string;

  @Field({
    type: 'varchar',
    size: 20,
    unique: true,
    index: true,
    string: _lt('Task Job', { scope: 'task.model.ImportJob.fields' }),
  })
  TaskJobId: string;

  @Field({
    type: 'varchar',
    size: 20,
    index: true,
    string: _lt('Company', { scope: 'task.model.ImportJob.fields' }),
  })
  CompanyId: string;

  @Field({
    type: 'int',
    default: () => 0,
    string: _lt('Progress Done', { scope: 'task.model.ImportJob.fields' }),
  })
  ProgressDone: number;

  @Field({
    type: 'int',
    default: () => 0,
    string: _lt('Progress Total', { scope: 'task.model.ImportJob.fields' }),
  })
  ProgressTotal: number;

  @Field({
    type: 'jsonobject',
    string: _lt('Report', { scope: 'task.model.ImportJob.fields' }),
  })
  ReportJson: Record<string, any>;

  @Field({
    type: 'varchar',
    size: 255,
    string: _lt('Report Ref', { scope: 'task.model.ImportJob.fields' }),
  })
  ReportRef: string;

  @Field({
    type: 'jsonobject',
    notNull: true,
    string: _lt('Spec Snapshot', { scope: 'task.model.ImportJob.fields' }),
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
    string: _lt('Direction', { scope: 'task.model.ImportJob.fields' }),
  })
  Direction: string;

  /** Creates ImportJob + task.Job and links them 1:1. */
  static async EnqueueRecordImport(input: EnqueueRecordImportInput): Promise<EnqueueRecordImportResult> {
    const userId = String(getUserId() || '').trim();
    if (!userId) {
      throw new Error('authenticated user is required to enqueue import job');
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
    const profile = normalizeSelection(input?.profile, 'record', ALLOWED_PROFILES, 'profile');
    const policy = normalizeSelection(input?.policy, 'atomic', ALLOWED_POLICIES, 'policy');

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
    } as Partial<ImportJob>);

    const taskJob = await Job.EnqueueJob(
      'task',
      IMPORT_JOB_EXECUTE_FULL_METHOD,
      { importJobId: row.Id },
      userId,
      userId
    );

    await (this as any).UpdateById(row.Id, { TaskJobId: taskJob.Id } as Partial<ImportJob>);

    return { importJobId: row.Id, taskJobId: taskJob.Id };
  }

  /** Task worker target for queued record imports. */
  static async ExecuteImport(importJobId: string): Promise<Record<string, any>> {
    return await executeImportJob(importJobId);
  }

  /** Persists import report and progress on the domain row. */
  static async FinalizeReport(importJobId: string, report: Record<string, any>): Promise<void> {
    const id = String(importJobId || '').trim();
    if (!id) {
      throw new Error('importJobId is required');
    }
    const stats = (report?.stats ?? report?.Stats ?? {}) as Record<string, any>;
    const total = Number(stats.total ?? stats.Total ?? 0) || 0;
    const artifactRef = String(report?.artifact_ref ?? report?.artifactRef ?? '').trim();
    const values: Partial<ImportJob> = {
      ReportJson: report ?? {},
      ProgressDone: total,
      ProgressTotal: total,
    };
    if (artifactRef) {
      values.ReportRef = artifactRef;
    }
    await (this as any).UpdateById(id, values as Partial<ImportJob>);
  }
}
