// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import User from '@/auth/service/models/user';
import Role from '@/auth/service/models/role';
import UserRole from '@/auth/service/models/user_role';
import RoleRecordRule from '@/auth/service/models/role_record_rule';
import { evaluateRecordRuleCondition } from '@/auth/service/models/_user_record_rule_eval';
import { createServiceByModel } from '@/core/service/rpc';
import type IrApplicationModel from '@/meta/service/models/ir_application';
import type IrFieldModel from '@/meta/service/models/ir_field';
import type IrModelModel from '@/meta/service/models/ir_model';

const IrApplication = createServiceByModel<typeof IrApplicationModel>('meta.IrApplication');
const IrModel = createServiceByModel<typeof IrModelModel>('meta.IrModel');
const IrField = createServiceByModel<typeof IrFieldModel>('meta.IrField');

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
  jsCtx.req = { depth: 0, fieldRuleMode: 'skip' };
  jsCtx.identity = {};
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

function setReq(patch: Record<string, any>): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.req) jsCtx.req = {};
  Object.assign(jsCtx.req, patch);
}

function setupAllowlistForFixtures(): void {
  setReq({
    depth: 0,
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'auth.User:read',
      'auth.User:write',
      'auth.User:create',
      'auth.User:delete',
      'User:read',
      'User:write',
      'User:create',
      'User:delete',
      'auth.Role:read',
      'auth.Role:write',
      'auth.Role:create',
      'auth.Role:delete',
      'Role:read',
      'Role:write',
      'Role:create',
      'Role:delete',
      'auth.UserRole:read',
      'auth.UserRole:write',
      'auth.UserRole:create',
      'auth.UserRole:delete',
      'UserRole:read',
      'UserRole:write',
      'UserRole:create',
      'UserRole:delete',
      'auth.RoleRecordRule:read',
      'auth.RoleRecordRule:write',
      'auth.RoleRecordRule:create',
      'auth.RoleRecordRule:delete',
      'RoleRecordRule:read',
      'RoleRecordRule:write',
      'RoleRecordRule:create',
      'RoleRecordRule:delete',
      'auth.CompanyScopedResource:read',
      'auth.CompanyScopedResource:write',
      'auth.CompanyScopedResource:create',
      'auth.CompanyScopedResource:delete',
      'CompanyScopedResource:read',
      'CompanyScopedResource:write',
      'CompanyScopedResource:create',
      'CompanyScopedResource:delete',
      'meta.IrModel:read',
      'meta.IrApplication:read',
      'meta.IrField:read',
      'IrModel:read',
      'IrApplication:read',
      'IrField:read',
    ],
  });
}

async function createUser(companyId: string): Promise<string> {
  const created = await User.Create(
    {
      Username: uid('u'),
      PasswordHash: 'test',
      FirstName: 'T',
      LastName: 'U',
      CompanyId: companyId,
      CompanyIds: [companyId],
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  return created.Id;
}

async function createRole(): Promise<string> {
  const created = await Role.Create(
    {
      Name: uid('role'),
      Code: uid('ROLE'),
      Description: 'test',
      IsActive: true,
      IsSystem: false,
    } as any,
    ['Id'] as any
  );
  return created.Id;
}

async function grantRoleGlobal(userId: string, roleId: string, companyId: string): Promise<void> {
  await withModelContext(
    { activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any,
    async () => {
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: roleId } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );
    },
    { merge: false }
  );
}

async function resolveModelId(appName: string, modelName: string): Promise<string> {
  const rows = await IrModel.Search(
    {
      And: [
        ['Name', '=', modelName],
        ['Application', '=', appName],
      ],
    } as any,
    { fields: ['Id'], limit: 1 }
  );
  const id = String((rows[0] as any)?.Id || '').trim();
  if (!id) throw new Error(`meta model not found: ${appName}.${modelName}`);
  return id;
}

test('P2-2 eval edges: model_not_found deny', async () => {
  resetRequestContext();
  const env = await evaluateRecordRuleCondition({
    appName: 'auth',
    modelName: uid('MissingModel'),
    hasCompany: false,
    opValue: 'read',
    roleIds: [],
    roleScopesById: {},
  });
  expect(env.kind).toBe('false');
  expect(String(env.reason || '')).toBe('model_not_found');
});

test('P2-2 eval edges: Search truncation fail-closed', async () => {
  resetRequestContext();
  const orig = (RoleRecordRule as any).Search;
  const fakeRows = Array.from({ length: 5001 }, () => ({
    RoleId: null,
    Kind: 'grant',
    Condition: null,
    IrModelId: 'x',
    IrApplicationId: null,
  }));
  (RoleRecordRule as any).Search = async () => fakeRows;
  const prevError = console.error;
  const errors: string[] = [];
  console.error = ((...args: unknown[]) => {
    errors.push(args.map(a => String(a)).join(' '));
  }) as typeof console.error;
  try {
    const env = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'User',
      hasCompany: false,
      opValue: 'read',
      roleIds: ['r1'],
      roleScopesById: {},
    });
    expect(env.kind).toBe('false');
    expect(String(env.reason || '')).toContain('record_rule_truncated_read_deny');
    expect(errors.some(e => e.includes('truncated'))).toBe(true);
  } finally {
    (RoleRecordRule as any).Search = orig;
    console.error = prevError;
  }
});

test('P2-2 eval edges: missing roleScopesById applies deny-lean company gate', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();
  const c1 = { Id: uid('C1') };

  const { roleId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);
      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);
      const modelId = await resolveModelId('auth', 'CompanyScopedResource');
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: modelId,
          Condition: { And: [['Name', '=', 'x']] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      return { roleId };
    },
    { merge: false }
  );

  // roleIds present but roleScopesById omits the role → deny-lean gate (empty company in).
  const env = await evaluateRecordRuleCondition({
    appName: 'auth',
    modelName: 'CompanyScopedResource',
    hasCompany: true,
    opValue: 'read',
    roleIds: [roleId],
    roleScopesById: {},
  });
  expect(env.kind).toBe('expr');
  const json = JSON.stringify((env as any).expr || {});
  expect(json).toContain('CompanyId');
  expect(json).toContain('Name');
  // Deny-lean scope → CompanyId in [].
  expect(json.includes('[]')).toBe(true);
});

test('P2-2 eval edges: true-condition variants and multi grant+restrict compose', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();
  const c1 = { Id: uid('C1') };

  const roleId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);
      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);
      const modelId = await resolveModelId('auth', 'CompanyScopedResource');

      // Two constrained grants (forces orMerge) + one restrict (forces andMerge).
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: modelId,
          Condition: { And: [['Name', '=', 'a']] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: modelId,
          Condition: { Or: [['Name', '=', 'b']] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'restrict',
          IrModelId: modelId,
          Condition: { And: [['Name', '!=', 'z']] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      // True-condition variants that should behave as unconstrained for a separate role grant.
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: modelId,
          Condition: { Or: [] } as any,
          PermWrite: true,
          PermRead: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: modelId,
          Condition: {} as any,
          PermWrite: true,
          PermRead: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: modelId,
          Condition: '' as any,
          PermWrite: true,
          PermRead: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: modelId,
          Condition: [] as any,
          PermWrite: true,
          PermRead: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      return roleId;
    },
    { merge: false }
  );

  const composed = await evaluateRecordRuleCondition({
    appName: 'auth',
    modelName: 'CompanyScopedResource',
    hasCompany: false,
    opValue: 'read',
    roleIds: [roleId],
    roleScopesById: { [roleId]: { global: true, companies: [] } },
  });
  expect(composed.kind).toBe('expr');
  expect(String((composed as any).reason || '')).toBe('grant_or_and_restricts');

  const writeTrue = await evaluateRecordRuleCondition({
    appName: 'auth',
    modelName: 'CompanyScopedResource',
    hasCompany: false,
    opValue: 'write',
    roleIds: [roleId],
    roleScopesById: { [roleId]: { global: true, companies: [] } },
  });
  expect(writeTrue.kind).toBe('true');
  expect(String((writeTrue as any).reason || '')).toBe('grant_unconstrained');
});

test('P2-2 eval edges: mismatched rule scopes skipped; company-gate meta error disables gate', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();
  const c1 = { Id: uid('C1') };

  const roleId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);
      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);
      const modelId = await resolveModelId('auth', 'CompanyScopedResource');
      const userModelId = await resolveModelId('auth', 'User');

      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: modelId,
          Condition: { And: [['Name', '=', 'ok']] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      // Wrong model id on a returned row is skipped by scope filter when Search is mocked.
      void userModelId;
      return roleId;
    },
    { merge: false }
  );

  const modelId = await resolveModelId('auth', 'CompanyScopedResource');
  const origSearch = (RoleRecordRule as any).Search;
  (RoleRecordRule as any).Search = async () => [
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: { And: [['Name', '=', 'ok']] },
      IrModelId: modelId,
      IrApplicationId: null,
    },
    {
      // Mismatched scope: model id for a different model with an app id set → skipped.
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: null,
      IrModelId: 'not-the-model',
      IrApplicationId: 'not-the-app',
    },
    {
      RoleId: { Id: roleId },
      Kind: 'restrict',
      Condition: null, // unconstrained restrict → no-op
      IrModelId: modelId,
      IrApplicationId: null,
    },
  ];

  const origCount = (IrField as any).Count;
  (IrField as any).Count = async () => {
    throw new Error('meta boom');
  };

  try {
    const env = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: false, companies: [c1.Id] } },
    });
    // Company gate disabled due to IrField.Count error; grant expr remains.
    expect(env.kind).toBe('expr');
    expect(String((env as any).reason || '')).toBe('grant_domain');
  } finally {
    (RoleRecordRule as any).Search = origSearch;
    (IrField as any).Count = origCount;
  }
});

test('P2-2 eval edges: company-scoped model without CompanyId field disables gate', async () => {
  resetRequestContext();
  const origCount = (IrField as any).Count;
  (IrField as any).Count = async () => 0;
  try {
    // Use a real company-scoped model so computeCompanyGateMode reaches the field probe.
    setupAllowlistForFixtures();
    const c1 = { Id: uid('C1') };
    const roleId = await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        const uid1 = await createUser(c1.Id);
        setIdentity(uid1);
        const roleId = await createRole();
        await grantRoleGlobal(uid1, roleId, c1.Id);
        const modelId = await resolveModelId('auth', 'CompanyScopedResource');
        await RoleRecordRule.Create(
          {
            RoleId: { Id: roleId } as any,
            Kind: 'grant',
            IrModelId: modelId,
            Condition: { And: [['Name', '=', 'nogate']] } as any,
            PermRead: true,
            PermWrite: false,
            PermCreate: false,
            PermDelete: false,
          } as any,
          ['Id'] as any
        );
        return roleId;
      },
      { merge: false }
    );

    const env = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: false, companies: [c1.Id] } },
    });
    expect(env.kind).toBe('expr');
    // Gate disabled → condition alone, no CompanyId clause.
    expect(JSON.stringify((env as any).expr || {})).not.toContain('CompanyId');
  } finally {
    (IrField as any).Count = origCount;
  }
});

test('P2-2 eval edges: unconstrained grant with one restrict uses grant_and_restrict', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();
  const c1 = { Id: uid('C1') };

  const roleId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);
      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);
      const modelId = await resolveModelId('auth', 'CompanyScopedResource');
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: modelId,
          Condition: null as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'restrict',
          IrModelId: modelId,
          Condition: { And: [['Name', '!=', 'nope']] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      const userModelId = await resolveModelId('auth', 'User');
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: userModelId,
          Condition: { And: [['Id', '!=', '']] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      return roleId;
    },
    { merge: false }
  );

  const andRestrict = await evaluateRecordRuleCondition({
    appName: 'auth',
    modelName: 'CompanyScopedResource',
    hasCompany: false,
    opValue: 'read',
    roleIds: [roleId],
    roleScopesById: { [roleId]: { global: true, companies: [] } },
  });
  expect(andRestrict.kind).toBe('expr');
  expect(String((andRestrict as any).reason || '')).toBe('grant_and_restrict');

  // User is not company-scoped → gate stays off even with hasCompany.
  const userEnv = await evaluateRecordRuleCondition({
    appName: 'auth',
    modelName: 'User',
    hasCompany: true,
    opValue: 'read',
    roleIds: [roleId],
    roleScopesById: { [roleId]: { global: false, companies: [c1.Id] } },
  });
  expect(userEnv.kind).toBe('expr');
  expect(JSON.stringify((userEnv as any).expr || {})).not.toContain('CompanyId');
});

test('P2-2 eval edges: app-scoped grant participates when IrApplication resolves', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();
  const c1 = { Id: uid('C1') };

  const roleId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);
      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);
      const apps = await IrApplication.Search({ And: [['Name', '=', 'auth']] } as any, {
        fields: ['Id'],
        limit: 1,
      } as any);
      const appId = String((apps[0] as any)?.Id || '').trim();
      expect(appId.length > 0).toBe(true);

      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'grant',
          IrModelId: null,
          IrApplicationId: appId,
          Condition: { And: [['Name', '=', 'from_app']] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      return roleId;
    },
    { merge: false }
  );

  const env = await evaluateRecordRuleCondition({
    appName: 'auth',
    modelName: 'CompanyScopedResource',
    hasCompany: false,
    opValue: 'read',
    roleIds: [roleId],
    roleScopesById: { [roleId]: { global: true, companies: [] } },
  });
  expect(env.kind).toBe('expr');
  expect(JSON.stringify((env as any).expr || {})).toContain('from_app');
});
