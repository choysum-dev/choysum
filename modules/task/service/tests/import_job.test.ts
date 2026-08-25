// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import ImportJob, { IMPORT_JOB_EXECUTE_FULL_METHOD } from '@/task/service/models/import_job';
import { getQueueStatus } from '@/task/service/models/import_job_queue';
import { executeImportJob } from '@/task/service/models/import_job_worker';
import Job from '@/task/service/models/job';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const CTX_OVERRIDE_KEY = Symbol.for('choysum.ctx.override');
const CTX_FROZEN_KEY = Symbol.for('choysum.ctx.frozen');

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum;
  if (!root) {
    throw new Error('missing global $choysum');
  }
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    companyMode: 'skip',
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'task.Job:create',
      'task.Job:read',
      'task.Job:write',
      'task.Job:delete',
      'task.ImportJob:create',
      'task.ImportJob:read',
      'task.ImportJob:write',
      'task.ImportJob:delete',
      'ImportJob:create',
      'ImportJob:read',
      'ImportJob:write',
      'ImportJob:delete',
    ],
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = { userId: 'admin' };
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
  delete (jsCtx as any)[CTX_OVERRIDE_KEY];
  delete (jsCtx as any)[CTX_FROZEN_KEY];
}

test('ImportJob model has no State field metadata', () => {
  resetRequestContext();
  const fields = (ImportJob as any).$meta?.fields;
  expect(fields?.has?.('State') ?? fields?.State).toBeFalsy();
});

test('ImportJob.EnqueueRecordImport creates linked task job', async () => {
  resetRequestContext();
  const result = await ImportJob.EnqueueRecordImport({
    targetModel: 'base.Country',
    sourceRef: 'doc-ref-1',
    companyId: 'cmp-1',
    policy: 'atomic',
    specSnapshot: {
      profile: 'record',
      caller: 'user',
      policy: 'atomic',
      model: 'base.Country',
      source: { format: 'csv', document_ref: 'doc-ref-1' },
    },
  });
  expect(result.importJobId).toBeTruthy();
  expect(result.taskJobId).toBeTruthy();

  const row = await ImportJob.Browse(result.importJobId, ['TaskJobId', 'TargetModel', 'Policy'] as any);
  expect((row as any).TaskJobId).toBe(result.taskJobId);
  expect((row as any).TargetModel).toBe('base.Country');
  expect((row as any).Policy).toBe('atomic');

  const taskJob = await Job.GetJob(result.taskJobId, ['FullMethod', 'PayloadJson'] as any);
  expect((taskJob as any).FullMethod).toBe(IMPORT_JOB_EXECUTE_FULL_METHOD);
  expect((taskJob as any).PayloadJson?.importJobId).toBe(result.importJobId);
});

test('getQueueStatus joins ImportJob with Job.Status', async () => {
  resetRequestContext();
  const enqueued = await ImportJob.EnqueueRecordImport({
    targetModel: 'base.Country',
    sourceRef: 'doc-ref-2',
    specSnapshot: {
      profile: 'record',
      caller: 'user',
      policy: 'atomic',
      model: 'base.Country',
      source: { format: 'csv', document_ref: 'doc-ref-2' },
    },
  });
  const status = await getQueueStatus(enqueued.importJobId);
  expect(status.queueStatus).toBe('queued');
  expect(status.taskJobId).toBe(enqueued.taskJobId);
});

test('executeImportJob writes report via import bridge', async () => {
  resetRequestContext();
  const root: any = (globalThis as any).$choysum;
  root.import = {
    run: async () => ({
      profile: 'record',
      policy: 'atomic',
      stats: { total: 2, ok: 2, error: 0, skip: 0 },
      messages: [],
    }),
  };

  const enqueued = await ImportJob.EnqueueRecordImport({
    targetModel: 'base.Country',
    sourceRef: 'doc-ref-3',
    specSnapshot: {
      profile: 'record',
      caller: 'user',
      policy: 'atomic',
      model: 'base.Country',
      source: { format: 'csv', document_ref: 'doc-ref-3' },
    },
  });

  const report = await executeImportJob(enqueued.importJobId);
  expect(report?.stats?.ok).toBe(2);

  const row = await ImportJob.Browse(enqueued.importJobId, ['ReportJson', 'ProgressDone', 'ProgressTotal'] as any);
  expect((row as any).ReportJson?.stats?.ok).toBe(2);
  expect((row as any).ProgressDone).toBe(2);
  expect((row as any).ProgressTotal).toBe(2);
});
