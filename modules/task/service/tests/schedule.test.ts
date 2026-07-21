// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Job from '@/task/service/models/job';
import Schedule from '@/task/service/models/schedule';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const CTX_OVERRIDE_KEY = Symbol.for('choysum.ctx.override');
const CTX_FROZEN_KEY = Symbol.for('choysum.ctx.frozen');

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

test('task.Schedule create/update/trigger basics', async () => {
  resetRequestContext();

  const now = Date.now();
  const schedule = await Schedule.CreateSchedule('test_schedule', 'auth', 'auth.User/Login', { email: 'a@b.com' }, 'admin', 'admin', '* * * * *', 'UTC');

  expect(Boolean(schedule)).toBe(true);
  expect(Boolean(schedule.NextRunAt)).toBe(true);
  const nextRunAtMs = new Date(schedule.NextRunAt as any).getTime();
  expect(nextRunAtMs === nextRunAtMs).toBe(true);
  expect(nextRunAtMs >= now).toBe(true);
  expect(nextRunAtMs <= now + 2 * 60 * 1000).toBe(true);

  await Schedule.UpdateSchedule(schedule.Id as any, { Active: false } as any);
  const updated = await Schedule.Browse(schedule.Id as any, ['Id', 'Active', 'NextRunAt'] as any);
  expect(updated.Active).toBe(false);
  expect(Boolean(updated.NextRunAt)).toBe(false);

  const triggerSchedule = await Schedule.CreateSchedule(
    'trigger_schedule',
    'auth',
    'auth.User/Login',
    { email: 'b@b.com' },
    'admin',
    'admin',
    '* * * * *',
    'UTC'
  );
  const triggerResp = await Schedule.TriggerSchedule(triggerSchedule.Id as any, { email: 'c@b.com' });
  expect(Boolean(triggerResp.jobId)).toBe(true);

  const job = await Job.GetJob(triggerResp.jobId, ['Id', 'TargetApp', 'FullMethod', 'PayloadJson'] as any);
  expect(job.TargetApp).toBe('auth');
  expect(job.FullMethod).toBe('auth.User/Login');

  const reloaded = await Schedule.Browse(triggerSchedule.Id as any, ['Id', 'LastTriggeredAt', 'LastRunAt'] as any);
  expect(Boolean(reloaded.LastTriggeredAt)).toBe(true);
  expect(Boolean(reloaded.LastRunAt)).toBe(true);
});

test('task.Schedule Timezone FieldsGet exposes dynamic IANA selection', async () => {
  resetRequestContext();
  const meta = await Schedule.FieldsGet(['Timezone'], ['type', 'selectionKind', 'selection']);
  expect(meta.Timezone?.type).toBe('selection');
  expect(meta.Timezone?.selectionKind).toBe('dynamic');
  const selection = meta.Timezone?.selection || [];
  expect(selection.length).toBeGreaterThan(100);
  expect(selection.some((item: { value?: string }) => item.value === 'UTC')).toBe(true);
});

test('task.Schedule validateTimezoneConstraint normalizes and rejects invalid values', () => {
  resetRequestContext();
  const host = Object.assign(Object.create(Schedule.prototype), { Timezone: '  UTC  ' }) as Schedule;
  host.validateTimezoneConstraint();
  expect(host.Timezone).toBe('UTC');

  const invalid = Object.assign(Object.create(Schedule.prototype), { Timezone: 'Not/A_Zone' }) as Schedule;
  expect(() => invalid.validateTimezoneConstraint()).toThrow('invalid timezone');
});

test('task.Schedule UpdateById runs timezone constraint', async () => {
  resetRequestContext();
  const schedule = await Schedule.CreateSchedule(
    'tz_constraint_schedule',
    'auth',
    'auth.User/Login',
    { email: 'tz@b.com' },
    'admin',
    'admin',
    '0 0 * * *',
    'UTC'
  );

  const updated = await (Schedule as any).UpdateById(schedule.Id, { Timezone: '  Asia/Shanghai  ' }, ['Id', 'Timezone']);
  expect(updated.Timezone).toBe('Asia/Shanghai');

  let error: unknown;
  try {
    await (Schedule as any).UpdateById(schedule.Id, { Timezone: 'Not/A_Zone' }, ['Id']);
  } catch (err) {
    error = err;
  }
  expect(Boolean(error)).toBe(true);
});
