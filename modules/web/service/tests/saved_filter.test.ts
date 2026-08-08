// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { createServiceByModel } from '@/core/service/rpc';
import SavedFilter from '@/web/service/models/saved_filter';
import { resolveEffectiveModelId } from '@/web/service/models/_resolve_effective_model';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};
  (globalThis as any).$choysum = root;
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'web.SavedFilter:read',
      'web.SavedFilter:write',
      'web.SavedFilter:create',
      'web.SavedFilter:delete',
      'SavedFilter:read',
      'SavedFilter:write',
      'SavedFilter:create',
      'SavedFilter:delete',
      'meta.MetaModel:read',
      'MetaModel:read',
      'web.FieldDefault:read',
      'FieldDefault:read',
      'web.AppSetting:read',
      'AppSetting:read',
    ],
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = { userId: uid('bootstrap') };
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

function setIdentity(userId?: string): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.identity) jsCtx.identity = {};
  if (userId) jsCtx.identity.userId = userId;
  else delete jsCtx.identity.userId;
}

function toErr(err: any): { domain?: string; code?: string } | null {
  if (!err) return null;
  if (err instanceof ChoysumError) return err as any;
  const visited = new Set<any>();
  const queue: any[] = [err];
  while (queue.length) {
    const cur = queue.shift();
    if (!cur || visited.has(cur)) continue;
    visited.add(cur);
    if (cur instanceof ChoysumError) return cur as any;
    if (typeof cur === 'object') {
      if (typeof cur.domain === 'string' || typeof cur.code === 'string') {
        return { domain: cur.domain, code: cur.code };
      }
      if (cur.cause) queue.push(cur.cause);
      if (cur.error) queue.push(cur.error);
    }
  }
  return null;
}

function errorBlob(err: any): string {
  const parts: string[] = [];
  const visited = new Set<any>();
  const queue: any[] = [err];
  while (queue.length) {
    const cur = queue.shift();
    if (!cur || visited.has(cur)) continue;
    visited.add(cur);
    if (typeof cur === 'string') {
      parts.push(cur);
      continue;
    }
    if (typeof cur.code === 'string') parts.push(cur.code);
    if (typeof cur.message === 'string') parts.push(cur.message);
    if (Array.isArray(cur.issues)) for (const issue of cur.issues) queue.push(issue);
    if (cur.cause) queue.push(cur.cause);
    if (cur.error) queue.push(cur.error);
  }
  return parts.join('\n');
}

async function expectCode(fn: () => Promise<any>, code: string, messageHint?: string): Promise<void> {
  let caught: any;
  try {
    await fn();
  } catch (e) {
    caught = e;
  }
  expect(caught, `expected error ${code}, got nothing`).toBeTruthy();
  const oe = toErr(caught);
  if (oe?.code === code) return;
  const blob = errorBlob(caught);
  // `@Constraint` wraps thrown ChoysumError as constraint_execution_failed / validation_failed.
  if (blob.includes(code)) return;
  if (messageHint && blob.includes(messageHint)) return;
  expect(false, `expected error ${code}${messageHint ? ` (hint=${messageHint})` : ''}, got ${blob}`).toBe(true);
}

function metaModel(): any {
  return createServiceByModel('meta.MetaModel');
}

test('SF13: web FieldDefault and AppSetting models exist after declared service', async () => {
  resetRequestContext();
  const fd = await metaModel().Search(
    {
      And: [
        ['Application', '=', 'web'],
        ['Name', '=', 'FieldDefault'],
      ],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  const as = await metaModel().Search(
    {
      And: [
        ['Application', '=', 'web'],
        ['Name', '=', 'AppSetting'],
      ],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  expect(Array.isArray(fd) && fd.length > 0).toBe(true);
  expect(Array.isArray(as) && as.length > 0).toBe(true);
});

test('SavedFilter CRUD + IsDefault exclusivity + visibility', async () => {
  resetRequestContext();
  const actor = uid('sf_actor');
  setIdentity(actor);

  const modelId = await resolveEffectiveModelId('web', 'SavedFilter');
  expect(modelId).toBeTruthy();

  const nameA = uid('fav_a');
  const privateFav = await SavedFilter.Create(
    {
      Name: nameA,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: { And: [['Active', '=', true]] },
      IsDefault: true,
    } as any,
    ['Id', 'UserId', 'ModelId', 'CreateUid', 'IsDefault'] as any
  );
  expect(String((privateFav as any).UserId)).toBe(actor);
  expect(String((privateFav as any).ModelId)).toBe(modelId);
  expect(String((privateFav as any).CreateUid)).toBe(actor);
  expect((privateFav as any).IsDefault).toBe(true);

  const nameB = uid('fav_b');
  const second = await SavedFilter.Create(
    {
      Name: nameB,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  expect((second as any).IsDefault).toBe(true);
  const firstAgain = await SavedFilter.Browse(String((privateFav as any).Id), ['IsDefault'] as any);
  expect((firstAgain as any).IsDefault).toBe(false);

  const sharedName = uid('shared');
  const shared = await SavedFilter.Create(
    {
      Name: sharedName,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      UserId: null,
      IsDefault: false,
    } as any,
    ['Id', 'UserId', 'CreateUid'] as any
  );
  expect((shared as any).UserId == null || (shared as any).UserId === '').toBe(true);

  const other = uid('sf_other');
  setIdentity(other);
  const visible = await SavedFilter.Search(
    {
      And: [
        ['Application', '=', 'web'],
        ['ModelName', '=', 'SavedFilter'],
        {
          Or: [
            ['UserId', '=', other],
            ['UserId', '=', null],
          ],
        },
      ],
    } as any,
    { fields: ['Id', 'Name', 'UserId'] } as any
  );
  const ids = new Set((visible || []).map((r: any) => String(r.Id)));
  expect(ids.has(String((shared as any).Id))).toBe(true);
  expect(ids.has(String((privateFav as any).Id))).toBe(false);

  setIdentity(actor);
  await SavedFilter.DeleteById(String((privateFav as any).Id));
  await SavedFilter.DeleteById(String((second as any).Id));
  await SavedFilter.DeleteById(String((shared as any).Id));
});

test('SavedFilter rejects Create without effective MetaModel', async () => {
  resetRequestContext();
  const actor = uid('sf_noeff');
  setIdentity(actor);
  await expectCode(
    async () =>
      SavedFilter.Create(
        {
          Name: uid('gone'),
          Application: 'no_such_app',
          ModelName: 'NoSuchModel',
          Condition: {},
        } as any,
        ['Id'] as any
      ),
    'FailedPrecondition',
    'No effective model'
  );
});

test('SF11: shared write/delete only for creator (sys.admin needs auth)', async () => {
  resetRequestContext();
  const creator = uid('sf_creator');
  setIdentity(creator);
  const shared = await SavedFilter.Create(
    {
      Name: uid('sf11'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      UserId: null,
    } as any,
    ['Id', 'CreateUid'] as any
  );

  const stranger = uid('sf_stranger');
  setIdentity(stranger);
  await expectCode(
    async () => SavedFilter.UpdateById(String((shared as any).Id), { Name: uid('hijack') } as any, ['Id'] as any),
    'PermissionDenied'
  );
  await expectCode(async () => SavedFilter.DeleteById(String((shared as any).Id)), 'PermissionDenied');

  setIdentity(creator);
  await SavedFilter.UpdateById(String((shared as any).Id), { Name: uid('ok') } as any, ['Id'] as any);
  const deleted = await SavedFilter.DeleteById(String((shared as any).Id));
  expect(deleted).toBe(1);
});
