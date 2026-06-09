// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import Role from '@/auth/service/models/role';
import RoleMethodAccess from '@/auth/service/models/role_method_access';
import RoleUiResource from '@/auth/service/models/role_ui_resource';
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

      'meta.IrUiResource:read',
      'meta.IrUiResource:write',
      'meta.IrUiResource:create',
      'meta.IrUiResource:delete',
      'IrUiResource:read',
      'IrUiResource:write',
      'IrUiResource:create',
      'IrUiResource:delete',

      'meta.IrModel:read',
      'meta.IrService:read',
      'meta.IrApplication:read',
      'IrModel:read',
      'IrService:read',
      'IrApplication:read',
    ],
  });
}

async function resolveApplicationId(applicationName: string): Promise<string> {
  const rows = await IrApplication.Search(
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
  const rows = await IrModel.Search(
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
  const rows = await IrService.Search(
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

async function listRoleUiGrants(roleId: string): Promise<Array<{ IrApplicationId: string; IrUiResourceId: string; Mode: string }>> {
  const rows = await RoleUiResource.Search(
    {
      And: [['RoleId', '=', roleId]],
    } as any,
    { fields: ['IrApplicationId', 'IrUiResourceId', 'Mode'], limit: 100 } as any
  );

  return (rows || []).map((row: any) => ({
    IrApplicationId: String(row?.IrApplicationId || '').trim(),
    IrUiResourceId: String(row?.IrUiResourceId || '').trim(),
    Mode: String(row?.Mode || '').trim(),
  }));
}

async function createUiResource(input: {
  resourceId: string;
  type: 'ROUTE' | 'MENU' | 'ACTION';
  requires?: string[];
  irApplicationId?: string | null;
}): Promise<string> {
  const row = await IrUiResource.Create(
    {
      Name: input.resourceId,
      Type: input.type,
      Requires: input.requires ?? [],
      IrApplicationId: input.irApplicationId ?? null,
      Module: 'auth',
    } as any,
    ['Id'] as any
  );

  const id = String((row as any)?.Id || '').trim();
  if (!id) throw new Error(`IrUiResource create failed: ${input.resourceId}`);
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
        IrApplicationId: null,
        IrUiResourceId: uiResourceId,
      } as any,
      ['Id'] as any
    );

    await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        IrServiceId: browseService.id,
        IrModelId: null,
        IrApplicationId: null,
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
      { fields: ['IrServiceId'], limit: 100 } as any
    );

    expect((manualRows || []).length > 0).toBe(true);
    expect(String((manualRows as any)[0]?.IrServiceId || '').trim()).toBe(browseService.id);
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
        IrApplicationId: null,
        IrUiResourceId: resourceId,
        Mode: 'deny',
      } as any,
      ['Id'] as any
    );

    const grants = await listRoleUiGrants(role.id);
    expect(grants.length).toBe(1);
    expect(grants[0]).toEqual({
      IrApplicationId: '',
      IrUiResourceId: resourceId,
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
        IrApplicationId: appId,
        IrUiResourceId: null,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );

    const grants = await listRoleUiGrants(role.id);
    expect(grants.length).toBe(1);
    expect(grants[0]).toEqual({
      IrApplicationId: appId,
      IrUiResourceId: '',
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
        IrApplicationId: null,
        IrUiResourceId: null,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );

    const grants = await listRoleUiGrants(role.id);
    expect(grants.length).toBe(1);
    expect(grants[0]).toEqual({
      IrApplicationId: '',
      IrUiResourceId: '',
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
        IrApplicationId: null,
        IrUiResourceId: resourceId,
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
        IrApplicationId: appId,
        IrUiResourceId: resourceId,
      } as any,
      ['Id', '=', grantId] as any
    );

    expect((updated || []).length).toBe(1);

    const rows = await RoleUiResource.Search(
      ['Id', '=', grantId] as any,
      {
        fields: ['Id', 'DeletedAt', 'IrApplicationId', 'IrUiResourceId'] as any,
        withDeleted: true,
      } as any
    );

    expect(rows.length).toBe(1);
    expect(String((rows[0] as any)?.IrApplicationId || '').trim()).toBe(appId);
    expect(String((rows[0] as any)?.IrUiResourceId || '').trim()).toBe(resourceId);
    expect((rows[0] as any)?.DeletedAt != null).toBe(true);
  });
});
