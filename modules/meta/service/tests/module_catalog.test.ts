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
const OP_ROWS: any[] = [];
let lastOpSearchDomain: unknown;

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
  const xid = (globalThis as any)?.$choysum?.xid?.New;
  const nextId = () => {
    const id = typeof xid === 'function' ? xid() : undefined;
    return (typeof id === 'string' && id.trim()) || String(Date.now());
  };
  JOB_STORE.clear();
  (Job as any).EnqueueJob = async (req: any) => {
    const job = {
      Id: `job_${nextId()}`,
      TargetApp: req?.TargetApp,
      FullMethod: req?.FullMethod,
      PayloadJson: req?.Payload ?? {},
    };
    JOB_STORE.set(job.Id, job);
    return job;
  };
}

beforeEach(() => {
  resetRequestContext();
  ensureJobMock();
  OP_ROWS.length = 0;
  lastOpSearchDomain = undefined;
  __setMetaPublishTipForTest(() => undefined);
  (MetaModuleOperation as any).Search = async (domain: unknown) => {
    lastOpSearchDomain = domain;
    return OP_ROWS.slice();
  };
  (MetaModuleOperation as any).UpdateById = async (id: string, vals: Record<string, unknown>) => {
    const row = OP_ROWS.find((r) => r.Id === id) || { Id: id };
    Object.assign(row, vals);
    if (!OP_ROWS.includes(row)) OP_ROWS.push(row);
    return row;
  };
  (MetaModuleCatalog as any).Search = async () => [];
});

afterEach(() => {
  __setMetaPublishTipForTest(undefined);
});

test('meta.MetaModuleCatalog RequestSync reuses active sync operation', async () => {
  OP_ROWS.push({ Id: 'op-sync-active', Action: 'sync', Status: 'queued' });
  const root: any = (globalThis as any).$choysum;
  const saved = root.moduleManagement;
  root.moduleManagement = {
    plan: async () => {
      throw new Error('should not plan when active sync exists');
    },
  };
  try {
    const resp = await MetaModuleCatalog.RequestSync({ Force: true });
    expect(resp.Id).toBe('op-sync-active');
    expect(JOB_STORE.size).toBe(0);
    expect(lastOpSearchDomain).toEqual({
      And: [
        ['Action', '=', 'sync'],
        ['Status', 'in', ['draft', 'queued', 'running']],
      ],
    });
  } finally {
    root.moduleManagement = saved;
  }
});

test('meta.MetaModuleCatalog RequestSync plans apply and enqueues Execute', async () => {
  const root: any = (globalThis as any).$choysum;
  const saved = root.moduleManagement;
  const calls: string[] = [];
  root.moduleManagement = {
    plan: async (params: any) => {
      calls.push('plan');
      expect(params.action).toBe('sync');
      return { id: 'op-sync-new' };
    },
    apply: async (params: any) => {
      calls.push('apply');
      expect(params.id).toBe('op-sync-new');
      return { ok: true };
    },
  };
  try {
    const resp = await MetaModuleCatalog.RequestSync({ Force: true });
    expect(resp.Id).toBe('op-sync-new');
    expect(calls).toEqual(['plan', 'apply']);
    expect(JOB_STORE.size).toBe(1);
    const job = [...JOB_STORE.values()][0];
    expect(job.FullMethod).toBe('meta.MetaModuleOperation/Execute');
    expect(job.PayloadJson).toEqual({ req: { Id: 'op-sync-new' } });
  } finally {
    root.moduleManagement = saved;
  }
});

test('meta.MetaModuleCatalog RequestSync IfStale skips when catalog is fresh', async () => {
  (MetaModuleCatalog as any).Search = async () => [{ LastSyncAt: new Date() }];
  const root: any = (globalThis as any).$choysum;
  const saved = root.moduleManagement;
  root.moduleManagement = {
    plan: async () => {
      throw new Error('should not plan when fresh');
    },
  };
  try {
    const resp = await MetaModuleCatalog.RequestSync({ Force: false, IfStale: true });
    expect(resp.Id).toBe('');
  } finally {
    root.moduleManagement = saved;
  }
});
