// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
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
  const xid = (globalThis as any)?.$choysum?.xid?.New;
  const nextId = () => {
    const id = typeof xid === 'function' ? xid() : undefined;
    return (typeof id === 'string' && id.trim()) || String(Date.now());
  };
  JOB_STORE.clear();
  (Job as any).EnqueueJob = async (req: {
    TargetApp: string;
    FullMethod: string;
    Payload?: Record<string, any>;
  }) => {
    const now = new Date();
    const job = {
      Id: `job_${nextId()}`,
      TargetApp: req?.TargetApp,
      FullMethod: req?.FullMethod,
      PayloadJson: req?.Payload ?? {},
      Status: 'queued',
      CreatedAt: now,
      UpdatedAt: now,
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

function installBridge(bridge: Record<string, any>): () => void {
  const root: any = (globalThis as any).$choysum;
  const saved = root.moduleManagement;
  root.moduleManagement = bridge;
  return () => {
    root.moduleManagement = saved;
  };
}

beforeEach(() => {
  resetRequestContext();
  ensureJobMock();
  ensureOperationStore();
  __setMetaPublishTipForTest(undefined);
});

afterEach(() => {
  __setMetaPublishTipForTest(undefined);
});

test('meta.MetaModuleOperation Plan returns blockers from bridge', async () => {
  const restore = installBridge({
    plan: async (params: any) => {
      expect(params.moduleName).toBe('missing.mod');
      expect(params.action).toBe('uninstall');
      return {
        id: 'op-block',
        action: 'uninstall',
        moduleName: 'missing.mod',
        moduleOrder: [],
        affectedApps: [],
        risks: [],
        blockers: [{ code: 'MODULE_NOT_FOUND', level: 'error', message: 'module not found' }],
        baseRevision: 'rev-1',
      };
    },
  });
  try {
    const resp = await MetaModuleOperation.Plan({ ModuleName: 'missing.mod', Action: 'uninstall' });
    expect(resp.Id).toBe('op-block');
    expect(resp.Blockers).toEqual([{ Code: 'MODULE_NOT_FOUND', Level: 'error', Message: 'module not found' }]);
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation Apply queues Execute job and publishes tip', async () => {
  const tips: any[] = [];
  __setMetaPublishTipForTest((event) => {
    tips.push(event);
  });
  const restore = installBridge({
    apply: async (params: any) => {
      expect(params.id).toBe('op-apply');
      return { ok: true };
    },
  });
  try {
    const resp = await MetaModuleOperation.Apply({ Id: 'op-apply' });
    expect(resp.Id).toBe('op-apply');
    expect(JOB_STORE.size).toBe(1);
    const job = [...JOB_STORE.values()][0];
    expect(job.FullMethod).toBe('meta.MetaModuleOperation/Execute');
    expect(job.PayloadJson).toEqual({ req: { Id: 'op-apply', WithDemo: false } });
    expect(OP_STORE.get('op-apply')?.JobId).toBe(job.Id);
    expect(tips[0]?.payload?.operationId).toBe('op-apply');
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation Apply always enqueues Execute with WithDemo false', async () => {
  const restore = installBridge({
    apply: async () => ({ ok: true }),
  });
  try {
    await MetaModuleOperation.Apply({ Id: 'op-demo-flag' });
    const job = [...JOB_STORE.values()][0];
    expect(job.PayloadJson).toEqual({ req: { Id: 'op-demo-flag', WithDemo: false } });
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation Apply rejects bridge failure', async () => {
  const restore = installBridge({
    apply: async () => ({ ok: false, errorMessage: 'baseRevision mismatch' }),
  });
  try {
    let err: unknown;
    try {
      await MetaModuleOperation.Apply({ Id: 'op-bad' });
    } catch (e) {
      err = e;
    }
    expect(err).toBeTruthy();
    expect(String((err as Error)?.message || err)).toContain('baseRevision mismatch');
    expect(OP_STORE.get('op-bad')?.Status).toBe('failed');
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation Apply marks failed when bridge.apply throws', async () => {
  const restore = installBridge({
    apply: async () => {
      throw new Error('apply boom');
    },
  });
  try {
    let err: unknown;
    try {
      await MetaModuleOperation.Apply({ Id: 'op-apply-throw' });
    } catch (e) {
      err = e;
    }
    expect(String((err as Error)?.message || err)).toContain('apply boom');
    expect(OP_STORE.get('op-apply-throw')?.Status).toBe('failed');
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation Apply marks failed when EnqueueJob throws', async () => {
  const restore = installBridge({
    apply: async () => ({ ok: true }),
  });
  const savedEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => {
    throw new Error('queue unavailable');
  };
  try {
    let err: unknown;
    try {
      await MetaModuleOperation.Apply({ Id: 'op-enqueue-fail' });
    } catch (e) {
      err = e;
    }
    expect(String((err as Error)?.message || err)).toContain('queue unavailable');
    expect(OP_STORE.get('op-enqueue-fail')?.Status).toBe('failed');
    expect(String(OP_STORE.get('op-enqueue-fail')?.ErrorMessage || '')).toContain('queue unavailable');
  } finally {
    (Job as any).EnqueueJob = savedEnqueue;
    restore();
  }
});

test('meta.MetaModuleOperation Apply marks failed when EnqueueJob returns empty id', async () => {
  const restore = installBridge({
    apply: async () => ({ ok: true }),
  });
  const savedEnqueue = (Job as any).EnqueueJob;
  (Job as any).EnqueueJob = async () => ({ Id: '' });
  try {
    let err: unknown;
    try {
      await MetaModuleOperation.Apply({ Id: 'op-empty-job' });
    } catch (e) {
      err = e;
    }
    expect(String((err as Error)?.message || err)).toMatch(/empty job id/i);
    expect(OP_STORE.get('op-empty-job')?.Status).toBe('failed');
  } finally {
    (Job as any).EnqueueJob = savedEnqueue;
    restore();
  }
});

test('meta.MetaModuleOperation Execute calls bridge.run and returns operation row', async () => {
  OP_STORE.set('op-run', {
    Id: 'op-run',
    Action: 'install',
    ModuleName: 'core',
    Status: 'succeeded',
    Phase: 'done',
    ReloadTriggered: true,
    ReloadFailed: false,
  });
  let ran = false;
  const restore = installBridge({
    run: async (params: any) => {
      ran = true;
      expect(params.id).toBe('op-run');
      expect(params.withDemo).toBe(false);
      return { ok: true };
    },
  });
  try {
    const resp = await MetaModuleOperation.Execute({ Id: 'op-run' });
    expect(ran).toBe(true);
    expect(resp).toMatchObject({
      Id: 'op-run',
      Action: 'install',
      ModuleName: 'core',
      Status: 'succeeded',
      Phase: 'done',
      ReloadTriggered: true,
      ReloadFailed: false,
    });
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation Execute rethrows when run fails and status stays non-terminal', async () => {
  OP_STORE.set('op-stuck', {
    Id: 'op-stuck',
    Action: 'install',
    ModuleName: 'core',
    Status: 'running',
    Phase: 'apply',
  });
  const restore = installBridge({
    run: async () => {
      throw new Error('bridge not injected');
    },
  });
  try {
    let err: unknown;
    try {
      await MetaModuleOperation.Execute({ Id: 'op-stuck', WithDemo: true });
    } catch (e) {
      err = e;
    }
    expect(String((err as Error)?.message || err)).toContain('bridge not injected');
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation Execute rethrows when run returns ok:false and status stays non-terminal', async () => {
  OP_STORE.set('op-okfalse', {
    Id: 'op-okfalse',
    Action: 'install',
    ModuleName: 'core',
    Status: 'queued',
    Phase: 'apply',
  });
  const restore = installBridge({
    run: async () => ({ ok: false, errorMessage: 'run refused' }),
  });
  try {
    let err: unknown;
    try {
      await MetaModuleOperation.Execute({ Id: 'op-okfalse' });
    } catch (e) {
      err = e;
    }
    expect(String((err as Error)?.message || err)).toContain('run refused');
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation Plan rejects empty id and ok:false', async () => {
  const emptyRestore = installBridge({
    plan: async () => ({ id: '', action: 'install', moduleName: 'core' }),
  });
  try {
    let err: unknown;
    try {
      await MetaModuleOperation.Plan({ ModuleName: 'core', Action: 'install' });
    } catch (e) {
      err = e;
    }
    expect(String((err as Error)?.message || err)).toMatch(/empty id|plan/i);
  } finally {
    emptyRestore();
  }

  const failRestore = installBridge({
    plan: async () => ({ ok: false, errorMessage: 'lease conflict' }),
  });
  try {
    let err: unknown;
    try {
      await MetaModuleOperation.PlanDemo({});
    } catch (e) {
      err = e;
    }
    expect(String((err as Error)?.message || err)).toContain('lease conflict');
  } finally {
    failRestore();
  }
});

test('meta.MetaModuleOperation PlanDemo plans load-demo action', async () => {
  const restore = installBridge({
    plan: async (params: any) => {
      expect(params.action).toBe('load-demo');
      return {
        id: 'op-demo',
        action: 'load-demo',
        moduleName: '',
        moduleOrder: [{ moduleName: 'base', reason: 'demo' }],
        blockers: [{ code: 'LOAD_DEMO_NOT_IMPLEMENTED', level: 'error', message: 'stub' }],
      };
    },
  });
  try {
    const resp = await MetaModuleOperation.PlanDemo({});
    expect(resp.Id).toBe('op-demo');
    expect(resp.Action).toBe('load-demo');
    expect(resp.ModuleOrder[0]?.ModuleName).toBe('base');
    expect(resp.Blockers[0]?.Code).toBe('LOAD_DEMO_NOT_IMPLEMENTED');
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation RequestLoadDemo wraps bridge.loadDemo', async () => {
  const restore = installBridge({
    loadDemo: async (params: any) => {
      expect(String(params.operatorUserId || '').trim()).not.toBe('');
      return { ok: true, id: 'op-load-demo' };
    },
  });
  try {
    const resp = await MetaModuleOperation.RequestLoadDemo({});
    expect(resp).toEqual({ Ok: true, Id: 'op-load-demo' });
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation RequestLoadDemo maps bridge failure', async () => {
  const restore = installBridge({
    loadDemo: async () => ({
      ok: false,
      id: 'op-fail',
      errorDomain: 'MODULE_MANAGEMENT',
      errorCode: 'OP_FAILED',
      errorMessage: 'demo boom',
    }),
  });
  try {
    const resp = await MetaModuleOperation.RequestLoadDemo({});
    expect(resp.Ok).toBe(false);
    expect(resp.Id).toBe('op-fail');
    expect(resp.ErrorDomain).toBe('MODULE_MANAGEMENT');
    expect(resp.ErrorCode).toBe('OP_FAILED');
    expect(resp.ErrorMessage).toBe('demo boom');
  } finally {
    restore();
  }
});

test('meta.MetaModuleOperation has no legacy GetOpStatus symbol', () => {
  expect((MetaModuleOperation as any).GetOpStatus).toBeUndefined();
  expect((MetaModuleOperation as any).RequestInstall).toBeUndefined();
});
