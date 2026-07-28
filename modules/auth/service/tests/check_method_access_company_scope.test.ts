// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import User from '@/auth/service/models/user';
import Role from '@/auth/service/models/role';
import UserRole from '@/auth/service/models/user_role';
import RoleMethodAccess from '@/auth/service/models/role_method_access';
import RoleUiResource from '@/auth/service/models/role_ui_resource';
import { evaluateUiDerivedMethodDecision } from '@/auth/service/models/_user_method_access';
import IrUiResource from '@/meta/service/models/ir_ui_resource';
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

      'auth.RoleUiResource:read',
      'auth.RoleUiResource:write',
      'auth.RoleUiResource:create',
      'auth.RoleUiResource:delete',
      'RoleUiResource:read',
      'RoleUiResource:write',
      'RoleUiResource:create',
      'RoleUiResource:delete',

      'meta.IrUiResource:read',
      'meta.IrUiResource:write',
      'meta.IrUiResource:create',
      'meta.IrUiResource:delete',
      'IrUiResource:read',
      'IrUiResource:write',
      'IrUiResource:create',
      'IrUiResource:delete',

      // meta
      'meta.IrApplication:read',
      'IrApplication:read',
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
  // Meta IR service names are not guaranteed to be case-stable (and can differ across build paths).
  // Follow the established pattern from permission_state.test.ts: fetch by ModelId then match by name.
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

async function resolveApplicationId(appName: string): Promise<string> {
  const hit = (await IrApplication.Search({ And: [['Name', '=', appName]] } as any, { fields: ['Id'], limit: 1 } as any))?.[0] as any;
  const id = String(hit?.Id || '').trim();
  if (!id) throw new Error(`meta application not found: ${appName}`);
  return id;
}

async function createUiResource(input: {
  resourceId: string;
  type: 'ROUTE' | 'MENU' | 'ACTION';
  requires?: string[];
  IrApplicationId?: string | null;
}): Promise<string> {
  const created = await IrUiResource.Create(
    {
      Name: input.resourceId,
      Type: input.type,
      Requires: input.requires ?? [],
      IrApplicationId: input.IrApplicationId ?? null,
      Module: 'auth',
    } as any,
    ['Id'] as any
  );
  const id = String((created as any)?.Id || '').trim();
  if (!id) throw new Error(`meta ui resource not found: ${input.resourceId}`);
  return id;
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

test('P3-2: CheckMethodAccess respects company-scoped roles', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id, c2.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_ACL');

      // grant role scoped to c2 only
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
      const fullMethod = `/auth.User/${browse.name}`;
      const allowC1 = await User.CheckMethodAccess(c1.Id, fullMethod);
      const allowC2 = await User.CheckMethodAccess(c2.Id, fullMethod);
      const allowEmpty = await User.CheckMethodAccess('', fullMethod);

      return { allowC1, allowC2, allowEmpty };
    },
    { merge: false }
  );

  // role is scoped to c2 => only c2 should pass
  expect(out.allowC1).toBe(false);
  expect(out.allowC2).toBe(true);
  // empty companyId must fail-closed
  expect(out.allowEmpty).toBe(false);
});

test('P3-2: CheckMethodAccess global role applies to any company', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id, c2.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_ACL_G');

      // global role
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
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
      const fullMethod = `/auth.User/${browse.name}`;
      const allowC1 = await User.CheckMethodAccess(c1.Id, fullMethod);
      const allowC2 = await User.CheckMethodAccess(c2.Id, fullMethod);
      return { allowC1, allowC2 };
    },
    { merge: false }
  );

  expect(out.allowC1).toBe(true);
  expect(out.allowC2).toBe(true);
});

test('P3-2: CheckMethodAccess application wildcard allow applies to any method within app', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_ACL_APP');

      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      const authAppId = await resolveApplicationId('auth');

      await RoleMethodAccess.Create(
        {
          RoleId: { Id: r.id } as any,
          IrServiceId: null,
          IrModelId: null,
          IrApplicationId: authAppId,
          Mode: 'allow',
        } as any,
        ['Id'] as any
      );

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');

      disableAllowlist();
      const fullMethod = `/auth.User/${browse.name}`;
      const ok = await User.CheckMethodAccess(c1.Id, fullMethod);
      return { ok };
    },
    { merge: false }
  );

  expect(out.ok).toBe(true);
});

test('P3-2: CheckMethodAccess deny wins across scopes (app allow + service deny)', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_ACL_DW');
      await UserRole.Create({ UserId: { Id: userId } as any, RoleId: { Id: r.id } as any, CompanyId: null as any } as any, ['Id'] as any);

      const authAppId = await resolveApplicationId('auth');
      await RoleMethodAccess.Create(
        { RoleId: { Id: r.id } as any, IrServiceId: null, IrModelId: null, IrApplicationId: authAppId, Mode: 'allow' } as any,
        ['Id'] as any
      );

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');
      const logout = await resolveService(userModelId, 'logout');

      // more specific deny overrides broader allow
      await RoleMethodAccess.Create(
        { RoleId: { Id: r.id } as any, IrServiceId: logout.id, IrModelId: null, IrApplicationId: null, Mode: 'deny' } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const allowBrowse = await User.CheckMethodAccess(c1.Id, `/auth.User/${browse.name}`);
      const allowLogout = await User.CheckMethodAccess(c1.Id, `/auth.User/${logout.name}`);
      return { allowBrowse, allowLogout };
    },
    { merge: false }
  );

  expect(out.allowBrowse).toBe(true);
  expect(out.allowLogout).toBe(false);
});

test('P3-2: CheckMethodAccess global wildcard allow grants access across apps', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_ACL_GLOBAL');
      await UserRole.Create({ UserId: { Id: userId } as any, RoleId: { Id: r.id } as any, CompanyId: null as any } as any, ['Id'] as any);

      await RoleMethodAccess.Create(
        { RoleId: { Id: r.id } as any, IrServiceId: null, IrModelId: null, IrApplicationId: null, Mode: 'allow' } as any,
        ['Id'] as any
      );

      const modelId = await resolveModelId('meta', 'IrModel');
      const browse = await resolveService(modelId, 'browse');

      disableAllowlist();
      const ok = await User.CheckMethodAccess(c1.Id, `/meta.IrModel/${browse.name}`);
      return { ok };
    },
    { merge: false }
  );

  expect(out.ok).toBe(true);
});

test('P3-2: CheckMethodAccess ui deny overrides ui allow on same runtime method', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_DENY');
      await UserRole.Create({ UserId: { Id: userId } as any, RoleId: { Id: r.id } as any, CompanyId: null as any } as any, ['Id'] as any);

      const authAppId = await resolveApplicationId('auth');
      const deniedResourceId = await createUiResource({
        resourceId: uid('auth.route.ui_deny'),
        type: 'ROUTE',
        requires: ['rpc:/auth.User/Browse'],
        IrApplicationId: authAppId,
      });

      await RoleUiResource.Create({ RoleId: { Id: r.id } as any, IrApplicationId: authAppId, IrUiResourceId: null, Mode: 'allow' } as any, ['Id'] as any);
      await RoleUiResource.Create({ RoleId: { Id: r.id } as any, IrApplicationId: null, IrUiResourceId: deniedResourceId, Mode: 'deny' } as any, ['Id'] as any);

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');

      disableAllowlist();
      const ok = await User.CheckMethodAccess(c1.Id, `/auth.User/${browse.name}`);
      return { ok };
    },
    { merge: false }
  );

  expect(out.ok).toBe(false);
});

test('P3-2: evaluateUiDerivedMethodDecision marks denied when allow and deny both match', async () => {
  resetRequestContext();
  const originalRoleUiSearch = (RoleUiResource as any).Search;
  const originalIrUiSearch = (IrUiResource as any).Search;

  (RoleUiResource as any).Search = async () => [
    { IrApplicationId: 'APP-1', IrUiResourceId: null, Mode: 'allow' },
    { IrApplicationId: null, IrUiResourceId: 'RES-DENY', Mode: 'deny' },
  ];

  (IrUiResource as any).Search = async () => [
    {
      Id: 'RES-DENY',
      Name: 'res-deny',
      IrApplicationId: 'APP-1',
      Requires: ['rpc:/auth.User/browse'],
    },
  ];

  try {
    const out = await evaluateUiDerivedMethodDecision(['ROLE-1'], 'auth.User', 'browse');
    expect(out.allowed).toBe(false);
    expect(out.denied).toBe(true);
  } finally {
    (RoleUiResource as any).Search = originalRoleUiSearch;
    (IrUiResource as any).Search = originalIrUiSearch;
  }
});

test('P3-2: CheckMethodAccess manual allow remains authoritative over ui deny', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_DENY_M');
      await UserRole.Create({ UserId: { Id: userId } as any, RoleId: { Id: r.id } as any, CompanyId: null as any } as any, ['Id'] as any);

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');
      const authAppId = await resolveApplicationId('auth');
      const deniedResourceId = await createUiResource({
        resourceId: uid('auth.route.ui_deny_manual'),
        type: 'ROUTE',
        requires: ['rpc:/auth.User/*'],
        IrApplicationId: authAppId,
      });

      await RoleMethodAccess.Create(
        { RoleId: { Id: r.id } as any, IrServiceId: browse.id, IrModelId: null, IrApplicationId: null, Mode: 'allow' } as any,
        ['Id'] as any
      );
      await RoleUiResource.Create({ RoleId: { Id: r.id } as any, IrApplicationId: null, IrUiResourceId: deniedResourceId, Mode: 'deny' } as any, ['Id'] as any);

      disableAllowlist();
      const ok = await User.CheckMethodAccess(c1.Id, `/auth.User/${browse.name}`);
      return { ok };
    },
    { merge: false }
  );

  expect(out.ok).toBe(true);
});

test('RoleMethodAccess db check: deleted rows bypass scope xor', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole('ROLE_METHOD_DELETED_CHECK');
    const userModelId = await resolveModelId('auth', 'User');
    const browse = await resolveService(userModelId, 'browse');

    const row = await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        IrServiceId: browse.id,
        IrModelId: null,
        IrApplicationId: null,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );

    const id = String((row as any)?.Id || '').trim();
    expect(id.length > 0).toBe(true);

    await RoleMethodAccess.DeleteById(id);

    const repo = RoleMethodAccess.getRepository().withDeleted();
    const updated = await repo.update(
      {
        IrServiceId: browse.id,
        IrModelId: userModelId,
        IrApplicationId: null,
      } as any,
      ['Id', '=', id] as any
    );

    expect((updated || []).length).toBe(1);

    const rows = await RoleMethodAccess.Search(
      ['Id', '=', id] as any,
      {
        fields: ['Id', 'DeletedAt', 'IrServiceId', 'IrModelId', 'IrApplicationId'] as any,
        withDeleted: true,
      } as any
    );

    expect(rows.length).toBe(1);
    expect(String((rows[0] as any)?.IrServiceId || '').trim()).toBe(browse.id);
    expect(String((rows[0] as any)?.IrModelId || '').trim()).toBe(userModelId);
    expect(String((rows[0] as any)?.IrApplicationId || '').trim()).toBe('');
    expect((rows[0] as any)?.DeletedAt != null).toBe(true);
  });
});

test('RoleMethodAccess: permission-only update must not rewrite scoped fields to global', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole('ROLE_METHOD_SCOPE_SAFE');
    const userModelId = await resolveModelId('auth', 'User');
    const browse = await resolveService(userModelId, 'browse');

    const created = await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        IrServiceId: browse.id,
        IrModelId: null,
        IrApplicationId: null,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );

    const id = String((created as any)?.Id || '').trim();
    expect(id.length > 0).toBe(true);

    await RoleMethodAccess.UpdateById(
      id,
      {
        Mode: 'deny',
      } as any,
      ['Id'] as any
    );

    const rows = await RoleMethodAccess.Search(
      ['Id', '=', id] as any,
      { fields: ['Id', 'IrServiceId', 'IrModelId', 'IrApplicationId', 'Mode'], limit: 1 } as any
    );

    expect(rows.length).toBe(1);
    // Scope must stay scoped to the service, not become global.
    expect(String((rows[0] as any)?.IrServiceId || '').trim()).toBe(browse.id);
    expect(String((rows[0] as any)?.IrModelId || '').trim()).toBe('');
    expect(String((rows[0] as any)?.IrApplicationId || '').trim()).toBe('');
    // Mode must reflect the update.
    expect(String((rows[0] as any)?.Mode || '').trim()).toBe('deny');
  });
});

test('RoleMethodAccess coverage: CreateMany and condition Update hit assertExclusiveScope', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole('ROLE_METHOD_SCOPE_COV');
    const userModelId = await resolveModelId('auth', 'User');
    const browse = await resolveService(userModelId, 'browse');

    const none = await RoleMethodAccess.CreateMany(null as any, ['Id'] as any);
    expect(Array.isArray(none)).toBe(true);
    expect(none.length).toBe(0);

    const many = await RoleMethodAccess.CreateMany(
      [
        {
          RoleId: { Id: role.id } as any,
          IrServiceId: browse.id,
          IrModelId: null,
          IrApplicationId: null,
          Mode: 'allow',
        } as any,
      ],
      ['Id', 'Mode'] as any
    );
    expect(many.length).toBe(1);
    const id = String((many[0] as any)?.Id || '').trim();
    expect(id.length > 0).toBe(true);

    await RoleMethodAccess.Update(['Id', '=', id] as any, { Mode: 'deny' } as any, ['Id'] as any);

    const rows = await RoleMethodAccess.Search(
      ['Id', '=', id] as any,
      { fields: ['Id', 'IrServiceId', 'IrModelId', 'IrApplicationId', 'Mode'], limit: 1 } as any
    );
    expect(rows.length).toBe(1);
    expect(String((rows[0] as any)?.IrServiceId || '').trim()).toBe(browse.id);
    expect(String((rows[0] as any)?.Mode || '').trim()).toBe('deny');
  });
});

test('RoleMethodAccess: Create/Update coerce Source=ui to manual (UI-Option-A)', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole('ROLE_METHOD_SOURCE_MANUAL');
    const userModelId = await resolveModelId('auth', 'User');
    const browse = await resolveService(userModelId, 'browse');

    const created = await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        IrServiceId: browse.id,
        IrModelId: null,
        IrApplicationId: null,
        Mode: 'allow',
        Source: 'ui',
      } as any,
      ['Id', 'Source'] as any
    );
    expect(String((created as any)?.Source || '').trim()).toBe('manual');

    const id = String((created as any)?.Id || '').trim();
    await RoleMethodAccess.UpdateById(id, { Source: 'ui' } as any, ['Id', 'Source'] as any);
    const rows = await RoleMethodAccess.Search(['Id', '=', id] as any, { fields: ['Id', 'Source'], limit: 1 } as any);
    expect(rows.length).toBe(1);
    expect(String((rows[0] as any)?.Source || '').trim()).toBe('manual');
  });
});
