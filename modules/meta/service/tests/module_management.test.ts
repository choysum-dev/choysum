// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import IrModule from '@/meta/service/models/ir_module';
import IrModuleIndex from '@/meta/service/models/ir_module_index';
import ModuleManagementLog from '@/meta/service/models/module_management_log';
import Job from '@/task/service/models/job';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const CTX_OVERRIDE_KEY = Symbol.for('choysum.ctx.override');
const CTX_FROZEN_KEY = Symbol.for('choysum.ctx.frozen');
const JOB_STORE = new Map<string, any>();

/**
 * Returns the mutable request-context root used by meta service tests.
 */
function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum;
  if (!root) {
    throw new Error('missing global $choysum; meta module tests must run under the QuickJS-first harness');
  }

  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};

  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};

  return jsCtx;
}

/**
 * Resets the request context to a permissive test baseline.
 */
function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    companyMode: 'skip',
    recordRuleMode: 'skip',
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = { userId: 'admin' };

  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
  delete (jsCtx as any)[CTX_OVERRIDE_KEY];
  delete (jsCtx as any)[CTX_FROZEN_KEY];
}

/**
 * Installs an in-memory Job model stub used by module management tests.
 */
function ensureJobMock(): void {
  const xid = (globalThis as any)?.$choysum?.xid?.New;
  const nextId = () => {
    const id = typeof xid === 'function' ? xid() : undefined;
    return (typeof id === 'string' && id.trim()) || String(Date.now());
  };

  JOB_STORE.clear();

  (Job as any).EnqueueJob = async (
    targetApp: string,
    fullMethod: string,
    payload: Record<string, any> = {},
    schedulerUserId: string,
    triggeredByUserId: string,
    runAfter?: string | number | Date,
    maxAttempts: number = 1,
    timeoutMs: number = 0
  ) => {
    const now = new Date();
    const runAfterAt = runAfter ? new Date(runAfter as any) : now;
    const job = {
      Id: `job_${nextId()}`,
      TargetApp: targetApp,
      FullMethod: fullMethod,
      PayloadJson: payload ?? {},
      SchedulerUserId: schedulerUserId,
      TriggeredByUserId: triggeredByUserId,
      Status: 'queued',
      RunAfter: runAfterAt,
      Attempt: 0,
      MaxAttempts: maxAttempts,
      TimeoutMs: timeoutMs,
      CreatedAt: now,
      UpdatedAt: now,
    };
    JOB_STORE.set(job.Id, job);
    return job;
  };

  (Job as any).GetJob = async (jobId: string) => {
    const job = JOB_STORE.get(String(jobId));
    if (!job) throw new Error(`job not found: ${jobId}`);
    return job;
  };

  (Job as any).UpdateById = async (jobId: string, values: Record<string, any>) => {
    const job = JOB_STORE.get(String(jobId));
    if (!job) throw new Error(`job not found: ${jobId}`);
    const updated = { ...job, ...values, UpdatedAt: new Date() };
    JOB_STORE.set(String(jobId), updated);
    return updated;
  };
}

/**
 * Seeds a queued job row into the in-memory Job store.
 */
function seedJob(
  jobId: string,
  payload: Record<string, any> = {},
  opts: { createdAt?: Date; finishedAt?: Date; attempt?: number; maxAttempts?: number } = {}
): void {
  const now = new Date();
  const createdAt = opts.createdAt ?? now;
  const finishedAt = opts.finishedAt ?? now;
  const job = {
    Id: String(jobId),
    TargetApp: 'meta',
    FullMethod: 'meta.module.install',
    PayloadJson: payload ?? {},
    SchedulerUserId: 'admin',
    TriggeredByUserId: 'admin',
    Status: 'queued',
    RunAfter: createdAt,
    Attempt: opts.attempt ?? 0,
    MaxAttempts: opts.maxAttempts ?? 1,
    TimeoutMs: 0,
    CreatedAt: createdAt,
    UpdatedAt: createdAt,
    FinishedAt: finishedAt,
  };
  JOB_STORE.set(String(jobId), job);
}

/**
 * Ensures the module management bridge and db shim exist on the global test root.
 */
function ensureModuleManagementBridge() {
  const root: any = (globalThis as any).$choysum;
  if (!root.moduleManagement) {
    root.moduleManagement = {};
  }
  if (!root.db) {
    root.db = {};
  }
  if (typeof root.db.query !== 'function') {
    root.db.query = async () => '[]';
  }
}

/**
 * Replaces the IrModuleIndex repository with a deterministic select builder stub.
 */
function mockIrModuleIndexRepo(rows: Array<Record<string, any>>): () => void {
  const original = (IrModuleIndex as any).getRepository;
  const builder: any = {
    select(sel: any) {
      if (typeof sel === 'function') {
        try {
          sel({ fn: { max: () => ({ as: () => undefined }) } });
        } catch {
          // ignore
        }
      }
      return builder;
    },
    where() {
      return builder;
    },
  };
  (IrModuleIndex as any).getRepository = () => ({
    selectQueryBuilder: () => builder,
    execute: async () => rows,
  });
  return () => {
    (IrModuleIndex as any).getRepository = original;
  };
}

/**
 * Replaces IrModuleIndex repository grouped-read methods for Search/Count tests.
 */
function mockIrModuleIndexGroupedRepo(groupRows: Array<Record<string, any>>, groupCount?: number): () => void {
  const original = (IrModuleIndex as any).getRepository;
  (IrModuleIndex as any).getRepository = () => ({
    readGroup: async () => groupRows,
    readGroupCount: async () => (typeof groupCount === 'number' ? groupCount : groupRows.length),
  });
  return () => {
    (IrModuleIndex as any).getRepository = original;
  };
}

/**
 * Replaces Job.Search with a fixed result set for the duration of a test.
 */
function mockJobSearch(result: any[]): () => void {
  const original = (Job as any).Search;
  (Job as any).Search = async () => result;
  return () => {
    (Job as any).Search = original;
  };
}

/**
 * Replaces BaseModel.Search used by IrModuleIndex aggregate Search.
 */
function mockIrModuleIndexBaseSearch(result: any[]): () => void {
  const baseModelCtor: any = Object.getPrototypeOf(IrModuleIndex);
  const original = baseModelCtor.Search;
  baseModelCtor.Search = async () => result;
  return () => {
    baseModelCtor.Search = original;
  };
}

/**
 * Generates a stable unique suffix for module-management test fixtures.
 */
function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

/**
 * Asserts an async operation throws and the message contains the given fragment.
 */
async function expectAsyncErrorContains(run: () => Promise<any>, fragment: string): Promise<void> {
  try {
    await run();
    throw new Error(`expected async error containing ${fragment}`);
  } catch (err: any) {
    const message = String(err?.message || err || '');
    expect(message.includes(fragment)).toBe(true);
  }
}

test('meta.IrModule PlanOperation returns blockers for missing module', async () => {
  resetRequestContext();
  ensureJobMock();

  const moduleName = uid('missing');
  const resp = await IrModule.PlanOperation({ action: 'uninstall', moduleName, baseRevision: '0' });

  expect(resp).toBeTruthy();
  expect(typeof resp.baseRevision).toBe('string');
  expect(resp.affectedModules.length).toBe(1);
  expect(resp.affectedModules[0].moduleName).toBe(moduleName);

  const hasBlocker = resp.blockers.some(item => item.code === 'MODULE_NOT_FOUND');
  expect(hasBlocker).toBe(true);

  if (resp.baseRevision !== '0') {
    const hasRisk = resp.risks.some(item => item.code === 'PLAN_REVISION_MISMATCH');
    expect(hasRisk).toBe(true);
  }
});

test('meta.IrModule PlanOperation adds risk when baseRevision mismatches', async () => {
  resetRequestContext();
  ensureJobMock();

  const moduleName = uid('rev_mismatch');
  const resp = await IrModule.PlanOperation({ action: 'install', moduleName, baseRevision: '999' });

  const hasRisk = resp.risks.some(item => item.code === 'PLAN_REVISION_MISMATCH');
  expect(hasRisk).toBe(true);
});

test('meta.IrModule RequestInstall writes operatorUserId into job payload', async () => {
  resetRequestContext();
  ensureJobMock();

  const moduleName = uid('req_install');
  const jobId = await IrModule.RequestInstall(moduleName, true);
  expect(jobId).toBeTruthy();

  const job = await Job.GetJob(jobId, ['Id', 'PayloadJson'] as any);
  expect(job).toBeTruthy();
  expect(job.PayloadJson?.moduleName).toBe(moduleName);
  expect(job.PayloadJson?.withDemo).toBe(true);
  expect(job.PayloadJson?.operatorUserId).toBe('admin');
});

test('meta.IrModule RequestUninstall writes operatorUserId into job payload', async () => {
  resetRequestContext();
  ensureJobMock();

  const moduleName = uid('req_uninstall');
  const jobId = await IrModule.RequestUninstall(moduleName);
  expect(jobId).toBeTruthy();

  const job = await Job.GetJob(jobId, ['Id', 'PayloadJson'] as any);
  expect(job).toBeTruthy();
  expect(job.PayloadJson?.moduleName).toBe(moduleName);
  expect(job.PayloadJson?.operatorUserId).toBe('admin');
});

test('meta.IrModule RequestUpgrade writes operatorUserId into job payload', async () => {
  resetRequestContext();
  ensureJobMock();

  const moduleName = uid('req_upgrade');
  const jobId = await IrModule.RequestUpgrade(moduleName);
  expect(jobId).toBeTruthy();

  const job = await Job.GetJob(jobId, ['Id', 'PayloadJson'] as any);
  expect(job).toBeTruthy();
  expect(job.PayloadJson?.moduleName).toBe(moduleName);
  expect(job.PayloadJson?.operatorUserId).toBe('admin');
});

test('meta.IrModule ExecuteInstall uses moduleManagement bridge and writes log', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const root: any = (globalThis as any).$choysum;
  const calls: any[] = [];
  root.moduleManagement.install = async (params: any) => {
    calls.push({ action: 'install', params });
    return { ok: true };
  };
  root.moduleManagement.reload = async () => ({ triggered: true, failed: false });

  const jobId = uid('job');
  const jsCtx = ensureRequestContext();
  jsCtx.ctx.jobId = jobId;
  seedJob(jobId, { moduleName: 'base', withDemo: true, operatorUserId: 'operator_1' }, { attempt: 2, maxAttempts: 5 });

  const result = await IrModule.ExecuteInstall('base', true, 'operator_1');
  expect(result.resultStatus).toBe('SUCCEEDED');
  expect(result.reload_web).toBe(true);
  expect(result.moduleName).toBe('base');
  expect(result.action).toBe('install');

  expect(calls.length).toBe(1);
  expect(calls[0].params.moduleName).toBe('base');
  expect(calls[0].params.withDemo).toBe(true);
  expect(calls[0].params.operatorUserId).toBe('operator_1');

  const logs = await ModuleManagementLog.Search(['JobId', '=', jobId] as any, { limit: 1 } as any);
  expect(logs?.length >= 1).toBe(true);
  expect(logs?.[0]?.Action).toBe('install');
  expect(logs?.[0]?.ModuleName).toBe('base');
  expect(logs?.[0]?.JobId).toBe(jobId);
  expect(logs?.[0]?.ResultStatus).toBe('SUCCEEDED');
  expect(logs?.[0]?.SummaryJson?.code).toBe('MODULE_INSTALLED');
  expect(logs?.[0]?.OperatorUserId).toBe('operator_1');
  expect(logs?.[0]?.JobCreatedAt).toBeTruthy();
  expect(logs?.[0]?.JobFinishedAt).toBeTruthy();
  expect(logs?.[0]?.Attempt).toBe(2);
  expect(logs?.[0]?.MaxAttempts).toBe(5);
});

test('meta.IrModule ExecuteInstall upserts module management log by jobId', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const root: any = (globalThis as any).$choysum;
  root.moduleManagement.install = async () => ({ ok: true });
  root.moduleManagement.reload = async () => ({ triggered: false, failed: false });

  const jobId = uid('job_upsert');
  const jsCtx = ensureRequestContext();
  jsCtx.ctx.jobId = jobId;

  await IrModule.ExecuteInstall('base', false, 'operator_1');
  await IrModule.ExecuteInstall('base', false, 'operator_1');

  const logs = await ModuleManagementLog.Search(['JobId', '=', jobId] as any, { limit: 10 } as any);
  expect(logs?.length).toBe(1);
  expect(logs?.[0]?.JobId).toBe(jobId);
});

test('meta.IrModule ExecuteUninstall uses moduleManagement bridge and writes log', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const root: any = (globalThis as any).$choysum;
  const calls: any[] = [];
  root.moduleManagement.uninstall = async (params: any) => {
    calls.push({ action: 'uninstall', params });
    return { ok: true };
  };
  root.moduleManagement.reload = async () => ({ triggered: true, failed: false });

  const jobId = uid('job_uninstall');
  const jsCtx = ensureRequestContext();
  jsCtx.ctx.jobId = jobId;
  seedJob(jobId, { moduleName: 'base', operatorUserId: 'operator_1' });

  const result = await IrModule.ExecuteUninstall('base', 'operator_1');
  expect(result.resultStatus).toBe('SUCCEEDED');
  expect(result.reload_web).toBe(true);
  expect(result.moduleName).toBe('base');
  expect(result.action).toBe('uninstall');

  expect(calls.length).toBe(1);
  expect(calls[0].params.moduleName).toBe('base');
  expect(calls[0].params.operatorUserId).toBe('operator_1');

  const logs = await ModuleManagementLog.Search(['JobId', '=', jobId] as any, { limit: 1 } as any);
  expect(logs?.length >= 1).toBe(true);
  expect(logs?.[0]?.Action).toBe('uninstall');
  expect(logs?.[0]?.ModuleName).toBe('base');
  expect(logs?.[0]?.OperatorUserId).toBe('operator_1');
  expect(logs?.[0]?.JobCreatedAt).toBeTruthy();
  expect(logs?.[0]?.JobFinishedAt).toBeTruthy();
});

test('meta.IrModule ExecuteUpgrade returns failed result and maps error fields', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const root: any = (globalThis as any).$choysum;
  root.moduleManagement.upgrade = async () => ({ ok: false, errorMessage: 'upgrade failed' });
  root.moduleManagement.reload = async () => ({ triggered: false, failed: false });

  const jobId = uid('job_upgrade_fail');
  const jsCtx = ensureRequestContext();
  jsCtx.ctx.jobId = jobId;
  seedJob(jobId, { moduleName: 'base', operatorUserId: 'operator_1' });

  const result = await IrModule.ExecuteUpgrade('base', 'operator_1');
  expect(result.resultStatus).toBe('FAILED');
  expect(result.reload_web).toBe(false);
  expect(result.errorDomain).toBe('MODULE_MANAGEMENT');
  expect(result.errorCode).toBe('OP_FAILED');
  expect(result.summary?.code).toBe('MODULE_OPERATION_FAILED');

  const logs = await ModuleManagementLog.Search(['JobId', '=', jobId] as any, { limit: 1 } as any);
  expect(logs?.length >= 1).toBe(true);
  expect(logs?.[0]?.ResultStatus).toBe('FAILED');
  expect(logs?.[0]?.ErrorDomain).toBe('MODULE_MANAGEMENT');
  expect(logs?.[0]?.ErrorCode).toBe('OP_FAILED');
  expect(logs?.[0]?.LastErrorJson?.message).toBe('upgrade failed');
  expect(logs?.[0]?.SummaryJson?.code).toBe('MODULE_OPERATION_FAILED');
});

test('meta.IrModule ExecuteInstall marks reload_failed when reload fails', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const root: any = (globalThis as any).$choysum;
  root.moduleManagement.install = async () => ({ ok: true });
  root.moduleManagement.reload = async () => ({ triggered: true, failed: true });

  const jobId = uid('job_reload_fail');
  const jsCtx = ensureRequestContext();
  jsCtx.ctx.jobId = jobId;

  const result = await IrModule.ExecuteInstall('base', false, 'operator_1');
  expect(result.resultStatus).toBe('SUCCEEDED');
  expect(result.reload_triggered).toBe(true);
  expect(result.reload_failed).toBe(true);
  expect(result.reload_web).toBe(false);
});

test('meta.IrModule GetOpStatus returns summary and reload flags', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('status');
  const job = await Job.EnqueueJob('meta', 'meta.IrModule/ExecuteInstall', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'succeeded',
      ResultJson: {
        resultStatus: 'SUCCEEDED',
        summary: { code: 'MODULE_INSTALLED', params: { moduleName } },
        reload_web: true,
        reload_triggered: true,
        reload_failed: false,
        moduleName,
        action: 'install',
        operatorUserId: 'admin',
      },
    } as any
  );

  const status = await IrModule.GetOpStatus(job.Id as any);
  expect(status.status).toBe('succeeded');
  expect(status.resultStatus).toBe('SUCCEEDED');
  expect(status.reload_web).toBe(true);
  expect(status.moduleName).toBe(moduleName);
  expect(status.action).toBe('install');
});

test('meta.IrModule GetOpStatus supports succeeded status with failed result', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('upgrade_failed');
  const job = await Job.EnqueueJob('meta', 'meta.IrModule/ExecuteUpgrade', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'succeeded',
      ResultJson: {
        resultStatus: 'FAILED',
        summary: { code: 'MODULE_OPERATION_FAILED', params: { moduleName, action: 'upgrade' } },
        reload_web: false,
        reload_triggered: false,
        reload_failed: false,
        moduleName,
        action: 'upgrade',
        operatorUserId: 'admin',
      },
    } as any
  );

  const status = await IrModule.GetOpStatus(job.Id as any);
  expect(status.status).toBe('succeeded');
  expect(status.resultStatus).toBe('FAILED');
  expect(status.failureKind).toBe('NON_RETRYABLE');
  expect(status.summary?.code).toBe('MODULE_OPERATION_FAILED');
  expect(status.reload_web).toBe(false);
});

test('meta.IrModule GetOpStatus maps retryable lock conflicts', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('lock_conflict');
  const job = await Job.EnqueueJob('meta', 'meta.IrModule/ExecuteUpgrade', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'failed',
      RunAfter: new Date(Date.now() + 2500),
      LastErrorJson: {
        domain: 'meta.lock',
        code: 'LEASE_CONFLICT',
        message: 'lease conflict',
        details: { retry_after_ms: 2500 },
      },
    } as any
  );

  const status = await IrModule.GetOpStatus(job.Id as any);
  expect(status.status).toBe('failed');
  expect(status.failureKind).toBe('RETRYABLE');
  expect(status.retryAfterMs).toBe(2500);
  expect(status.errorDomain).toBe('meta.lock');
  expect(status.errorCode).toBe('LEASE_CONFLICT');
  expect(status.summary?.code).toBe('MODULE_OPERATION_FAILED');
});

test('meta.IrModuleIndex RequestSync enqueues when stale and no batch sync', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreRepo = mockIrModuleIndexRepo([{ last_batch_sync_at: null }]);
  const restoreSearch = mockJobSearch([]);

  try {
    const jobId = await IrModuleIndex.RequestSync({ originType: 'local', ifStale: true, force: false });
    expect(jobId).toBeTruthy();
  } finally {
    restoreSearch();
    restoreRepo();
  }
});

test('meta.IrModuleIndex RequestSync skips enqueue when not stale', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreRepo = mockIrModuleIndexRepo([{ last_batch_sync_at: new Date() }]);
  const restoreSearch = mockJobSearch([]);

  let enqueueCalled = false;
  const originalEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => {
    enqueueCalled = true;
    return { Id: 'job_unexpected' };
  };

  try {
    const jobId = await IrModuleIndex.RequestSync({ originType: 'local', ifStale: true, force: false });
    expect(jobId).toBe('');
    expect(enqueueCalled).toBe(false);
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
    restoreRepo();
  }
});

test('meta.IrModuleIndex RequestSync(all) skips enqueue when both origins are fresh', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreRepo = mockIrModuleIndexRepo([{ last_batch_sync_at: new Date() }]);
  const restoreSearch = mockJobSearch([]);

  let enqueueCalled = false;
  const originalEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => {
    enqueueCalled = true;
    return { Id: 'job_unexpected' };
  };

  try {
    const jobId = await IrModuleIndex.RequestSync({ ifStale: true, force: false });
    expect(jobId).toBe('');
    expect(enqueueCalled).toBe(false);
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
    restoreRepo();
  }
});

test('meta.IrModuleIndex RequestSync(all) enqueues when stale', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreRepo = mockIrModuleIndexRepo([{ last_batch_sync_at: null }]);
  const restoreSearch = mockJobSearch([]);

  try {
    const jobId = await IrModuleIndex.RequestSync({ ifStale: true, force: false });
    expect(jobId).toBeTruthy();
  } finally {
    restoreSearch();
    restoreRepo();
  }
});

test('meta.IrModuleIndex RequestSync reuses running job for non-force requests', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreSearch = mockJobSearch([{ Id: 'job_running_sync', PayloadJson: { originType: 'local' } }]);
  let enqueueCalled = false;
  const originalEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => {
    enqueueCalled = true;
    return { Id: 'job_unexpected' };
  };

  try {
    const jobId = await IrModuleIndex.RequestSync({ originType: 'local', ifStale: true, force: false });
    expect(jobId).toBe('job_running_sync');
    expect(enqueueCalled).toBe(false);
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
  }
});

test('meta.IrModuleIndex RequestSync ignores incompatible running origin and enqueues', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreSearch = mockJobSearch([{ Id: 'job_running_registry', PayloadJson: { originType: 'registry' } }]);
  const originalEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => ({ Id: 'job_local_sync' });

  try {
    const jobId = await IrModuleIndex.RequestSync({ originType: 'local', ifStale: true, force: false });
    expect(jobId).toBe('job_local_sync');
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
  }
});

test('meta.IrModuleIndex RequestSync(force) does not reuse running job', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreSearch = mockJobSearch([{ Id: 'job_running_sync', PayloadJson: { originType: 'local' } }]);
  const originalEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => ({ Id: 'job_forced_sync' });

  try {
    const jobId = await IrModuleIndex.RequestSync({ originType: 'local', force: true, ifStale: false });
    expect(jobId).toBe('job_forced_sync');
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
  }
});

test('meta.IrModuleIndex RequestSync rejects invalid originType', async () => {
  resetRequestContext();
  ensureJobMock();

  await expectAsyncErrorContains(
    () => IrModuleIndex.RequestSync({ originType: 'remote' as any, force: true, ifStale: false }),
    'originType must be one of: local, registry, all'
  );
});

test('meta.IrModuleIndex Sync rejects invalid originType before bridge call', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();

  const root: any = (globalThis as any).$choysum;
  let called = false;
  root.moduleManagement.syncIndex = async () => {
    called = true;
    return { ok: true };
  };

  await expectAsyncErrorContains(() => IrModuleIndex.Sync('remote' as any, false), 'originType must be one of: local, registry, all');
  expect(called).toBe(false);
});

test('meta.IrModuleIndex Search honors requested fields after aggregation', async () => {
  resetRequestContext();

  const now = new Date();
  const restoreGroupedRepo = mockIrModuleIndexGroupedRepo([{ ModuleName: 'auth' }]);
  const restoreBaseSearch = mockIrModuleIndexBaseSearch([
    {
      Id: 'idx_local',
      ModuleName: 'auth',
      OriginType: 'local',
      OriginRef: 'local',
      Available: true,
      Version: '1.0.0',
      ManifestJson: { source: 'local' },
      LocalPath: '/modules/auth',
      LastSyncAt: now,
      LastBatchSyncAt: now,
      SyncRevision: 'r1',
      LastErrorMessage: '',
      InstalledStatus: 'installed',
      InstalledVersion: '1.0.0',
    },
    {
      Id: 'idx_registry',
      ModuleName: 'auth',
      OriginType: 'registry',
      OriginRef: '@choysum-dev/auth',
      Available: true,
      Version: '2.0.0',
      ManifestJson: { source: 'registry' },
      LocalPath: '',
      LastSyncAt: now,
      LastBatchSyncAt: now,
      SyncRevision: 'r2',
      LastErrorMessage: '',
      InstalledStatus: 'installed',
      InstalledVersion: '1.0.0',
    },
  ]);

  try {
    const rows = await (IrModuleIndex as any).Search([], { fields: ['ModuleName', 'RegistryVersion'], limit: 10 });
    expect(Array.isArray(rows)).toBe(true);
    expect(rows.length).toBe(1);
    expect(rows[0].ModuleName).toBe('auth');
    expect(rows[0].RegistryVersion).toBe('2.0.0');
    expect(rows[0].OriginType).toBeUndefined();
    expect(rows[0].LocalVersion).toBeUndefined();
  } finally {
    restoreGroupedRepo();
    restoreBaseSearch();
  }
});

test('meta.IrModuleIndex Count uses grouped module count', async () => {
  resetRequestContext();

  const restoreGroupedRepo = mockIrModuleIndexGroupedRepo([], 7);
  try {
    const total = await (IrModuleIndex as any).Count([], {});
    expect(total).toBe(7);
  } finally {
    restoreGroupedRepo();
  }
});
