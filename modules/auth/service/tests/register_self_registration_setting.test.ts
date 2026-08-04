// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import type { AppSettingModelCtor } from '@/core/service';
import User from '@/auth/service/models/user';
import { AuthErrCode } from '@/auth/service/error';
import { ALLOW_SELF_REGISTRATION_KEY } from '@/auth/service/models/_user_lifecycle_auth';
import { withPermissionGraphBypass } from '@/auth/service/models/_user_authz_shared';

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
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = {};

  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function setupAllowlistForRegister(): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.req) jsCtx.req = {};
  Object.assign(jsCtx.req, {
    depth: 0,
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'auth.User:read',
      'auth.User:write',
      'auth.User:create',
      'User:read',
      'User:write',
      'User:create',

      'auth.Role:read',
      'Role:read',

      'auth.UserRole:read',
      'auth.UserRole:create',
      'UserRole:read',
      'UserRole:create',

      'auth.RoleMethodAccess:read',
      'auth.RoleMethodAccess:create',
      'RoleMethodAccess:read',
      'RoleMethodAccess:create',

      'meta.MetaModel:read',
      'meta.MetaService:read',
      'MetaModel:read',
      'MetaService:read',

      'base.Company:read',
      'Company:read',

      'auth.AppSetting:read',
      'auth.AppSetting:write',
      'auth.AppSetting:create',
      'auth.AppSetting:unlink',
      'AppSetting:read',
      'AppSetting:write',
      'AppSetting:create',
      'AppSetting:unlink',
    ],
  });
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const value = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${value}`;
}

async function setSelfRegistrationFlag(value: string | null): Promise<void> {
  await withModelContext(
    {} as any,
    async () => {
      await withPermissionGraphBypass(async () => {
        await User.pool<AppSettingModelCtor>('AppSetting').Set(ALLOW_SELF_REGISTRATION_KEY, value);
      });
    },
    { merge: false }
  );
}

async function registerOnce(prefix: string): Promise<string> {
  const username = uid(prefix);
  return await withModelContext(
    {} as any,
    async () => {
      return await User.Register(
        {
          Username: username,
          Email: `${username}@example.com`,
          FirstName: 'Self',
          LastName: 'Reg',
        } as any,
        'password-123'
      );
    },
    { merge: false }
  );
}

test('Register: default allow_self_registration stays open', async () => {
  resetRequestContext();
  setupAllowlistForRegister();
  await setSelfRegistrationFlag(null);

  const userId = await registerOnce('as3_open');
  expect(String(userId || '').trim() === '').toBe(false);
});

test('Register: Set(allow_self_registration, 0) rejects signup', async () => {
  resetRequestContext();
  setupAllowlistForRegister();
  await setSelfRegistrationFlag('0');

  let caught: any;
  try {
    await registerOnce('as3_closed');
  } catch (err) {
    caught = err;
  }
  expect(caught).toBeTruthy();
  expect(String(caught?.code || caught?.Code || '')).toContain(AuthErrCode.REGISTRATION_DISABLED);
});

test('Register: Set(1) and hard-delete null reopen; same key reusable', async () => {
  resetRequestContext();
  setupAllowlistForRegister();

  await setSelfRegistrationFlag('0');
  await setSelfRegistrationFlag('1');
  const idOpen = await registerOnce('as3_reopen');
  expect(String(idOpen || '').trim() === '').toBe(false);

  // Set(null) hard-deletes so the unique key can be written again.
  await setSelfRegistrationFlag(null);
  await setSelfRegistrationFlag('0');
  await setSelfRegistrationFlag('1');
  const idAgain = await registerOnce('as3_reuse');
  expect(String(idAgain || '').trim() === '').toBe(false);
});
