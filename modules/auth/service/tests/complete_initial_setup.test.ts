// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import { withPermissionGraphBypass } from '@/core/service/api/authz_bypass';
import { pool, type AppSettingModelCtor } from '@/core/service';
import { createServiceByModel } from '@/core/service/rpc';
import User from '@/auth/service/models/user/user';
import { AuthErrCode } from '@/auth/service/error';
import { SETUP_COMPLETED_AT_KEY } from '@/auth/service/models/user/setup_completed_at_key';
import type MetaModelData from '@/meta/service/models/model_data';

const MetaModelDataService = createServiceByModel<typeof MetaModelData>('meta.MetaModelData');

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

function setReq(patch: Record<string, any>): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.req) jsCtx.req = {};
  Object.assign(jsCtx.req, patch);
}

function setupAllowlistForCompleteInitialSetup(): void {
  setReq({
    depth: 0,
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'auth.User:read',
      'auth.User:write',
      'User:read',
      'User:write',
      'auth.AppSetting:read',
      'auth.AppSetting:write',
      'auth.AppSetting:create',
      'auth.AppSetting:unlink',
      'AppSetting:read',
      'AppSetting:write',
      'AppSetting:create',
      'AppSetting:unlink',
      'meta.MetaModelData:read',
      'MetaModelData:read',
      'auth.Session:read',
      'auth.Session:write',
      'Session:read',
      'Session:write',
    ],
  });
}

async function clearSetupCompletedAt(): Promise<void> {
  await withModelContext(
    {} as any,
    async () => {
      await withPermissionGraphBypass(async () => {
        await pool<AppSettingModelCtor>('auth', 'AppSetting').Set(SETUP_COMPLETED_AT_KEY, null);
      });
    },
    { merge: false }
  );
}

async function expectAuthCode(fn: () => Promise<unknown>, code: string): Promise<void> {
  let caught: any;
  try {
    await fn();
    throw new Error(`expected throw with code ${code}`);
  } catch (err) {
    caught = err;
  }
  expect(String(caught?.code || '')).toBe(code);
}

test('CompleteInitialSetup: updates seed admin and writes setup.completed_at', async () => {
  resetRequestContext();
  setupAllowlistForCompleteInitialSetup();
  await clearSetupCompletedAt();

  const adminId = String(
    await withModelContext(
      {} as any,
      async () => MetaModelDataService.Ref('auth.user_admin'),
      { merge: false }
    ) || ''
  ).trim();
  expect(adminId === '').toBe(false);

  const beforeHash = String(
    await withModelContext(
      {} as any,
      async () => {
        const row = await withPermissionGraphBypass(async () =>
          User.Browse(adminId, ['Id', 'Username', 'PasswordHash'] as any)
        );
        return (row as any)?.PasswordHash;
      },
      { merge: false }
    ) || ''
  );

  // Keep Username=admin so the shared unit DB still resolves the bootstrap user by name.
  const result = await withModelContext(
    {} as any,
    async () =>
      User.CompleteInitialSetup({
        Username: 'admin',
        Password: 'password-123',
      }),
    { merge: false }
  );
  expect(String(result.UserId || '')).toBe(adminId);

  const after = await withModelContext(
    {} as any,
    async () =>
      withPermissionGraphBypass(async () => User.Browse(adminId, ['Id', 'Username', 'PasswordHash'] as any)),
    { merge: false }
  );
  expect(String((after as any)?.Username || '')).toBe('admin');
  const afterHash = String((after as any)?.PasswordHash || '');
  expect(afterHash.length > 0).toBe(true);
  expect(afterHash === beforeHash).toBe(false);

  const completedAt = String(
    (await withModelContext(
      {} as any,
      async () => pool<AppSettingModelCtor>('auth', 'AppSetting').Get(SETUP_COMPLETED_AT_KEY, null),
      { merge: false }
    )) || ''
  ).trim();
  expect(completedAt === '').toBe(false);

  await expectAuthCode(
    () =>
      withModelContext(
        {} as any,
        async () =>
          User.CompleteInitialSetup({
            Username: 'admin',
            Password: 'password-456',
          }),
        { merge: false }
      ),
    AuthErrCode.INITIAL_SETUP_ALREADY_COMPLETE
  );

  await clearSetupCompletedAt();
});

test('CompleteInitialSetup: rejects invalid envelope fields', async () => {
  resetRequestContext();
  setupAllowlistForCompleteInitialSetup();
  await expectAuthCode(
    () =>
      withModelContext(
        {} as any,
        async () => User.CompleteInitialSetup({ Username: '   ', Password: 'x' } as any),
        { merge: false }
      ),
    AuthErrCode.VALIDATION_FAILED
  );
  await expectAuthCode(
    () =>
      withModelContext(
        {} as any,
        async () => User.CompleteInitialSetup({ Username: 'admin', Password: '' } as any),
        { merge: false }
      ),
    AuthErrCode.VALIDATION_FAILED
  );
});
