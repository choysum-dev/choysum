// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Job from '@/task/service/models/job';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const CTX_OVERRIDE_KEY = Symbol.for('choysum.ctx.override');
const CTX_FROZEN_KEY = Symbol.for('choysum.ctx.frozen');

const ENV_KEY = 'CHOYSUM_TASK_DEFAULT_MAX_ATTEMPTS';

/**
 * Ensures task model tests have a mutable request context scaffold.
 */
function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum;
  if (!root) {
    throw new Error('missing global $choysum; task model tests must run under the QuickJS-first harness');
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
 * Resets the request context to the minimal allowlisted task-model test state.
 */
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
      'Job:create',
      'Job:read',
      'Job:write',
      'Job:delete',
      'task.Schedule:create',
      'task.Schedule:read',
      'task.Schedule:write',
      'task.Schedule:delete',
      'Schedule:create',
      'Schedule:read',
      'Schedule:write',
      'Schedule:delete',
    ],
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = { userId: 'admin' };

  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
  delete (jsCtx as any)[CTX_OVERRIDE_KEY];
  delete (jsCtx as any)[CTX_FROZEN_KEY];
}

/**
 * Temporarily sets a backend env value while running an async test step.
 */
function withBackendEnv<T>(key: string, value: any, run: () => Promise<T>): Promise<T> {
  const meta = import.meta as any;
  if (!meta.env) meta.env = {};
  const prev = meta.env[key];
  meta.env[key] = value;
  const globalAny = globalThis as any;
  const envKey = '__choysumBackendEnv';
  if (!globalAny[envKey]) globalAny[envKey] = {};
  const prevGlobal = globalAny[envKey][key];
  globalAny[envKey][key] = value;
  return run().finally(() => {
    if (typeof prev === 'undefined') {
      delete meta.env[key];
    } else {
      meta.env[key] = prev;
    }
    if (typeof prevGlobal === 'undefined') {
      delete globalAny[envKey][key];
    } else {
      globalAny[envKey][key] = prevGlobal;
    }
  });
}

/**
 * Builds a unique test identifier with a stable prefix.
 */
function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

test('task.Job enqueue/list/cancel basics', async () => {
  resetRequestContext();

  (import.meta as any).env = (import.meta as any).env ?? {};
  (import.meta as any).env[ENV_KEY] = 1;

  const fullMethod = 'auth.User/Login';
  const payload = { email: 'a@b.com' };
  const runAfter = new Date('2026-01-19T00:00:00Z');
  const job = await Job.EnqueueJob('auth', fullMethod, payload, 'admin', 'admin', runAfter, 0, -1);

  expect(job).toBeTruthy();
  expect(job.Status).toBe('queued');
  expect(job.Attempt).toBe(0);
  expect(job.MaxAttempts).toBe(1);
  expect(job.TimeoutMs).toBe(0);

  const listed = await Job.ListJobsPaged({ targetApp: 'auth', fullMethod, limit: 10, offset: 0 });
  expect(listed.total >= 1).toBe(true);
  expect(listed.items.length > 0).toBe(true);

  const reason = uid('cancel');
  await Job.CancelJob(job.Id as any, reason);
  const reloaded = await Job.GetJob(job.Id as any, ['Id', 'Status', 'CancelRequestedAt', 'CancelledAt', 'FinishedAt', 'LastErrorJson'] as any);
  expect(reloaded.Status).toBe('cancelled');
  expect(reloaded.CancelledAt).toBeTruthy();
  expect(reloaded.FinishedAt).toBeTruthy();
  expect(reloaded.CancelRequestedAt).toBeFalsy();
  expect(reloaded.LastErrorJson?.reason).toBe(reason);
});

test('task.Job enqueue uses configured default maxAttempts', async () => {
  resetRequestContext();

  await withBackendEnv(ENV_KEY, 3, async () => {
    const job = await Job.EnqueueJob('auth', 'auth.User/Login', {}, 'admin', 'admin', undefined, 0, 0);
    expect(job.MaxAttempts).toBe(3);
  });
});

test('task.Job enqueue payload sanitize/truncate', async () => {
  resetRequestContext();

  const payload = {
    email: 'a@b.com',
    password: 'secret123',
    profile: {
      access_token: 'abc',
      nested: { refresh_token: 'def' },
    },
    tokens: [{ token: 't1' }, { token: 't2' }],
  };

  const job = await Job.EnqueueJob('auth', 'auth.User/Login', payload, 'admin', 'admin');
  const reloaded = await Job.GetJob(job.Id as any, ['Id', 'PayloadJson'] as any);

  expect(reloaded.PayloadJson?.password).toBe('***');
  expect(reloaded.PayloadJson?.profile?.access_token).toBe('***');
  expect(reloaded.PayloadJson?.profile?.nested?.refresh_token).toBe('***');
  expect(reloaded.PayloadJson?.tokens).toBe('***');

  const bigPayload = { blob: 'x'.repeat(20000) };
  const bigJob = await Job.EnqueueJob('auth', 'auth.User/Login', bigPayload, 'admin', 'admin');
  const bigReloaded = await Job.GetJob(bigJob.Id as any, ['Id', 'PayloadJson'] as any);

  expect(bigReloaded.PayloadJson?._truncated).toBe(true);
  expect(typeof bigReloaded.PayloadJson?._preview).toBe('string');
  const previewLen = (bigReloaded.PayloadJson?._preview || '').length;
  expect(previewLen > 0).toBe(true);
  expect(previewLen <= 16 * 1024).toBe(true);
});
