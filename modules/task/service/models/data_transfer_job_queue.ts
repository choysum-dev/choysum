// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import DataTransferJob from './data_transfer_job';
import Job from './job';

export type DataTransferJobQueueStatus = {
  dataTransferJobId: string;
  taskJobId: string;
  queueStatus: string;
  progressDone: number;
  progressTotal: number;
  reportJson?: Record<string, any>;
  reportRef?: string;
};

/** Joins lean DataTransferJob domain fields with task.Job queue status. */
export async function getQueueStatus(dataTransferJobId: string): Promise<DataTransferJobQueueStatus> {
  const id = String(dataTransferJobId || '').trim();
  if (!id) {
    throw new Error('dataTransferJobId is required');
  }
  const row = await DataTransferJob.Browse(id, [
    'Id',
    'TaskJobId',
    'ProgressDone',
    'ProgressTotal',
    'ReportJson',
    'ReportRef',
  ]);
  if (!row) {
    throw new Error(`data transfer job ${id} not found`);
  }
  const taskJobId = String(row.TaskJobId || '').trim();
  if (!taskJobId) {
    throw new Error('data transfer job is missing task job link');
  }
  const taskJob = await Job.GetJob(taskJobId, ['Id', 'Status']);
  return {
    dataTransferJobId: id,
    taskJobId,
    queueStatus: String(taskJob?.Status || ''),
    progressDone: Number(row.ProgressDone ?? 0),
    progressTotal: Number(row.ProgressTotal ?? 0),
    reportJson: (row.ReportJson as Record<string, any>) || undefined,
    reportRef: String(row.ReportRef || '').trim() || undefined,
  };
}
