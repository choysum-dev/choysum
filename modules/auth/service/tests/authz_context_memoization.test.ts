// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import { memoizeInReqState } from '@/core/service/api/context';
import User from '@/auth/service/models/user';
import Role from '@/auth/service/models/role';
import UserRole from '@/auth/service/models/user_role';
import RoleMethodAccess from '@/auth/service/models/role_method_access';
import RoleInheritance from '@/auth/service/models/role_inheritance';
import { evaluateRecordRuleCondition } from '@/auth/service/models/_user_record_rule_eval';
import { evaluateFieldRules } from '@/auth/service/models/_user_field_rule_eval';
import { resolveMethodAccessMeta } from '@/auth/service/models/_user_method_access';
import { createServiceByModel } from '@/core/service/rpc';
import type IrApplicationModel from '@/meta/service/models/ir_application';
import type IrModelModel from '@/meta/service/models/ir_model';
import type IrServiceModel from '@/meta/service/models/ir_service';
const IrApplication = createServiceByModel<typeof IrApplicationModel>('meta.IrApplication');
const IrModel = createServiceByModel<typeof IrModelModel>('meta.IrModel');
const IrService = createServiceByModel<typeof IrServiceModel>('meta.IrService');

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
      // auth
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

      'auth.RoleMethodAccess:read',
      'auth.RoleMethodAccess:write',
      'auth.RoleMethodAccess:create',
      'auth.RoleMethodAccess:delete',
      'RoleMethodAccess:read',
      'RoleMethodAccess:write',
      'RoleMethodAccess:create',
      'RoleMethodAccess:delete',

      // inheritance graph reads
      'auth.RoleInheritance:read',
      'RoleInheritance:read',

      // inheritance graph writes (used by same-request invalidation tests)
      'auth.RoleInheritance:write',
      'auth.RoleInheritance:create',
      'auth.RoleInheritance:delete',
      'RoleInheritance:write',
      'RoleInheritance:create',
      'RoleInheritance:delete',

      // meta
      'meta.IrModel:read',
      'meta.IrService:read',
      'IrModel:read',
      'IrService:read',
    ],
  });
}

function disableAllowlist(): void {
  setReq({ recordRuleMode: '', recordRuleAllow: [] });
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

async function createRole(codePrefix: string): Promise<{ id: string; code: string }> {
  const code = uid(codePrefix);
  const created = await Role.Create(
    {
      Name: uid('role'),
      Code: code,
      Description: 'test',
      IsActive: true,
      IsSystem: false,
    } as any,
    ['Id'] as any
  );
  return { id: created.Id, code };
}

async function resolveModelId(app: string, name: string): Promise<string> {
  const hit = (
    await IrModel.Search(
      {
        And: [
          ['Application', '=', app],
          ['Name', '=', name],
        ],
      } as any,
      { fields: ['Id'], limit: 1 } as any
    )
  )?.[0] as any;
  const id = String(hit?.Id || '').trim();
  if (!id) throw new Error(`meta model not found: ${app}.${name}`);
  return id;
}

async function resolveService(modelId: string, serviceName: string): Promise<{ id: string; name: string }> {
  const rows = await IrService.Search(
    {
      And: [['ModelId', '=', modelId]],
    } as any,
    { fields: ['Id', 'Name'], limit: 5000 } as any
  );

  const target = String(serviceName || '')
    .trim()
    .toLowerCase();
  const hit = (rows || []).find(
    (r: any) =>
      String((r as any).Name || '')
        .trim()
        .toLowerCase() === target
  ) as any;

  const id = String(hit?.Id || '').trim();
  const name = String(hit?.Name || '').trim();
  if (!id || !name) throw new Error(`meta service not found: ${modelId}.${serviceName}`);
  return { id, name };
}

test('P3-1: authz context is request-scoped memoized', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  setupAllowlistForFixtures();

  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id, c2.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_MEMO');

      // grant role scoped to c2
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: c2.Id,
        } as any,
        ['Id'] as any
      );

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');

      await RoleMethodAccess.Create(
        {
          RoleId: { Id: r.id } as any,
          IrServiceId: browse.id,
          IrModelId: null,
          IrApplicationId: null,
          Mode: 'allow',
        } as any,
        ['Id'] as any
      );

      disableAllowlist();

      let userRoleSearchCalls = 0;
      let roleInheritanceSearchCalls = 0;

      const origUserRoleSearch = (UserRole as any).Search;
      const origRoleInhSearch = (RoleInheritance as any).Search;

      (UserRole as any).Search = async (...args: any[]) => {
        userRoleSearchCalls++;
        return await origUserRoleSearch.apply(UserRole, args);
      };
      (RoleInheritance as any).Search = async (...args: any[]) => {
        roleInheritanceSearchCalls++;
        return await origRoleInhSearch.apply(RoleInheritance, args);
      };

      try {
        const fullMethod = `/auth.User/${browse.name}`;

        const a1 = await User.CheckMethodAccess(c2.Id, fullMethod);
        const a2 = await User.CheckMethodAccess(c2.Id, fullMethod);
        const rr = await User.GetRecordRuleCondition('auth.User', 'read');
        const fr = await User.GetFieldRuleSpec('auth.User');

        const jsCtx = ensureRequestContext();
        const state = (jsCtx as any)?.req?.__choysumServiceState as any;
        const injected =
          Boolean(state?.authzContext) ||
          (state && typeof state === 'object' && Object.keys(state).some(k => k === 'authzContext' || k.startsWith('authzContext::')));

        return { a1, a2, rr, fr, userRoleSearchCalls, roleInheritanceSearchCalls, injected };
      } finally {
        (UserRole as any).Search = origUserRoleSearch;
        (RoleInheritance as any).Search = origRoleInhSearch;
      }
    },
    { merge: false }
  );

  expect(out.a1?.allowed).toBe(true);
  expect(out.a2?.allowed).toBe(true);
  expect(out.injected).toBe(true);

  // Within a single request, authz role graph should be computed once.
  expect(out.userRoleSearchCalls).toBe(1);
  expect(out.roleInheritanceSearchCalls).toBe(1);

  // Smoke: record rule condition returns an envelope
  expect(typeof (out.rr as any)?.kind).toBe('string');

  // Smoke: field rule returns arrays
  expect(Array.isArray((out.fr as any)?.denyReadFields)).toBe(true);
  expect(Array.isArray((out.fr as any)?.denyWriteFields)).toBe(true);
});

test('P3-2: meta lookups are request-scoped memoized for record/field/method evaluators', async () => {
  resetRequestContext();

  let appSearchCalls = 0;
  let modelSearchCalls = 0;
  let serviceSearchCalls = 0;

  const origIrApplicationSearch = (IrApplication as any).Search;
  const origIrModelSearch = (IrModel as any).Search;
  const origIrServiceSearch = (IrService as any).Search;

  (IrApplication as any).Search = async (...args: any[]) => {
    appSearchCalls++;
    return await origIrApplicationSearch.apply(IrApplication, args);
  };

  (IrModel as any).Search = async (...args: any[]) => {
    modelSearchCalls++;
    return await origIrModelSearch.apply(IrModel, args);
  };

  (IrService as any).Search = async (...args: any[]) => {
    serviceSearchCalls++;
    return await origIrServiceSearch.apply(IrService, args);
  };

  try {
    const out = await withModelContext(
      { activeCompanyId: uid('C_ACTIVE'), enabledCompanyIds: [] } as any,
      async () => {
        const recordInput = {
          appName: 'auth',
          modelName: 'User',
          hasCompany: false,
          opValue: 'read' as const,
          roleIds: [],
          roleScopesById: {},
        };

        const fieldInput = {
          appName: 'auth',
          modelName: 'User',
          rawModel: 'auth.User',
          roleIds: [],
        };

        const rr1 = await evaluateRecordRuleCondition(recordInput);
        const rr2 = await evaluateRecordRuleCondition(recordInput);

        const fr1 = await evaluateFieldRules(fieldInput);
        const fr2 = await evaluateFieldRules(fieldInput);

        const meta1 = await resolveMethodAccessMeta('auth', 'User', 'browse');
        const meta2 = await resolveMethodAccessMeta('auth', 'User', 'browse');

        return { rr1, rr2, fr1, fr2, meta1, meta2 };
      },
      { merge: false }
    );

    // Empty roleIds + no everyone grant ⇒ RecordRule deny-default (§5.4 / PR-B-2).
    expect(out.rr1.kind).toBe('false');
    expect(out.rr2.kind).toBe('false');
    expect(String(out.rr1.reason || '')).toContain('no_grant');
    expect(String(out.rr2.reason || '')).toContain('no_grant');
    // Empty roleIds ⇒ FieldRule deny-default (§5.5 / PR-C-1): all non-system fields denied.
    expect(out.fr1.reason).toBe('no_roles_deny_by_default');
    expect(out.fr2.reason).toBe('no_roles_deny_by_default');
    expect(Array.isArray(out.fr1.denyReadFields)).toBe(true);
    expect(out.fr1.denyReadFields.length > 0).toBe(true);
    expect(out.fr1.denyWriteFields).toEqual(out.fr1.denyReadFields);
    expect(out.meta1).toBeTruthy();
    expect(out.meta2).toBeTruthy();

    expect(appSearchCalls).toBe(3);
    expect(modelSearchCalls).toBe(3);
    expect(serviceSearchCalls).toBe(1);
  } finally {
    (IrApplication as any).Search = origIrApplicationSearch;
    (IrModel as any).Search = origIrModelSearch;
    (IrService as any).Search = origIrServiceSearch;
  }
});

test('P3-1: same request RoleInheritance write invalidates authz context', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();

  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const parent = await createRole('ROLE_PARENT');
      const child = await createRole('ROLE_CHILD');

      // Grant parent role only; child role will be implied later via RoleInheritance.
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: parent.id } as any,
          CompanyId: c1.Id,
        } as any,
        ['Id'] as any
      );

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');

      // Allow browse on the implied child role.
      await RoleMethodAccess.Create(
        {
          RoleId: { Id: child.id } as any,
          IrServiceId: browse.id,
          IrModelId: null,
          IrApplicationId: null,
          Mode: 'allow',
        } as any,
        ['Id'] as any
      );

      disableAllowlist();

      const fullMethod = `/auth.User/${browse.name}`;

      // Warm request-scoped authz context (no inheritance yet => should be denied).
      const a1 = await User.CheckMethodAccess(c1.Id, fullMethod);

      // Mutate role graph mid-request: parent implies child.
      setupAllowlistForFixtures();
      await RoleInheritance.Create(
        {
          ParentRoleId: { Id: parent.id } as any,
          ChildRoleId: { Id: child.id } as any,
        } as any,
        ['Id'] as any
      );
      disableAllowlist();

      // Must reflect latest graph (requires request cache invalidation).
      const a2 = await User.CheckMethodAccess(c1.Id, fullMethod);
      return { a1, a2 };
    },
    { merge: false }
  );

  expect(out.a1?.allowed).toBe(false);
  expect(out.a2?.allowed).toBe(true);
});

test('P3-1: same request UserRole.CreateMany invalidates authz context', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  setupAllowlistForFixtures();

  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id, c2.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_CREATE_MANY_INV');

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');

      await RoleMethodAccess.Create(
        {
          RoleId: { Id: r.id } as any,
          IrServiceId: browse.id,
          IrModelId: null,
          IrApplicationId: null,
          Mode: 'allow',
        } as any,
        ['Id'] as any
      );

      disableAllowlist();

      const before = await User.CheckMethodAccess(c2.Id, `/auth.User/${browse.name}`);

      // Mid-request fixture write: re-enable allowlist (deny-default blocks UserRole create).
      setupAllowlistForFixtures();
      await UserRole.CreateMany(
        [
          {
            UserId: { Id: userId } as any,
            RoleId: { Id: r.id } as any,
            CompanyId: c2.Id,
          },
        ] as any,
        ['Id'] as any
      );
      disableAllowlist();

      const after = await User.CheckMethodAccess(c2.Id, `/auth.User/${browse.name}`);
      return { before, after };
    },
    { merge: false }
  );

  expect(out.before?.allowed).toBe(false);
  expect(out.after?.allowed).toBe(true);
});

test('P3-2: memoizeInReqState caches undefined values to avoid repeated factory calls', async () => {
  let callCount = 0;
  const state: Record<string, unknown> = {};

  const factory = async (): Promise<string | undefined> => {
    callCount++;
    return undefined;
  };

  const v1 = await memoizeInReqState(state, 'testUndefinedKey', factory);
  expect(v1).toBeUndefined();
  expect(callCount).toBe(1);

  const v2 = await memoizeInReqState(state, 'testUndefinedKey', factory);
  expect(v2).toBeUndefined();
  // factory must not be called again because undefined is a cacheable resolved value.
  expect(callCount).toBe(1);
});
