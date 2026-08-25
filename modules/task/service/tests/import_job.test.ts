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

function sampleSnapshot(sourceRef: string) {
  return {
    profile: 'record',
    caller: 'user',
    policy: 'atomic',
    model: 'base.Country',
    source: { format: 'csv', document_ref: sourceRef },
  };
}

async function expectAsyncError(run: () => Promise<unknown>, pattern: RegExp): Promise<void> {
  let thrown: any;
  try {
    await run();
  } catch (err) {
    thrown = err;
  }
  expect(thrown).toBeTruthy();
  expect(String(thrown?.message || thrown)).toMatch(pattern);
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
    specSnapshot: sampleSnapshot('doc-ref-1'),
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

test('ImportJob.EnqueueRecordImport validation paths', async () => {
  resetRequestContext();
  const jsCtx = ensureRequestContext();
  const previousUserId = jsCtx.identity.userId;
  jsCtx.identity.userId = '';
  await expectAsyncError(
    () =>
      ImportJob.EnqueueRecordImport({
        targetModel: 'base.Country',
        sourceRef: 'doc',
        specSnapshot: sampleSnapshot('doc'),
      }),
    /authenticated user/
  );
  jsCtx.identity.userId = previousUserId;

  await expectAsyncError(
    () =>
      ImportJob.EnqueueRecordImport({
        targetModel: '',
        sourceRef: 'doc',
        specSnapshot: sampleSnapshot('doc'),
      } as any),
    /targetModel and sourceRef/
  );

  await expectAsyncError(
    () =>
      ImportJob.EnqueueRecordImport({
        targetModel: 'base.Country',
        sourceRef: '',
        specSnapshot: sampleSnapshot('doc'),
      } as any),
    /targetModel and sourceRef/
  );

  await expectAsyncError(
    () =>
      ImportJob.EnqueueRecordImport({
        targetModel: 'base.Country',
        sourceRef: 'doc',
        specSnapshot: null as any,
      }),
    /specSnapshot/
  );

  await expectAsyncError(
    () =>
      ImportJob.EnqueueRecordImport({
        targetModel: 'base.Country',
        sourceRef: 'doc',
        profile: 'nope',
        specSnapshot: sampleSnapshot('doc'),
      }),
    /unsupported import profile/
  );

  await expectAsyncError(
    () =>
      ImportJob.EnqueueRecordImport({
        targetModel: 'base.Country',
        sourceRef: 'doc',
        policy: 'nope',
        specSnapshot: sampleSnapshot('doc'),
      }),
    /unsupported import policy/
  );

  const defaults = await ImportJob.EnqueueRecordImport({
    targetModel: 'base.Country',
    sourceRef: 'doc-defaults',
    profile: '  ',
    policy: '',
    specSnapshot: sampleSnapshot('doc-defaults'),
  });
  const row = await ImportJob.Browse(defaults.importJobId, ['Profile', 'Policy'] as any);
  expect((row as any).Profile).toBe('record');
  expect((row as any).Policy).toBe('atomic');
});

test('ImportJob field defaults apply on minimal Create', async () => {
  resetRequestContext();
  const row = await ImportJob.Create({
    TargetModel: 'base.Country',
    SourceRef: 'doc-field-defaults',
    SpecSnapshotJson: sampleSnapshot('doc-field-defaults'),
  } as Partial<ImportJob>);
  const loaded = await ImportJob.Browse(row.Id, ['Profile', 'Policy', 'DryRun', 'Direction'] as any);
  expect((loaded as any).Profile).toBe('record');
  expect((loaded as any).Policy).toBe('atomic');
  expect((loaded as any).DryRun).toBe(false);
  expect((loaded as any).Direction).toBe('import');
});

test('getQueueStatus joins ImportJob with Job.Status', async () => {
  resetRequestContext();
  const enqueued = await ImportJob.EnqueueRecordImport({
    targetModel: 'base.Country',
    sourceRef: 'doc-ref-2',
    specSnapshot: sampleSnapshot('doc-ref-2'),
  });
  const status = await getQueueStatus(enqueued.importJobId);
  expect(status.queueStatus).toBe('queued');
  expect(status.taskJobId).toBe(enqueued.taskJobId);
});

test('getQueueStatus error paths', async () => {
  resetRequestContext();
  await expectAsyncError(() => getQueueStatus(''), /importJobId is required/);
  await expectAsyncError(() => getQueueStatus('missing-import-job'), /not found/i);

  const browse = (ImportJob as any).Browse.bind(ImportJob);
  (ImportJob as any).Browse = async () => null;
  try {
    await expectAsyncError(() => getQueueStatus('ghost-import-job'), /import job ghost-import-job not found/);
  } finally {
    (ImportJob as any).Browse = browse;
  }

  const row = await ImportJob.Create({
    Profile: 'record',
    Policy: 'atomic',
    DryRun: false,
    TargetModel: 'base.Country',
    SourceRef: 'doc-unlinked',
    SpecSnapshotJson: sampleSnapshot('doc-unlinked'),
    Direction: 'import',
    ProgressDone: 0,
    ProgressTotal: 0,
  } as Partial<ImportJob>);
  await expectAsyncError(() => getQueueStatus(row.Id), /missing task job link/);
});

test('getQueueStatus joins empty progress and report fallbacks', async () => {
  resetRequestContext();
  const browse = (ImportJob as any).Browse.bind(ImportJob);
  const getJob = (Job as any).GetJob.bind(Job);
  (ImportJob as any).Browse = async () =>
    ({
      Id: 'import-fallback',
      TaskJobId: 'task-fallback',
      ProgressDone: null,
      ProgressTotal: undefined,
      ReportJson: null,
      ReportRef: '',
    }) as any;
  (Job as any).GetJob = async () => ({ Id: 'task-fallback', Status: '' }) as any;
  try {
    const status = await getQueueStatus('import-fallback');
    expect(status.queueStatus).toBe('');
    expect(status.progressDone).toBe(0);
    expect(status.progressTotal).toBe(0);
    expect(status.reportJson).toBeUndefined();
    expect(status.reportRef).toBeUndefined();
  } finally {
    (ImportJob as any).Browse = browse;
    (Job as any).GetJob = getJob;
  }
});

test('executeImportJob writes report via import bridge', async () => {
  resetRequestContext();
  const root: any = (globalThis as any).$choysum;
  const previousImport = root.import;
  try {
    root.import = {
      run: async () => ({
        profile: 'record',
        policy: 'atomic',
        stats: { total: 2, ok: 2, error: 0, skip: 0 },
        messages: [],
        artifact_ref: 'art-1',
      }),
    };

    const enqueued = await ImportJob.EnqueueRecordImport({
      targetModel: 'base.Country',
      sourceRef: 'doc-ref-3',
      specSnapshot: sampleSnapshot('doc-ref-3'),
    });

    const report = await ImportJob.ExecuteImport(enqueued.importJobId);
    expect(report?.stats?.ok).toBe(2);

    const row = await ImportJob.Browse(enqueued.importJobId, [
      'ReportJson',
      'ReportRef',
      'ProgressDone',
      'ProgressTotal',
    ] as any);
    expect((row as any).ReportJson?.stats?.ok).toBe(2);
    expect((row as any).ReportRef).toBe('art-1');
    expect((row as any).ProgressDone).toBe(2);
    expect((row as any).ProgressTotal).toBe(2);

    const status = await getQueueStatus(enqueued.importJobId);
    expect(status.reportRef).toBe('art-1');
    expect(status.reportJson?.stats?.ok).toBe(2);
  } finally {
    if (previousImport === undefined) {
      delete root.import;
    } else {
      root.import = previousImport;
    }
  }
});

test('executeImportJob and FinalizeReport error paths', async () => {
  resetRequestContext();
  const root: any = (globalThis as any).$choysum;
  const previousImport = root.import;
  try {
    await expectAsyncError(() => executeImportJob(''), /importJobId is required/);
    await expectAsyncError(() => ImportJob.FinalizeReport('', {}), /importJobId is required/);

    const withoutSnapshot = await ImportJob.Create({
      Profile: 'record',
      Policy: 'atomic',
      DryRun: false,
      TargetModel: 'base.Country',
      SourceRef: 'doc-no-snapshot',
      SpecSnapshotJson: { ok: true },
      Direction: 'import',
    } as Partial<ImportJob>);
    await (ImportJob as any).UpdateById(withoutSnapshot.Id, { SpecSnapshotJson: 0 } as any);
    await expectAsyncError(() => executeImportJob(withoutSnapshot.Id), /missing spec snapshot/);

    delete root.import;
    const enqueued = await ImportJob.EnqueueRecordImport({
      targetModel: 'base.Country',
      sourceRef: 'doc-no-bridge',
      specSnapshot: sampleSnapshot('doc-no-bridge'),
    });
    await expectAsyncError(() => executeImportJob(enqueued.importJobId), /import bridge is not available/);

    root.import = {
      run: async () => null,
    };
    const nullReportJob = await ImportJob.EnqueueRecordImport({
      targetModel: 'base.Country',
      sourceRef: 'doc-null-report',
      specSnapshot: sampleSnapshot('doc-null-report'),
    });
    const empty = await executeImportJob(nullReportJob.importJobId);
    expect(empty).toEqual({});

    await ImportJob.FinalizeReport(nullReportJob.importJobId, {
      Stats: { Total: 3 },
      artifactRef: 'art-alt',
    });
    const finalized = await ImportJob.Browse(nullReportJob.importJobId, ['ProgressTotal', 'ReportRef'] as any);
    expect((finalized as any).ProgressTotal).toBe(3);
    expect((finalized as any).ReportRef).toBe('art-alt');

    await ImportJob.FinalizeReport(nullReportJob.importJobId, null as any);
    const cleared = await ImportJob.Browse(nullReportJob.importJobId, ['ReportJson', 'ProgressTotal'] as any);
    expect((cleared as any).ReportJson).toEqual({});
    expect((cleared as any).ProgressTotal).toBe(0);
  } finally {
    if (previousImport === undefined) {
      delete root.import;
    } else {
      root.import = previousImport;
    }
  }
});
