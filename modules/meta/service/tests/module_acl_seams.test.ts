// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import MetaModuleCatalog from '@/meta/service/models/module_catalog';
import MetaModuleOperation from '@/meta/service/models/module_operation';
import { __setMetaPublishTipForTest } from '@/meta/service/tips';
import type JobModel from '@/task/service/models/job';

const Job = createServiceByModel<typeof JobModel>('task.Job');

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const CTX_OVERRIDE_KEY = Symbol.for('choysum.ctx.override');
const CTX_FROZEN_KEY = Symbol.for('choysum.ctx.frozen');
const JOB_STORE = new Map<string, any>();
const OP_STORE = new Map<string, any>();

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

function ensureJobMock(): void {
  JOB_STORE.clear();
  (Job as any).EnqueueJob = async (req: any) => {
    const job = {
      Id: `job_${JOB_STORE.size + 1}`,
      TargetApp: req?.TargetApp,
      FullMethod: req?.FullMethod,
      PayloadJson: req?.Payload ?? {},
    };
    JOB_STORE.set(job.Id, job);
    return job;
  };
}

function ensureOperationStore(): void {
  OP_STORE.clear();
  (MetaModuleOperation as any).UpdateById = async (id: string, vals: Record<string, unknown>) => {
    const row = OP_STORE.get(id) || { Id: id };
    Object.assign(row, vals);
    OP_STORE.set(id, row);
    return row;
  };
  (MetaModuleOperation as any).Search = async (domain: any) => {
    const idEq = Array.isArray(domain) && domain[0] === 'Id' ? String(domain[2] || '') : '';
    if (idEq && OP_STORE.has(idEq)) return [OP_STORE.get(idEq)];
    return [];
  };
}

let restoreBridge: (() => void) | undefined;

function installBridge(bridge: Record<string, any>): () => void {
  const root: any = (globalThis as any).$choysum;
  const saved = root.moduleManagement;
  root.moduleManagement = bridge;
  const restore = () => {
    root.moduleManagement = saved;
    if (restoreBridge === restore) restoreBridge = undefined;
  };
  restoreBridge = restore;
  return restore;
}

beforeEach(() => {
  resetRequestContext();
  ensureJobMock();
  ensureOperationStore();
  __setMetaPublishTipForTest(undefined);
});

afterEach(() => {
  restoreBridge?.();
  __setMetaPublishTipForTest(undefined);
});

test('meta ACL seams: Execute is not an interactive Plan/Apply substitute', () => {
  // Interactive roles get Plan/Apply via UI actions; Execute stays a worker FullMethod.
  expect(typeof MetaModuleOperation.Plan).toBe('function');
  expect(typeof MetaModuleOperation.Apply).toBe('function');
  expect(typeof MetaModuleOperation.Execute).toBe('function');
  expect(typeof MetaModuleOperation.PlanDemo).toBe('function');
});

test('meta ACL seams: sync path uses bridge.plan(action=sync), not MetaModuleOperation.Plan', async () => {
  // Sync ACL is RequestSync-only; MetaModuleOperation.Plan stays install/upgrade/uninstall.
  const planCalls: any[] = [];
  const restore = installBridge({
    plan: async (req: any) => {
      planCalls.push(req);
      if (req?.action !== 'sync') {
        throw new Error(`unexpected plan action: ${String(req?.action)}`);
      }
      return { id: 'op-sync-acl' };
    },
    apply: async () => ({ ok: true }),
  });
  try {
    const resp = await MetaModuleCatalog.RequestSync({ Force: true });
    expect(resp.Id).toBe('op-sync-acl');
    expect(planCalls).toHaveLength(1);
    expect(planCalls[0].action).toBe('sync');
    expect(planCalls[0].origin).toBe('local');
  } finally {
    restore();
  }
});


test('meta E2E seams: forced run failure is injectable via test bridge', async () => {
  OP_STORE.set('op-force-fail', {
    Id: 'op-force-fail',
    Action: 'install',
    ModuleName: 'core',
    Status: 'queued',
    Phase: 'apply',
  });
  const restore = installBridge({
    run: async () => ({ ok: false, errorMessage: 'E2E forced failure', errorCode: 'OP_FAILED' }),
  });
  try {
    let err: unknown;
    try {
      await MetaModuleOperation.Execute({ Id: 'op-force-fail' });
    } catch (e) {
      err = e;
    }
    expect(String((err as Error)?.message || err)).toContain('E2E forced failure');
  } finally {
    restore();
  }
});

test('meta E2E seams: forced lock conflict is injectable via apply mock', async () => {
  OP_STORE.set('op-lock', {
    Id: 'op-lock',
    Action: 'install',
    ModuleName: 'core',
    Status: 'queued',
    Phase: 'apply',
  });
  const restore = installBridge({
    apply: async () => ({ ok: false, errorCode: 'LEASE_CONFLICT', errorMessage: 'lease is busy' }),
  });
  try {
    let err: unknown;
    try {
      await MetaModuleOperation.Apply({ Id: 'op-lock' });
    } catch (e) {
      err = e;
    }
    expect(String((err as Error)?.message || err)).toContain('lease is busy');
    expect(OP_STORE.get('op-lock')?.Status).toBe('failed');
  } finally {
    restore();
  }
});
