// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import { withPermissionGraphBypass } from '@/core/service/api/authz_bypass';
import { pool, type AppSettingModelCtor } from '@/core/service';
import { createServiceByModel, getServiceFactory } from '@/core/service/rpc';
import User from '@/auth/service/models/user/user';
import { AuthErrCode } from '@/auth/service/error';
import { SETUP_COMPLETED_AT_KEY } from '@/auth/service/models/user/setup_completed_at_key';
import type MetaModelData from '@/meta/service/models/model_data';
import type CompanyModel from '@/base/service/models/company';
import type CurrencyModel from '@/base/service/models/currency';

const MetaModelDataService = createServiceByModel<typeof MetaModelData>('meta.MetaModelData');
const Company = createServiceByModel<typeof CompanyModel>('base.Company');
const Currency = createServiceByModel<typeof CurrencyModel>('base.Currency');

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
      'base.Company:read',
      'base.Company:write',
      'Company:read',
      'Company:write',
      'base.Currency:read',
      'base.Currency:write',
      'Currency:read',
      'Currency:write',
      'base.Country:read',
      'Country:read',
      'base.Language:read',
      'Language:read',
    ],
  });
}

function setupReq(patch?: { Admin?: Record<string, unknown>; Organization?: Record<string, unknown> }): any {
  return {
    Admin: { Username: 'admin', Password: 'password-123', LanguageCode: 'zh_CN', ...(patch?.Admin || {}) },
    Organization: {
      Name: 'Main Company',
      CurrencyCode: 'CNY',
      Timezone: 'Asia/Shanghai',
      ...(patch?.Organization || {}),
    },
  };
}

async function browseBypass(model: any, id: string, fields: string[]): Promise<any> {
  return await withModelContext(
    {} as any,
    async () => withPermissionGraphBypass(async () => model.Browse(id, fields as any)),
    { merge: false }
  );
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
  // Avatar soft dial must not be required for initial setup (auth does not depend on document).
  expect(getServiceFactory('document.AttachmentBinding')).toBeUndefined();

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

  const companyId = String(
    (await withModelContext({} as any, async () => MetaModelDataService.Ref('base.company_main'), { merge: false })) || ''
  ).trim();
  const [usdBefore] = (await withModelContext(
    {} as any,
    async () => withPermissionGraphBypass(async () => Currency.Search(['Code', '=', 'USD'] as any, { fields: ['Id', 'IsActive'] } as any)),
    { merge: false }
  )) as any[];
  expect(usdBefore.IsActive).toBe(false);

  // Keep Username=admin so the shared unit DB still resolves the bootstrap user by name.
  const result = await withModelContext(
    {} as any,
    async () =>
      User.CompleteInitialSetup(
        setupReq({
          Admin: { Email: 'boss@acme.io', LanguageCode: 'en_US' },
          Organization: { Name: 'Acme Ltd', CountryCode: 'US', CurrencyCode: 'USD', Timezone: 'America/New_York' },
        })
      ),
    { merge: false }
  );
  expect(String(result.UserId || '')).toBe(adminId);

  const usdAfter = await browseBypass(Currency, String(usdBefore.Id), ['Id', 'IsActive']);
  expect(usdAfter.IsActive).toBe(true);
  const company = await browseBypass(Company, companyId, ['Id', 'Name', 'Timezone', 'CurrencyId']);
  expect(String(company.Name)).toBe('Acme Ltd');
  expect(String(company.Timezone)).toBe('America/New_York');
  expect(String(company.CurrencyId?.Id ?? company.CurrencyId)).toBe(String(usdBefore.Id));
  const adminRow = await browseBypass(User, adminId, ['Id', 'Email', 'LanguageId']);
  expect(String(adminRow.Email)).toBe('boss@acme.io');
  expect(Boolean(adminRow.LanguageId)).toBe(true);

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
          User.CompleteInitialSetup(setupReq({ Admin: { Password: 'password-456' } })),
        { merge: false }
      ),
    AuthErrCode.INITIAL_SETUP_ALREADY_COMPLETE
  );

  await clearSetupCompletedAt();
});

test('GetInitialSetupState: reports Complete from setup.completed_at', async () => {
  resetRequestContext();
  setReq({
    depth: 0,
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'auth.AppSetting:read',
      'AppSetting:read',
    ],
  });
  await clearSetupCompletedAt();

  const pending = await withModelContext(
    {} as any,
    async () => User.GetInitialSetupState({}),
    { merge: false }
  );
  expect(pending.Complete).toBe(false);

  await withModelContext(
    {} as any,
    async () => {
      await withPermissionGraphBypass(async () => {
        await pool<AppSettingModelCtor>('auth', 'AppSetting').Set(
          SETUP_COMPLETED_AT_KEY,
          '2020-01-01T00:00:00Z'
        );
      });
    },
    { merge: false }
  );

  const ready = await withModelContext(
    {} as any,
    async () => User.GetInitialSetupState({}),
    { merge: false }
  );
  expect(ready.Complete).toBe(true);

  await clearSetupCompletedAt();
});

test('CompleteInitialSetup: rejects invalid envelope fields', async () => {
  resetRequestContext();
  setupAllowlistForCompleteInitialSetup();
  await expectAuthCode(
    () =>
      withModelContext(
        {} as any,
        async () => User.CompleteInitialSetup(setupReq({ Admin: { Username: '   ', Password: 'x' } })),
        { merge: false }
      ),
    AuthErrCode.VALIDATION_FAILED
  );
  await expectAuthCode(
    () =>
      withModelContext(
        {} as any,
        async () => User.CompleteInitialSetup(setupReq({ Admin: { Password: '' } })),
        { merge: false }
      ),
    AuthErrCode.VALIDATION_FAILED
  );
});

test('CompleteInitialSetup: claims over a blank setup.completed_at row', async () => {
  resetRequestContext();
  setupAllowlistForCompleteInitialSetup();
  await clearSetupCompletedAt();

  await withModelContext(
    {} as any,
    async () => {
      await withPermissionGraphBypass(async () => {
        // Pending blank marker occupies the unique Key; claim must reclaim it.
        await pool<AppSettingModelCtor>('auth', 'AppSetting').Set(SETUP_COMPLETED_AT_KEY, '   ');
      });
    },
    { merge: false }
  );

  const adminId = String(
    await withModelContext(
      {} as any,
      async () => MetaModelDataService.Ref('auth.user_admin'),
      { merge: false }
    ) || ''
  ).trim();
  expect(adminId === '').toBe(false);

  const result = await withModelContext(
    {} as any,
    async () =>
      // Restores bootstrap-like org values and clears the email for later tests.
      User.CompleteInitialSetup(setupReq({ Admin: { Password: 'password-blank-claim', Email: '' } })),
    { merge: false }
  );
  expect(String(result.UserId || '')).toBe(adminId);

  const completedAt = String(
    (await withModelContext(
      {} as any,
      async () => pool<AppSettingModelCtor>('auth', 'AppSetting').Get(SETUP_COMPLETED_AT_KEY, null),
      { merge: false }
    )) || ''
  ).trim();
  expect(completedAt === '').toBe(false);
  const adminRow = await browseBypass(User, adminId, ['Id', 'Email']);
  expect(adminRow.Email == null || adminRow.Email === '').toBe(true);

  await clearSetupCompletedAt();
});
