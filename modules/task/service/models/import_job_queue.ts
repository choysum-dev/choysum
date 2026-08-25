// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import ImportJob from './import_job';
import Job from './job';

export type ImportJobQueueStatus = {
  importJobId: string;
  taskJobId: string;
  queueStatus: string;
  progressDone: number;
  progressTotal: number;
  reportJson?: Record<string, any>;
  reportRef?: string;
};

/** Joins lean ImportJob domain fields with task.Job queue status. */
export async function getQueueStatus(importJobId: string): Promise<ImportJobQueueStatus> {
  const id = String(importJobId || '').trim();
  if (!id) {
    throw new Error('importJobId is required');
  }
  const row = await ImportJob.Browse(id, [
    'Id',
    'TaskJobId',
    'ProgressDone',
    'ProgressTotal',
    'ReportJson',
    'ReportRef',
  ] as any);
  if (!row) {
    throw new Error(`import job ${id} not found`);
  }
  const taskJobId = String((row as any)?.TaskJobId || '').trim();
  if (!taskJobId) {
    throw new Error('import job is missing task job link');
  }
  const taskJob = await Job.GetJob(taskJobId, ['Id', 'Status'] as any);
  return {
    importJobId: id,
    taskJobId,
    queueStatus: String((taskJob as any)?.Status || ''),
    progressDone: Number((row as any)?.ProgressDone ?? 0) || 0,
    progressTotal: Number((row as any)?.ProgressTotal ?? 0) || 0,
    reportJson: ((row as any)?.ReportJson as Record<string, any>) || undefined,
    reportRef: String((row as any)?.ReportRef || '').trim() || undefined,
  };
}
