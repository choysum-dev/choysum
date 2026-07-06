// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import Role from '@/auth/service/models/role';
import RoleUiResource from '@/auth/service/models/role_ui_resource';
import IrUiResource from '@/meta/service/models/ir_ui_resource';
import IrUiResourceMenuRoute from '@/meta/service/models/ir_ui_resource_menu_route';
import IrUiResourceRouteAction from '@/meta/service/models/ir_ui_resource_route_action';
import { createServiceByModel } from '@/core/service/rpc';
import type IrApplicationModel from '@/meta/service/models/ir_application';
const IrApplication = createServiceByModel<typeof IrApplicationModel>('meta.IrApplication');

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  let jsCtx: any;
  const getRequestContext = root.getRequestContext;
  if (typeof getRequestContext === 'function') {
    try {
      jsCtx = getRequestContext();
    } catch {
      jsCtx = undefined;
    }
  }
  if (!jsCtx || typeof jsCtx !== 'object') {
    if (!root.context || typeof root.context !== 'object') root.context = {};
    jsCtx = root.context;
  }

  root.context = jsCtx;
  root.request.context = jsCtx;
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

      'meta.IrUiResourceMenuRoute:read',
      'meta.IrUiResourceMenuRoute:write',
      'meta.IrUiResourceMenuRoute:create',
      'meta.IrUiResourceMenuRoute:delete',
      'IrUiResourceMenuRoute:read',
      'IrUiResourceMenuRoute:write',
      'IrUiResourceMenuRoute:create',
      'IrUiResourceMenuRoute:delete',

      'meta.IrUiResourceRouteAction:read',
      'meta.IrUiResourceRouteAction:write',
      'meta.IrUiResourceRouteAction:create',
      'meta.IrUiResourceRouteAction:delete',
      'IrUiResourceRouteAction:read',
      'IrUiResourceRouteAction:write',
      'IrUiResourceRouteAction:create',
      'IrUiResourceRouteAction:delete',

      'meta.IrApplication:read',
      'IrApplication:read',
    ],
  });
}

async function createRole(code: string): Promise<{ id: string }> {
  const row = await Role.Create(
    {
      Name: `Role ${code} ${uid('n')}`,
      Code: code,
      Description: 'role access resource fixture',
      IsActive: true,
      IsSystem: false,
    } as any,
    ['Id'] as any
  );
  const id = String((row as any)?.Id || '').trim();
  if (!id) throw new Error('role create failed');
  return { id };
}

async function createUiResource(input: {
  resourceId: string;
  type: 'ROUTE' | 'MENU' | 'ACTION';
  irApplicationId?: string | null;
  parentId?: string | null;
  sequence?: number;
}): Promise<string> {
  const row = await IrUiResource.Create(
    {
      Name: input.resourceId,
      Type: input.type,
      ParentId: input.parentId ? ({ Id: input.parentId } as any) : null,
      Sequence: Number.isFinite(input.sequence) ? Number(input.sequence) : undefined,
      Requires: [],
      IrApplicationId: input.irApplicationId ?? null,
      Module: 'auth',
    } as any,
    ['Id'] as any
  );

  const id = String((row as any)?.Id || '').trim();
  if (!id) throw new Error(`IrUiResource create failed: ${input.resourceId}`);
  return id;
}

async function createMenuRouteRelation(menuUiResourceId: string, routeUiResourceId: string): Promise<void> {
  await IrUiResourceMenuRoute.Create(
    {
      MenuUiResourceId: { Id: menuUiResourceId } as any,
      RouteUiResourceId: { Id: routeUiResourceId } as any,
    } as any,
    ['Id'] as any
  );
}

async function createRouteActionRelation(routeUiResourceId: string, actionUiResourceId: string): Promise<void> {
  await IrUiResourceRouteAction.Create(
    {
      RouteUiResourceId: { Id: routeUiResourceId } as any,
      ActionUiResourceId: { Id: actionUiResourceId } as any,
    } as any,
    ['Id'] as any
  );
}

async function getUiResourceParentPath(id: string): Promise<string> {
  const rows = await IrUiResource.Search(
    {
      And: [['Id', '=', id]],
    } as any,
    { fields: ['Id', 'ParentPath'], limit: 1 } as any
  );
  return String((rows as any)?.[0]?.ParentPath || '').trim();
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

async function listRoleUiGrants(roleId: string): Promise<Array<{ IrApplicationId: string; IrUiResourceId: string; Mode: string }>> {
  const rows = await RoleUiResource.Search(
    {
      And: [['RoleId', '=', roleId]],
    } as any,
    { fields: ['IrApplicationId', 'IrUiResourceId', 'Mode'], limit: 200 } as any
  );

  return (rows || []).map((row: any) => ({
    IrApplicationId: String(row?.IrApplicationId || '').trim(),
    IrUiResourceId: String(row?.IrUiResourceId || '').trim(),
    Mode: String(row?.Mode || '').trim(),
  }));
}

test('Role.Create with AccessUiResourceIds writes allow resource-scope grants', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const resourceA = await createUiResource({
      resourceId: `auth.route.access_create_a_${uid('X')}`,
      type: 'ROUTE',
    });
    const resourceB = await createUiResource({
      resourceId: `auth.route.access_create_b_${uid('X')}`,
      type: 'ROUTE',
    });

    const created = await Role.Create(
      {
        Name: `Role access create ${uid('N')}`,
        Code: `ROLE_ACCESS_CREATE_${uid('R')}`,
        IsActive: true,
        AccessUiResourceIds: [resourceA, resourceB],
      } as any,
      ['Id'] as any
    );

    const roleId = String((created as any)?.Id || '').trim();
    const grants = await listRoleUiGrants(roleId);

    const allowResourceIds = grants
      .filter(g => g.Mode === 'allow' && g.IrUiResourceId && !g.IrApplicationId)
      .map(g => g.IrUiResourceId)
      .sort();

    expect(allowResourceIds).toEqual([resourceA, resourceB].sort());
  });
});

test('IrUiResource.Create keeps ParentPath empty for non-MENU resources', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const routeId = await createUiResource({
      resourceId: `auth.route.parent_path_route_${uid('X')}`,
      type: 'ROUTE',
    });
    const actionId = await createUiResource({
      resourceId: `auth.action.parent_path_action_${uid('X')}`,
      type: 'ACTION',
    });

    expect(await getUiResourceParentPath(routeId)).toBe('');
    expect(await getUiResourceParentPath(actionId)).toBe('');
  });
});

test('IrUiResource.Childs returns one-level sorted projection without recursion', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const menuRootId = await createUiResource({
      resourceId: `auth.menu.childs_root_${uid('X')}`,
      type: 'MENU',
      sequence: 10,
    });
    const menuChildId = await createUiResource({
      resourceId: `auth.menu.childs_menu_${uid('X')}`,
      type: 'MENU',
      parentId: menuRootId,
      sequence: 5,
    });

    const routeAId = await createUiResource({
      resourceId: `auth.route.childs_route_a_${uid('X')}`,
      type: 'ROUTE',
      sequence: 20,
    });
    const routeBId = await createUiResource({
      resourceId: `auth.route.childs_route_b_${uid('X')}`,
      type: 'ROUTE',
      sequence: 30,
    });
    await createMenuRouteRelation(menuRootId, routeAId);
    await createMenuRouteRelation(menuRootId, routeBId);

    const actionA1Id = await createUiResource({
      resourceId: `auth.action.childs_action_a1_${uid('X')}`,
      type: 'ACTION',
      sequence: 2,
    });
    const actionA2Id = await createUiResource({
      resourceId: `auth.action.childs_action_a2_${uid('X')}`,
      type: 'ACTION',
      sequence: 7,
    });
    await createRouteActionRelation(routeAId, actionA2Id);
    await createRouteActionRelation(routeAId, actionA1Id);

    const [menuRoot] = await IrUiResource.Search(
      {
        And: [['Id', '=', menuRootId]],
      } as any,
      { fields: ['Id', 'Type', 'Childs'], limit: 1 } as any
    );
    const menuChilds = ((menuRoot as any)?.Childs || []) as any[];
    expect(menuChilds.map(row => String(row?.Id || ''))).toEqual([menuChildId, routeAId, routeBId]);
    expect(
      menuChilds.some(row => {
        const entity = typeof row?.toEntity === 'function' ? row.toEntity() : {};
        return Object.prototype.hasOwnProperty.call(entity, 'Childs');
      })
    ).toBe(false);

    const [routeA] = await IrUiResource.Search(
      {
        And: [['Id', '=', routeAId]],
      } as any,
      { fields: ['Id', 'Type', 'Childs'], limit: 1 } as any
    );
    const routeChilds = ((routeA as any)?.Childs || []) as any[];
    expect(routeChilds.map(row => String(row?.Id || ''))).toEqual([actionA1Id, actionA2Id]);

    const [actionA1] = await IrUiResource.Search(
      {
        And: [['Id', '=', actionA1Id]],
      } as any,
      { fields: ['Id', 'Type', 'Childs'], limit: 1 } as any
    );
    expect(Array.isArray((actionA1 as any)?.Childs)).toBe(true);
    expect(((actionA1 as any)?.Childs || []).length).toBe(0);
  });
});

test('Role.UpdateById with AccessUiResourceIds replaces allow resource-scope only', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole(`ROLE_ACCESS_UPDATE_${uid('R')}`);
    const appId = await resolveApplicationId('auth');

    const allowOld = await createUiResource({
      resourceId: `auth.route.access_update_allow_old_${uid('X')}`,
      type: 'ROUTE',
    });
    const allowNew = await createUiResource({
      resourceId: `auth.route.access_update_allow_new_${uid('X')}`,
      type: 'ROUTE',
    });
    const denyKeep = await createUiResource({
      resourceId: `auth.route.access_update_deny_keep_${uid('X')}`,
      type: 'ROUTE',
    });

    await RoleUiResource.CreateMany(
      [
        {
          RoleId: { Id: role.id } as any,
          Mode: 'allow',
          IrApplicationId: null,
          IrUiResourceId: allowOld,
        },
        {
          RoleId: { Id: role.id } as any,
          Mode: 'deny',
          IrApplicationId: null,
          IrUiResourceId: denyKeep,
        },
        {
          RoleId: { Id: role.id } as any,
          Mode: 'allow',
          IrApplicationId: appId,
          IrUiResourceId: null,
        },
        {
          RoleId: { Id: role.id } as any,
          Mode: 'deny',
          IrApplicationId: null,
          IrUiResourceId: null,
        },
      ] as any,
      ['Id'] as any
    );

    await Role.UpdateById(
      role.id,
      {
        AccessUiResourceIds: [allowNew],
      } as any,
      ['Id'] as any
    );

    const grants = await listRoleUiGrants(role.id);

    const hasAllowOld = grants.some(g => g.Mode === 'allow' && g.IrUiResourceId === allowOld && !g.IrApplicationId);
    const hasAllowNew = grants.some(g => g.Mode === 'allow' && g.IrUiResourceId === allowNew && !g.IrApplicationId);
    const hasDenyKeep = grants.some(g => g.Mode === 'deny' && g.IrUiResourceId === denyKeep && !g.IrApplicationId);
    const hasAppScope = grants.some(g => g.Mode === 'allow' && !g.IrUiResourceId && g.IrApplicationId === appId);
    const hasGlobalDeny = grants.some(g => g.Mode === 'deny' && !g.IrUiResourceId && !g.IrApplicationId);

    expect(hasAllowOld).toBe(false);
    expect(hasAllowNew).toBe(true);
    expect(hasDenyKeep).toBe(true);
    expect(hasAppScope).toBe(true);
    expect(hasGlobalDeny).toBe(true);
  });
});

test('Role.UpdateById returns refreshed UiResources after clearing AccessUiResourceIds', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole(`ROLE_ACCESS_RETURN_${uid('R')}`);

    const allowA = await createUiResource({
      resourceId: `auth.route.access_return_allow_a_${uid('X')}`,
      type: 'ROUTE',
    });
    const allowB = await createUiResource({
      resourceId: `auth.route.access_return_allow_b_${uid('X')}`,
      type: 'ROUTE',
    });

    await RoleUiResource.CreateMany(
      [
        {
          RoleId: { Id: role.id } as any,
          Mode: 'allow',
          IrApplicationId: null,
          IrUiResourceId: allowA,
        },
        {
          RoleId: { Id: role.id } as any,
          Mode: 'allow',
          IrApplicationId: null,
          IrUiResourceId: allowB,
        },
        {
          RoleId: { Id: role.id } as any,
          Mode: 'allow',
          IrApplicationId: null,
          IrUiResourceId: null,
        },
      ] as any,
      ['Id'] as any
    );

    const updated = (await Role.UpdateById(
      role.id,
      {
        AccessUiResourceIds: [],
      } as any,
      ['Id', 'AccessUiResourceIds', { UiResources: ['Mode', 'IrApplicationId', 'IrUiResourceId'] }] as any
    )) as any;

    const updatedAccess = Array.isArray(updated?.AccessUiResourceIds) ? updated.AccessUiResourceIds : [];
    expect(updatedAccess).toEqual([]);

    const rows = (Array.isArray(updated?.UiResources) ? updated.UiResources : []).map((row: any) =>
      typeof row?.toEntity === 'function' ? row.toEntity() : row
    );

    const hasAllowResourceScope = rows.some((row: any) => {
      const mode = String(row?.Mode || '')
        .trim()
        .toLowerCase();
      const uiId = String(row?.IrUiResourceId || '').trim();
      const appId = String(row?.IrApplicationId || '').trim();
      return mode === 'allow' && !!uiId && !appId;
    });
    const hasGlobalAllow = rows.some((row: any) => {
      const mode = String(row?.Mode || '')
        .trim()
        .toLowerCase();
      const uiId = String(row?.IrUiResourceId || '').trim();
      const appId = String(row?.IrApplicationId || '').trim();
      return mode === 'allow' && !uiId && !appId;
    });

    expect(hasAllowResourceScope).toBe(false);
    expect(hasGlobalAllow).toBe(true);
  });
});
