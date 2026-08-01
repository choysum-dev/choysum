// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import Role from '@/auth/service/models/role';
import RoleMethodAccess from '@/auth/service/models/role_method_access';
import RoleUiResource from '@/auth/service/models/role_ui_resource';
import MetaUiResource from '@/meta/service/models/ui_resource';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
import type MetaModelModel from '@/meta/service/models/model';
import type MetaServiceModel from '@/meta/service/models/service';
const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');
const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');
const MetaService = createServiceByModel<typeof MetaServiceModel>('meta.MetaService');

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
      'auth.Role:read',
      'auth.Role:write',
      'auth.Role:create',
      'auth.Role:delete',
      'Role:read',
      'Role:write',
      'Role:create',
      'Role:delete',

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

      'meta.MetaUiResource:read',
      'meta.MetaUiResource:write',
      'meta.MetaUiResource:create',
      'meta.MetaUiResource:delete',
      'MetaUiResource:read',
      'MetaUiResource:write',
      'MetaUiResource:create',
      'MetaUiResource:delete',

      'meta.MetaModel:read',
      'meta.MetaService:read',
      'meta.MetaApplication:read',
      'MetaModel:read',
      'MetaService:read',
      'MetaApplication:read',
    ],
  });
}

async function resolveApplicationId(applicationName: string): Promise<string> {
  const rows = await MetaApplication.Search(
    {
      And: [['Name', '=', applicationName]],
    } as any,
    { fields: ['Id'], limit: 1 }
  );
  const id = String((rows as any)?.[0]?.Id || '').trim();
  if (!id) throw new Error(`meta application not found: ${applicationName}`);
  return id;
}

async function resolveModelId(appName: string, modelName: string): Promise<string> {
  const rows = await MetaModel.Search(
    {
      And: [
        ['Name', '=', modelName],
        ['Application', '=', appName],
      ],
    } as any,
    { fields: ['Id'], limit: 1 }
  );
  const id = String((rows as any)?.[0]?.Id || '').trim();
  if (!id) throw new Error(`meta model not found: ${appName}.${modelName}`);
  return id;
}

async function resolveService(modelId: string, serviceName: string): Promise<{ id: string; name: string }> {
  const rows = await MetaService.Search(
    {
      And: [['ModelId', '=', modelId]],
    } as any,
    { fields: ['Id', 'Name'], limit: 5000 }
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

async function createRole(code: string): Promise<{ id: string }> {
  const row = await Role.Create(
    {
      Name: `Role ${code} ${uid('n')}`,
      Code: code,
      Description: 'role ui sync fixture',
      IsActive: true,
      IsSystem: false,
    } as any,
    ['Id'] as any
  );
  const id = String((row as any)?.Id || '').trim();
  if (!id) throw new Error('role create failed');
  return { id };
}

async function listRoleUiGrants(roleId: string): Promise<Array<{ MetaApplicationId: string; MetaUiResourceId: string; Mode: string }>> {
  const rows = await RoleUiResource.Search(
    {
      And: [['RoleId', '=', roleId]],
    } as any,
    { fields: ['MetaApplicationId', 'MetaUiResourceId', 'Mode'], limit: 100 } as any
  );

  return (rows || []).map((row: any) => ({
    MetaApplicationId: String(row?.MetaApplicationId || '').trim(),
    MetaUiResourceId: String(row?.MetaUiResourceId || '').trim(),
    Mode: String(row?.Mode || '').trim(),
  }));
}

async function createUiResource(input: {
  resourceId: string;
  type: 'ROUTE' | 'MENU' | 'ACTION';
  requires?: string[];
  irApplicationId?: string | null;
}): Promise<string> {
  const row = await MetaUiResource.Create(
    {
      Name: input.resourceId,
      Type: input.type,
      Requires: input.requires ?? [],
      MetaApplicationId: input.irApplicationId ?? null,
      Module: 'auth',
    } as any,
    ['Id'] as any
  );

  const id = String((row as any)?.Id || '').trim();
  if (!id) throw new Error(`MetaUiResource create failed: ${input.resourceId}`);
  return id;
}

test('RoleUiResource delete grant: keeps manual ACL untouched', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole(`ROLE_UI_SYNC_${uid('R')}`);

    const modelId = await resolveModelId('auth', 'User');
    const browseService = await resolveService(modelId, 'browse');

    const uiResourceId = await createUiResource({
      resourceId: `auth.route.user_delete_sync_${uid('X')}`,
      type: 'ROUTE',
      requires: ['rpc:/auth.User/*'],
    });

    const grant = await RoleUiResource.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaApplicationId: null,
        MetaUiResourceId: uiResourceId,
      } as any,
      ['Id'] as any
    );

    await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaServiceId: browseService.id,
        MetaModelId: null,
        MetaApplicationId: null,
        Mode: 'allow',
        Source: 'manual',
      } as any,
      ['Id'] as any
    );

    await RoleUiResource.DeleteById(String((grant as any)?.Id || '').trim());

    const manualRows = await RoleMethodAccess.Search(
      {
        And: [
          ['RoleId', '=', role.id],
          ['Source', '=', 'manual'],
        ],
      } as any,
      { fields: ['MetaServiceId'], limit: 100 } as any
    );

    expect((manualRows || []).length > 0).toBe(true);
    expect(String((manualRows as any)[0]?.MetaServiceId || '').trim()).toBe(browseService.id);
  });
});

test('RoleUiResource create: resource scope grant persists with mode', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole(`ROLE_UI_RESOURCE_SCOPE_${uid('R')}`);
    const resourceId = await createUiResource({
      resourceId: `auth.route.resource_scope_${uid('X')}`,
      type: 'ROUTE',
      requires: [],
    });

    await RoleUiResource.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaApplicationId: null,
        MetaUiResourceId: resourceId,
        Mode: 'deny',
      } as any,
      ['Id'] as any
    );

    const grants = await listRoleUiGrants(role.id);
    expect(grants.length).toBe(1);
    expect(grants[0]).toEqual({
      MetaApplicationId: '',
      MetaUiResourceId: resourceId,
      Mode: 'deny',
    });
  });
});

test('RoleUiResource create: application scope grant persists', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole(`ROLE_UI_APP_SCOPE_${uid('R')}`);
    const appId = await resolveApplicationId('auth');

    await RoleUiResource.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaApplicationId: appId,
        MetaUiResourceId: null,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );

    const grants = await listRoleUiGrants(role.id);
    expect(grants.length).toBe(1);
    expect(grants[0]).toEqual({
      MetaApplicationId: appId,
      MetaUiResourceId: '',
      Mode: 'allow',
    });
  });
});

test('RoleUiResource create: global scope grant persists', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole(`ROLE_UI_GLOBAL_SCOPE_${uid('R')}`);

    await RoleUiResource.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaApplicationId: null,
        MetaUiResourceId: null,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );

    const grants = await listRoleUiGrants(role.id);
    expect(grants.length).toBe(1);
    expect(grants[0]).toEqual({
      MetaApplicationId: '',
      MetaUiResourceId: '',
      Mode: 'allow',
    });
  });
});

test('RoleUiResource db check: deleted rows bypass scope xor', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole(`ROLE_UI_DELETED_CHECK_${uid('R')}`);
    const appId = await resolveApplicationId('auth');
    const resourceId = await createUiResource({
      resourceId: `auth.route.deleted_check_${uid('X')}`,
      type: 'ROUTE',
      requires: [],
    });

    const grant = await RoleUiResource.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaApplicationId: null,
        MetaUiResourceId: resourceId,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );

    const grantId = String((grant as any)?.Id || '').trim();
    expect(grantId.length > 0).toBe(true);

    await RoleUiResource.DeleteById(grantId);

    const repo = RoleUiResource.getRepository().withDeleted();
    const updated = await repo.update(
      {
        MetaApplicationId: appId,
        MetaUiResourceId: resourceId,
      } as any,
      ['Id', '=', grantId] as any
    );

    expect((updated || []).length).toBe(1);

    const rows = await RoleUiResource.Search(
      ['Id', '=', grantId] as any,
      {
        fields: ['Id', 'DeletedAt', 'MetaApplicationId', 'MetaUiResourceId'] as any,
        withDeleted: true,
      } as any
    );

    expect(rows.length).toBe(1);
    expect(String((rows[0] as any)?.MetaApplicationId || '').trim()).toBe(appId);
    expect(String((rows[0] as any)?.MetaUiResourceId || '').trim()).toBe(resourceId);
    expect((rows[0] as any)?.DeletedAt != null).toBe(true);
  });
});

test('RoleUiResource: permission-only update must not rewrite scoped fields to global', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole(`ROLE_UI_SCOPE_SAFE_${uid('R')}`);
    const resourceId = await createUiResource({
      resourceId: `auth.route.scope_safe_${uid('X')}`,
      type: 'ROUTE',
      requires: [],
    });

    const created = await RoleUiResource.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaApplicationId: null,
        MetaUiResourceId: resourceId,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );

    const id = String((created as any)?.Id || '').trim();
    expect(id.length > 0).toBe(true);

    await RoleUiResource.UpdateById(
      id,
      {
        Mode: 'deny',
      } as any,
      ['Id'] as any
    );

    const rows = await RoleUiResource.Search(
      ['Id', '=', id] as any,
      { fields: ['Id', 'MetaApplicationId', 'MetaUiResourceId', 'Mode'], limit: 1 } as any
    );

    expect(rows.length).toBe(1);
    // Scope must stay scoped to the UI resource, not become global.
    expect(String((rows[0] as any)?.MetaUiResourceId || '').trim()).toBe(resourceId);
    expect(String((rows[0] as any)?.MetaApplicationId || '').trim()).toBe('');
    // Mode must reflect the update.
    expect(String((rows[0] as any)?.Mode || '').trim()).toBe('deny');
  });
});

test('RoleUiResource coverage: condition Update hits assertExclusiveScope', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole(`ROLE_UI_SCOPE_COV_${uid('R')}`);
    const resourceId = await createUiResource({
      resourceId: `auth.route.scope_cov_${uid('X')}`,
      type: 'ROUTE',
      requires: [],
    });

    // Cover CreateMany `values || []` falsy branch.
    const none = await RoleUiResource.CreateMany(null as any, ['Id'] as any);
    expect(Array.isArray(none)).toBe(true);
    expect(none.length).toBe(0);

    const many = await RoleUiResource.CreateMany(
      [
        {
          RoleId: { Id: role.id } as any,
          MetaApplicationId: null,
          MetaUiResourceId: resourceId,
          Mode: 'allow',
        } as any,
      ],
      ['Id', 'Mode'] as any
    );
    expect(many.length).toBe(1);
    const id = String((many[0] as any)?.Id || '').trim();
    expect(id.length > 0).toBe(true);

    await RoleUiResource.Update(['Id', '=', id] as any, { Mode: 'deny' } as any, ['Id'] as any);

    const rows = await RoleUiResource.Search(
      ['Id', '=', id] as any,
      { fields: ['Id', 'MetaApplicationId', 'MetaUiResourceId', 'Mode'], limit: 1 } as any
    );
    expect(rows.length).toBe(1);
    expect(String((rows[0] as any)?.MetaUiResourceId || '').trim()).toBe(resourceId);
    expect(String((rows[0] as any)?.Mode || '').trim()).toBe('deny');
  });
});
