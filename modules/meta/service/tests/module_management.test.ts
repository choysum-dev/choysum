// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RepositoryFactory } from '@/core/service/orm/repository';
import { createServiceByModel } from '@/core/service/rpc';
import MetaModule from '@/meta/service/models/module';
import MetaModuleIndex from '@/meta/service/models/module_index';
import { DEFAULT_MODULE_INDEX_SEARCH } from '@/meta/service/models/_module_index_query';
import ModuleManagementLog from '@/meta/service/models/module_management_log';
import type JobModel from '@/task/service/models/job';

const Job = createServiceByModel<typeof JobModel>('task.Job');

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
 * Replaces the MetaModuleIndex repository with a deterministic select builder stub.
 */
function mockMetaModuleIndexRepo(rows: Array<Record<string, any>>): () => void {
  const original = RepositoryFactory.getRepository(MetaModuleIndex as any);
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
  RepositoryFactory.setRepository(MetaModuleIndex as any, {
    selectQueryBuilder: () => builder,
    execute: async () => rows,
  } as any);
  return () => {
    RepositoryFactory.setRepository(MetaModuleIndex as any, original);
  };
}

type MetaModuleIndexGroupedRepoStub = {
  readGroup?: () => Promise<Array<Record<string, any>>>;
  readGroupCount?: () => Promise<number>;
};

/**
 * Replaces MetaModuleIndex repository grouped-read methods for Search/Count tests.
 */
function mockMetaModuleIndexGroupedRepo(
  groupRows: Array<Record<string, any>>,
  groupCount?: number,
  overrides?: MetaModuleIndexGroupedRepoStub
): () => void {
  const original = RepositoryFactory.getRepository(MetaModuleIndex as any);
  RepositoryFactory.setRepository(MetaModuleIndex as any, {
    readGroup: async () => groupRows,
    readGroupCount: async () => (typeof groupCount === 'number' ? groupCount : groupRows.length),
    ...(overrides || {}),
  } as any);
  return () => {
    RepositoryFactory.setRepository(MetaModuleIndex as any, original);
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
 * Replaces BaseModel.Search used by MetaModuleIndex aggregate Search.
 */
function mockMetaModuleIndexBaseSearch(result: any[]): () => void {
  const baseModelCtor: any = Object.getPrototypeOf(MetaModuleIndex);
  const original = baseModelCtor.Search;
  baseModelCtor.Search = async () => result;
  return () => {
    baseModelCtor.Search = original;
  };
}

/**
 * Replaces MetaModule.Search used by MetaModuleIndex aggregate Search status merge.
 */
function mockMetaModuleSearch(result: any[]): () => void {
  const original = (MetaModule as any).Search;
  (MetaModule as any).Search = async () => result;
  return () => {
    (MetaModule as any).Search = original;
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
  let captured: unknown;
  try {
    await run();
  } catch (err: any) {
    captured = err;
  }

  if (captured == null) {
    throw new Error('__expected_async_throw__');
  }

  const message = String((captured as any)?.message || captured || '');
  expect(message.includes(fragment)).toBe(true);
}

test('meta.MetaModule PlanOperation returns blockers for missing module', async () => {
  resetRequestContext();
  ensureJobMock();

  const moduleName = uid('missing');
  const resp = await MetaModule.PlanOperation({ action: 'uninstall', moduleName, baseRevision: '0' });

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

test('meta.MetaModule PlanOperation adds risk when baseRevision mismatches', async () => {
  resetRequestContext();
  ensureJobMock();

  const moduleName = uid('rev_mismatch');
  const resp = await MetaModule.PlanOperation({ action: 'install', moduleName, baseRevision: '999' });

  const hasRisk = resp.risks.some(item => item.code === 'PLAN_REVISION_MISMATCH');
  expect(hasRisk).toBe(true);
});

test('meta.MetaModule RequestInstall writes operatorUserId into job payload', async () => {
  resetRequestContext();
  ensureJobMock();

  const published: any[] = [];
  const { __setMetaPublishTipForTest } = await import('../tips');
  __setMetaPublishTipForTest(event => {
    published.push(event);
  });
  try {
    const moduleName = uid('req_install');
    const jobId = await MetaModule.RequestInstall(moduleName, true);
    expect(jobId).toBeTruthy();

    const job = await Job.GetJob(jobId, ['Id', 'PayloadJson'] as any);
    expect(job).toBeTruthy();
    expect(job.PayloadJson?.moduleName).toBe(moduleName);
    expect(job.PayloadJson?.withDemo).toBe(true);
    expect(job.PayloadJson?.operatorUserId).toBe('admin');
    expect(published.some(event => event?.payload?.jobId === jobId)).toBe(true);
  } finally {
    __setMetaPublishTipForTest(undefined);
  }
});

test('meta.MetaModule RequestInstall publishes lease-conflict tip when forced', async () => {
  resetRequestContext();
  ensureJobMock();

  const priorEnv = (globalThis as any).__choysumBackendEnv;
  const published: any[] = [];
  const { __setMetaPublishTipForTest } = await import('../tips');
  __setMetaPublishTipForTest(event => {
    published.push(event);
  });
  (globalThis as any).__choysumBackendEnv = {
    ...(priorEnv || {}),
    CHOYSUM_E2E_FORCE_LOCK_CONFLICT: '1',
  };
  try {
    const moduleName = uid('req_lock');
    const jobId = await MetaModule.RequestInstall(moduleName);
    expect(jobId).toBeTruthy();
    const tip = published.find(event => event?.source === 'meta.MetaModule.leaseConflict');
    expect(tip).toBeTruthy();
    expect(tip.payload.jobId).toBe(jobId);
  } finally {
    if (priorEnv === undefined) delete (globalThis as any).__choysumBackendEnv;
    else (globalThis as any).__choysumBackendEnv = priorEnv;
    __setMetaPublishTipForTest(undefined);
  }
});

test('meta.MetaModule RequestUninstall writes operatorUserId into job payload', async () => {
  resetRequestContext();
  ensureJobMock();

  const moduleName = uid('req_uninstall');
  const jobId = await MetaModule.RequestUninstall(moduleName);
  expect(jobId).toBeTruthy();

  const job = await Job.GetJob(jobId, ['Id', 'PayloadJson'] as any);
  expect(job).toBeTruthy();
  expect(job.PayloadJson?.moduleName).toBe(moduleName);
  expect(job.PayloadJson?.operatorUserId).toBe('admin');
});

test('meta.MetaModule RequestUpgrade writes operatorUserId into job payload', async () => {
  resetRequestContext();
  ensureJobMock();

  const moduleName = uid('req_upgrade');
  const jobId = await MetaModule.RequestUpgrade(moduleName);
  expect(jobId).toBeTruthy();

  const job = await Job.GetJob(jobId, ['Id', 'PayloadJson'] as any);
  expect(job).toBeTruthy();
  expect(job.PayloadJson?.moduleName).toBe(moduleName);
  expect(job.PayloadJson?.operatorUserId).toBe('admin');
});

test('meta.MetaModule ExecuteInstall uses moduleManagement bridge and writes log', async () => {
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

  const result = await MetaModule.ExecuteInstall('base', true, 'operator_1');
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

test('meta.MetaModule ExecuteInstall upserts module management log by jobId', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const root: any = (globalThis as any).$choysum;
  root.moduleManagement.install = async () => ({ ok: true });
  root.moduleManagement.reload = async () => ({ triggered: false, failed: false });

  const jobId = uid('job_upsert');
  const jsCtx = ensureRequestContext();
  jsCtx.ctx.jobId = jobId;

  await MetaModule.ExecuteInstall('base', false, 'operator_1');
  await MetaModule.ExecuteInstall('base', false, 'operator_1');

  const logs = await ModuleManagementLog.Search(['JobId', '=', jobId] as any, { limit: 10 } as any);
  expect(logs?.length).toBe(1);
  expect(logs?.[0]?.JobId).toBe(jobId);
});

test('meta.MetaModule ExecuteUninstall uses moduleManagement bridge and writes log', async () => {
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

  const result = await MetaModule.ExecuteUninstall('base', 'operator_1');
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

test('meta.MetaModule ExecuteUpgrade returns failed result and maps error fields', async () => {
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

  const result = await MetaModule.ExecuteUpgrade('base', 'operator_1');
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

test('meta.MetaModule ExecuteInstall marks reload_failed when reload fails', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const root: any = (globalThis as any).$choysum;
  root.moduleManagement.install = async () => ({ ok: true });
  root.moduleManagement.reload = async () => ({ triggered: true, failed: true });

  const jobId = uid('job_reload_fail');
  const jsCtx = ensureRequestContext();
  jsCtx.ctx.jobId = jobId;

  const result = await MetaModule.ExecuteInstall('base', false, 'operator_1');
  expect(result.resultStatus).toBe('SUCCEEDED');
  expect(result.reload_triggered).toBe(true);
  expect(result.reload_failed).toBe(true);
  expect(result.reload_web).toBe(false);
});

test('meta.MetaModule GetOpStatus returns summary and reload flags', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('status');
  const job = await Job.EnqueueJob('meta', 'meta.MetaModule/ExecuteInstall', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

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

  const status = await MetaModule.GetOpStatus(job.Id as any);
  expect(status.status).toBe('succeeded');
  expect(status.resultStatus).toBe('SUCCEEDED');
  expect(status.reload_web).toBe(true);
  expect(status.moduleName).toBe(moduleName);
  expect(status.action).toBe('install');
});

test('meta.MetaModule GetOpStatus supports succeeded status with failed result', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('upgrade_failed');
  const job = await Job.EnqueueJob('meta', 'meta.MetaModule/ExecuteUpgrade', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

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

  const status = await MetaModule.GetOpStatus(job.Id as any);
  expect(status.status).toBe('succeeded');
  expect(status.resultStatus).toBe('FAILED');
  expect(status.failureKind).toBe('NON_RETRYABLE');
  expect(status.summary?.code).toBe('MODULE_OPERATION_FAILED');
  expect(status.reload_web).toBe(false);
});

test('meta.MetaModule GetOpStatus maps plain failures as non-retryable', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('plain_fail');
  const job = await Job.EnqueueJob('meta', 'meta.MetaModule/ExecuteUpgrade', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'failed',
      LastErrorJson: { message: 'boom' },
      ResultJson: null,
    } as any
  );

  const status = await MetaModule.GetOpStatus(job.Id as any);
  expect(status.status).toBe('failed');
  expect(status.resultStatus).toBe('FAILED');
  expect(status.failureKind).toBe('NON_RETRYABLE');
});

test('meta.MetaModule GetOpStatus maps retryable lock conflicts via errorDomain/errorCode', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('lock_conflict_alt');
  const job = await Job.EnqueueJob('meta', 'meta.MetaModule/ExecuteUpgrade', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'failed',
      LastErrorJson: {
        errorDomain: 'meta.lock',
        errorCode: 'LEASE_CONFLICT',
        message: 'lease conflict',
        details: { retry_after_ms: 1000 },
      },
    } as any
  );

  const status = await MetaModule.GetOpStatus(job.Id as any);
  expect(status.failureKind).toBe('RETRYABLE');
  expect(status.retryAfterMs).toBe(1000);
});

test('meta.MetaModule GetOpStatus classifies ResultJson lock conflicts without LastErrorJson', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('result_lock_conflict');
  const job = await Job.EnqueueJob('meta', 'meta.MetaModule/ExecuteUpgrade', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

  // Logical executeModuleOp failures are stored on ResultJson; the worker leaves LastErrorJson unset.
  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'succeeded',
      LastErrorJson: null,
      ResultJson: {
        resultStatus: 'FAILED',
        errorDomain: 'meta.lock',
        errorCode: 'LEASE_CONFLICT',
        errorMessage: 'lease conflict',
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

  const status = await MetaModule.GetOpStatus(job.Id as any);
  expect(status.status).toBe('succeeded');
  expect(status.resultStatus).toBe('FAILED');
  expect(status.failureKind).toBe('RETRYABLE');
  expect(status.errorDomain).toBe('meta.lock');
  expect(status.errorCode).toBe('LEASE_CONFLICT');

  // Unstructured LastErrorJson must not hide structured ResultJson lock conflicts.
  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'succeeded',
      LastErrorJson: { message: 'failed' },
      ResultJson: {
        resultStatus: 'FAILED',
        errorDomain: 'meta.lock',
        errorCode: 'LEASE_CONFLICT',
        errorMessage: 'lease conflict',
        summary: { code: 'MODULE_OPERATION_FAILED', params: { moduleName, action: 'upgrade' } },
        moduleName,
        action: 'upgrade',
        operatorUserId: 'admin',
      },
    } as any
  );
  const unstructuredErr = await MetaModule.GetOpStatus(job.Id as any);
  expect(unstructuredErr.failureKind).toBe('RETRYABLE');
  expect(unstructuredErr.errorDomain).toBe('meta.lock');
  expect(unstructuredErr.errorCode).toBe('LEASE_CONFLICT');

  // FAILED with no LastErrorJson and no ResultJson error fields → pickErrString(null).
  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'failed',
      LastErrorJson: null,
      ResultJson: { resultStatus: 'FAILED' },
    } as any
  );
  const emptySource = await MetaModule.GetOpStatus(job.Id as any);
  expect(emptySource.failureKind).toBe('NON_RETRYABLE');

  // ResultJson carries only errorCode (no domain) for resolveFailureSource code branch.
  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'failed',
      LastErrorJson: null,
      ResultJson: { resultStatus: 'FAILED', errorCode: 'ONLY_CODE' },
    } as any
  );
  const codeOnlyPath = await MetaModule.GetOpStatus(job.Id as any);
  expect(codeOnlyPath.failureKind).toBe('NON_RETRYABLE');
});

test('meta.MetaModule GetOpStatus maps mismatched lock codes as non-retryable', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('lock_mismatch');
  const job = await Job.EnqueueJob('meta', 'meta.MetaModule/ExecuteUpgrade', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'failed',
      LastErrorJson: {
        domain: 'meta.lock',
        code: 'OTHER',
        message: 'not a lease conflict',
      },
    } as any
  );

  const status = await MetaModule.GetOpStatus(job.Id as any);
  expect(status.failureKind).toBe('NON_RETRYABLE');

  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'failed',
      LastErrorJson: {
        domain: '',
        errorDomain: 'module_management',
        code: '',
        errorCode: 'LOCK_LEASE_LOST',
        message: 'lease lost via alt keys',
      },
    } as any
  );
  const alt = await MetaModule.GetOpStatus(job.Id as any);
  expect(alt.failureKind).toBe('RETRYABLE');

  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'failed',
      LastErrorJson: {
        domain: 'module_management',
        code: 'OTHER',
        message: 'not lease lost',
      },
    } as any
  );
  const mismatchMgmt = await MetaModule.GetOpStatus(job.Id as any);
  expect(mismatchMgmt.failureKind).toBe('NON_RETRYABLE');
});

test('meta.MetaModule GetOpStatus maps cancelled jobs as non-retryable failures', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('cancelled_op');
  const job = await Job.EnqueueJob('meta', 'meta.MetaModule/ExecuteUpgrade', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

  await (Job as any).UpdateById(job.Id as any, { Status: 'cancelled', ResultJson: null } as any);

  const status = await MetaModule.GetOpStatus(job.Id as any);
  expect(status.status).toBe('cancelled');
  expect(status.resultStatus).toBe('FAILED');
  expect(status.failureKind).toBe('NON_RETRYABLE');
});

test('meta.MetaModule GetOpStatus maps retryable lock lease lost', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('lock_lease_lost');
  const job = await Job.EnqueueJob('meta', 'meta.MetaModule/ExecuteUpgrade', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

  await (Job as any).UpdateById(
    job.Id as any,
    {
      Status: 'failed',
      LastErrorJson: { domain: 'module_management', code: 'LOCK_LEASE_LOST', message: 'lease lost' },
      ResultJson: null,
    } as any
  );

  const status = await MetaModule.GetOpStatus(job.Id as any);
  expect(status.status).toBe('failed');
  expect(status.resultStatus).toBe('FAILED');
  expect(status.failureKind).toBe('RETRYABLE');
  expect(status.errorDomain).toBe('module_management');
  expect(status.errorCode).toBe('LOCK_LEASE_LOST');
});

test('meta.MetaModule GetOpStatus maps retryable lock conflicts', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();
  ensureJobMock();

  const moduleName = uid('lock_conflict');
  const job = await Job.EnqueueJob('meta', 'meta.MetaModule/ExecuteUpgrade', { moduleName, operatorUserId: 'admin' }, 'admin', 'admin');

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

  const status = await MetaModule.GetOpStatus(job.Id as any);
  expect(status.status).toBe('failed');
  expect(status.failureKind).toBe('RETRYABLE');
  expect(status.retryAfterMs).toBe(2500);
  expect(status.errorDomain).toBe('meta.lock');
  expect(status.errorCode).toBe('LEASE_CONFLICT');
  expect(status.summary?.code).toBe('MODULE_OPERATION_FAILED');
});

test('meta.MetaModuleIndex RequestSync enqueues when stale and no batch sync', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreRepo = mockMetaModuleIndexRepo([{ last_batch_sync_at: null }]);
  const restoreSearch = mockJobSearch([]);

  try {
    const jobId = await MetaModuleIndex.RequestSync({ originType: 'local', ifStale: true, force: false });
    expect(jobId).toBeTruthy();
  } finally {
    restoreSearch();
    restoreRepo();
  }
});

test('meta.MetaModuleIndex RequestSync skips enqueue when not stale', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreRepo = mockMetaModuleIndexRepo([{ last_batch_sync_at: new Date() }]);
  const restoreSearch = mockJobSearch([]);

  let enqueueCalled = false;
  const originalEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => {
    enqueueCalled = true;
    return { Id: 'job_unexpected' };
  };

  try {
    const jobId = await MetaModuleIndex.RequestSync({ originType: 'local', ifStale: true, force: false });
    expect(jobId).toBe('');
    expect(enqueueCalled).toBe(false);
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
    restoreRepo();
  }
});

test('meta.MetaModuleIndex RequestSync(all) skips enqueue when both origins are fresh', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreRepo = mockMetaModuleIndexRepo([{ last_batch_sync_at: new Date() }]);
  const restoreSearch = mockJobSearch([]);

  let enqueueCalled = false;
  const originalEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => {
    enqueueCalled = true;
    return { Id: 'job_unexpected' };
  };

  try {
    const jobId = await MetaModuleIndex.RequestSync({ ifStale: true, force: false });
    expect(jobId).toBe('');
    expect(enqueueCalled).toBe(false);
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
    restoreRepo();
  }
});

test('meta.MetaModuleIndex RequestSync(all) enqueues when stale', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreRepo = mockMetaModuleIndexRepo([{ last_batch_sync_at: null }]);
  const restoreSearch = mockJobSearch([]);

  try {
    const jobId = await MetaModuleIndex.RequestSync({ ifStale: true, force: false });
    expect(jobId).toBeTruthy();
  } finally {
    restoreSearch();
    restoreRepo();
  }
});

test('meta.MetaModuleIndex RequestSync ignores invalid running origin and enqueues', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreSearch = mockJobSearch([{ Id: 'job_running_bad_origin', PayloadJson: { originType: 'remote' } }]);
  const originalEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => ({ Id: 'job_after_skip_invalid_origin' });

  try {
    const jobId = await MetaModuleIndex.RequestSync({ originType: 'local', ifStale: true, force: false });
    expect(jobId).toBe('job_after_skip_invalid_origin');
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
  }
});

test('meta.MetaModuleIndex RequestSync reuses running job for non-force requests', async () => {
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
    const jobId = await MetaModuleIndex.RequestSync({ originType: 'local', ifStale: true, force: false });
    expect(jobId).toBe('job_running_sync');
    expect(enqueueCalled).toBe(false);
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
  }
});

test('meta.MetaModuleIndex RequestSync ignores incompatible running origin and enqueues', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreSearch = mockJobSearch([{ Id: 'job_running_registry', PayloadJson: { originType: 'registry' } }]);
  const originalEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => ({ Id: 'job_local_sync' });

  try {
    const jobId = await MetaModuleIndex.RequestSync({ originType: 'local', ifStale: true, force: false });
    expect(jobId).toBe('job_local_sync');
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
  }
});

test('meta.MetaModuleIndex RequestSync(force) does not reuse running job', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreSearch = mockJobSearch([{ Id: 'job_running_sync', PayloadJson: { originType: 'local' } }]);
  const originalEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => ({ Id: 'job_forced_sync' });

  try {
    const jobId = await MetaModuleIndex.RequestSync({ originType: 'local', force: true, ifStale: false });
    expect(jobId).toBe('job_forced_sync');
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
  }
});

test('meta.MetaModuleIndex RequestSync defaults null originType to all', async () => {
  resetRequestContext();
  ensureJobMock();

  const restoreSearch = mockJobSearch([]);
  const originalEnqueue = (Job as any).EnqueueJob;
  let enqueuedOrigin: unknown;
  (Job as any).EnqueueJob = async (_app: string, _method: string, payload: any) => {
    enqueuedOrigin = payload?.originType;
    return { Id: 'job_default_all_origin' };
  };

  try {
    const jobId = await MetaModuleIndex.RequestSync({ originType: null as any, force: true, ifStale: false });
    expect(jobId).toBe('job_default_all_origin');
    expect(enqueuedOrigin).toBe('all');
  } finally {
    (Job as any).EnqueueJob = originalEnqueue;
    restoreSearch();
  }
});

test('meta.MetaModuleIndex RequestSync rejects invalid originType', async () => {
  resetRequestContext();
  ensureJobMock();

  await expectAsyncErrorContains(
    () => MetaModuleIndex.RequestSync({ originType: 'remote' as any, force: true, ifStale: false }),
    'originType'
  );
});

test('meta.MetaModuleIndex Sync defaults omitted originType to all', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();

  const root: any = (globalThis as any).$choysum;
  const seen: Array<{ originType?: string; force?: boolean }> = [];
  root.moduleManagement.syncIndex = async (params: any) => {
    seen.push(params);
    return { ok: true };
  };

  await MetaModuleIndex.Sync(undefined, false);
  await MetaModuleIndex.Sync(null as any, true);
  await MetaModuleIndex.Sync('local', false);

  expect(seen).toEqual([
    { originType: 'all', force: false },
    { originType: 'all', force: true },
    { originType: 'local', force: false },
  ]);
});

test('meta.MetaModuleIndex Sync rejects invalid originType before bridge call', async () => {
  resetRequestContext();
  ensureModuleManagementBridge();

  const root: any = (globalThis as any).$choysum;
  let called = false;
  root.moduleManagement.syncIndex = async () => {
    called = true;
    return { ok: true };
  };

  await expectAsyncErrorContains(() => MetaModuleIndex.Sync('remote' as any, false), 'originType');
  expect(called).toBe(false);
});

test('meta.MetaModuleIndex Search honors requested fields after aggregation', async () => {
  resetRequestContext();

  const now = new Date();
  const restoreGroupedRepo = mockMetaModuleIndexGroupedRepo([{ ModuleName: 'auth' }]);
  const restoreBaseSearch = mockMetaModuleIndexBaseSearch([
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
    const rows = await (MetaModuleIndex as any).Search(undefined, { fields: ['ModuleName', 'RegistryVersion'], limit: 10 });
    expect(Array.isArray(rows)).toBe(true);
    expect(rows.length).toBe(1);
    expect(typeof rows[0].toPlainObject).toBe('function');
    expect(rows[0].ModuleName).toBe('auth');
    expect(rows[0].RegistryVersion).toBe('2.0.0');
    expect(rows[0].OriginType).toBeUndefined();
    expect(rows[0].LocalVersion).toBeUndefined();
  } finally {
    restoreGroupedRepo();
    restoreBaseSearch();
  }
});

test('meta.MetaModuleIndex Search projection returns model instances and blocks dangerous fields', async () => {
  resetRequestContext();

  const now = new Date();
  const restoreGroupedRepo = mockMetaModuleIndexGroupedRepo([{ ModuleName: 'auth' }]);
  const restoreBaseSearch = mockMetaModuleIndexBaseSearch([
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
  ]);

  try {
    const rows = await (MetaModuleIndex as any).Search(DEFAULT_MODULE_INDEX_SEARCH, { fields: ['ModuleName', '__proto__', 'constructor', 'prototype'], limit: 10 });
    expect(Array.isArray(rows)).toBe(true);
    expect(rows.length).toBe(1);
    expect(rows[0].ModuleName).toBe('auth');
    expect(typeof rows[0].toPlainObject).toBe('function');

    const plain = rows[0].toPlainObject();
    expect(Object.getPrototypeOf(plain)).toBe(Object.prototype);
    expect(Object.prototype.hasOwnProperty.call(plain, '__proto__')).toBe(false);
    expect(Object.prototype.hasOwnProperty.call(plain, 'constructor')).toBe(false);
    expect(Object.prototype.hasOwnProperty.call(plain, 'prototype')).toBe(false);
  } finally {
    restoreGroupedRepo();
    restoreBaseSearch();
  }
});

test('meta.MetaModuleIndex Search prefers MetaModule status over aggregate default uninstalled', async () => {
  resetRequestContext();

  const now = new Date();
  const restoreGroupedRepo = mockMetaModuleIndexGroupedRepo([{ ModuleName: 'auth' }]);
  const restoreBaseSearch = mockMetaModuleIndexBaseSearch([
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
    },
  ]);
  const restoreMetaModuleSearch = mockMetaModuleSearch([
    {
      Name: 'auth',
      Status: 'installed',
      Version: '1.0.0',
    },
  ]);

  try {
    const rows = await (MetaModuleIndex as any).Search(DEFAULT_MODULE_INDEX_SEARCH, { fields: ['ModuleName', 'InstalledStatus', 'InstalledVersion'], limit: 10 });
    expect(Array.isArray(rows)).toBe(true);
    expect(rows.length).toBe(1);
    expect(rows[0].ModuleName).toBe('auth');
    expect(rows[0].InstalledStatus).toBe('installed');
    expect(rows[0].InstalledVersion).toBe('1.0.0');
  } finally {
    restoreGroupedRepo();
    restoreBaseSearch();
    restoreMetaModuleSearch();
  }
});

test('meta.MetaModuleIndex Count uses grouped module count', async () => {
  resetRequestContext();

  const restoreGroupedRepo = mockMetaModuleIndexGroupedRepo([], 7);
  try {
    const total = await (MetaModuleIndex as any).Count();
    expect(total).toBe(7);
    // Omit options so `options || {}` false branch is covered for patch coverage.
    const totalDefaultOpts = await (MetaModuleIndex as any).Count();
    expect(totalDefaultOpts).toBe(7);
  } finally {
    restoreGroupedRepo();
  }
});

test('meta.MetaModuleIndex Search hydrates with no field filter and empty aggregate rows', async () => {
  resetRequestContext();

  const restoreGroupedRepo = mockMetaModuleIndexGroupedRepo([{ ModuleName: 'ghost' }]);
  const restoreBaseSearch = mockMetaModuleIndexBaseSearch([]);

  try {
    const rows = await (MetaModuleIndex as any).Search(DEFAULT_MODULE_INDEX_SEARCH, { limit: 10 });
    expect(Array.isArray(rows)).toBe(true);
    expect(rows.length).toBe(0);
  } finally {
    restoreGroupedRepo();
    restoreBaseSearch();
  }
});

test('meta.MetaModuleIndex Count propagates readGroupCount failures', async () => {
  resetRequestContext();

  const restoreGroupedRepo = mockMetaModuleIndexGroupedRepo([], 0, {
    readGroupCount: async () => {
      throw new Error('readGroupCount boom');
    },
  });

  try {
    await expectAsyncErrorContains(() => (MetaModuleIndex as any).Count(), 'readGroupCount boom');
  } finally {
    restoreGroupedRepo();
  }
});
