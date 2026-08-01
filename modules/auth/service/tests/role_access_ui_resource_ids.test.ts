// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import Role from '@/auth/service/models/role';
import RoleUiResource from '@/auth/service/models/role_ui_resource';
import MetaUiResource from '@/meta/service/models/ui_resource';
import MetaUiResourceMenuRoute from '@/meta/service/models/ui_resource_menu_route';
import MetaUiResourceRouteAction from '@/meta/service/models/ui_resource_route_action';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');

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

      'meta.MetaUiResourceMenuRoute:read',
      'meta.MetaUiResourceMenuRoute:write',
      'meta.MetaUiResourceMenuRoute:create',
      'meta.MetaUiResourceMenuRoute:delete',
      'MetaUiResourceMenuRoute:read',
      'MetaUiResourceMenuRoute:write',
      'MetaUiResourceMenuRoute:create',
      'MetaUiResourceMenuRoute:delete',

      'meta.MetaUiResourceRouteAction:read',
      'meta.MetaUiResourceRouteAction:write',
      'meta.MetaUiResourceRouteAction:create',
      'meta.MetaUiResourceRouteAction:delete',
      'MetaUiResourceRouteAction:read',
      'MetaUiResourceRouteAction:write',
      'MetaUiResourceRouteAction:create',
      'MetaUiResourceRouteAction:delete',

      'meta.MetaApplication:read',
      'MetaApplication:read',
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
  const row = await MetaUiResource.Create(
    {
      Name: input.resourceId,
      Type: input.type,
      ParentId: input.parentId ? ({ Id: input.parentId } as any) : null,
      Sequence: Number.isFinite(input.sequence) ? Number(input.sequence) : undefined,
      Requires: [],
      MetaApplicationId: input.irApplicationId ?? null,
      Module: 'auth',
    } as any,
    ['Id'] as any
  );

  const id = String((row as any)?.Id || '').trim();
  if (!id) throw new Error(`MetaUiResource create failed: ${input.resourceId}`);
  return id;
}

async function createMenuRouteRelation(menuUiResourceId: string, routeUiResourceId: string): Promise<void> {
  await MetaUiResourceMenuRoute.Create(
    {
      MenuUiResourceId: { Id: menuUiResourceId } as any,
      RouteUiResourceId: { Id: routeUiResourceId } as any,
    } as any,
    ['Id'] as any
  );
}

async function createRouteActionRelation(routeUiResourceId: string, actionUiResourceId: string): Promise<void> {
  await MetaUiResourceRouteAction.Create(
    {
      RouteUiResourceId: { Id: routeUiResourceId } as any,
      ActionUiResourceId: { Id: actionUiResourceId } as any,
    } as any,
    ['Id'] as any
  );
}

async function getUiResourceParentPath(id: string): Promise<string> {
  const rows = await MetaUiResource.Search(
    {
      And: [['Id', '=', id]],
    } as any,
    { fields: ['Id', 'ParentPath'], limit: 1 } as any
  );
  return String((rows as any)?.[0]?.ParentPath || '').trim();
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

async function listRoleUiGrants(roleId: string): Promise<Array<{ MetaApplicationId: string; MetaUiResourceId: string; Mode: string }>> {
  const rows = await RoleUiResource.Search(
    {
      And: [['RoleId', '=', roleId]],
    } as any,
    { fields: ['MetaApplicationId', 'MetaUiResourceId', 'Mode'], limit: 200 } as any
  );

  return (rows || []).map((row: any) => ({
    MetaApplicationId: String(row?.MetaApplicationId || '').trim(),
    MetaUiResourceId: String(row?.MetaUiResourceId || '').trim(),
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
      .filter(g => g.Mode === 'allow' && g.MetaUiResourceId && !g.MetaApplicationId)
      .map(g => g.MetaUiResourceId)
      .sort();

    expect(allowResourceIds).toEqual([resourceA, resourceB].sort());
  });
});

test('MetaUiResource.Create keeps ParentPath empty for non-MENU resources', async () => {
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

test('MetaUiResource.Childs returns one-level sorted projection without recursion', async () => {
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

    const [menuRoot] = await MetaUiResource.Search(
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

    const [routeA] = await MetaUiResource.Search(
      {
        And: [['Id', '=', routeAId]],
      } as any,
      { fields: ['Id', 'Type', 'Childs'], limit: 1 } as any
    );
    const routeChilds = ((routeA as any)?.Childs || []) as any[];
    expect(routeChilds.map(row => String(row?.Id || ''))).toEqual([actionA1Id, actionA2Id]);

    const [actionA1] = await MetaUiResource.Search(
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
          MetaApplicationId: null,
          MetaUiResourceId: allowOld,
        },
        {
          RoleId: { Id: role.id } as any,
          Mode: 'deny',
          MetaApplicationId: null,
          MetaUiResourceId: denyKeep,
        },
        {
          RoleId: { Id: role.id } as any,
          Mode: 'allow',
          MetaApplicationId: appId,
          MetaUiResourceId: null,
        },
        {
          RoleId: { Id: role.id } as any,
          Mode: 'deny',
          MetaApplicationId: null,
          MetaUiResourceId: null,
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

    const hasAllowOld = grants.some(g => g.Mode === 'allow' && g.MetaUiResourceId === allowOld && !g.MetaApplicationId);
    const hasAllowNew = grants.some(g => g.Mode === 'allow' && g.MetaUiResourceId === allowNew && !g.MetaApplicationId);
    const hasDenyKeep = grants.some(g => g.Mode === 'deny' && g.MetaUiResourceId === denyKeep && !g.MetaApplicationId);
    const hasAppScope = grants.some(g => g.Mode === 'allow' && !g.MetaUiResourceId && g.MetaApplicationId === appId);
    const hasGlobalDeny = grants.some(g => g.Mode === 'deny' && !g.MetaUiResourceId && !g.MetaApplicationId);

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
          MetaApplicationId: null,
          MetaUiResourceId: allowA,
        },
        {
          RoleId: { Id: role.id } as any,
          Mode: 'allow',
          MetaApplicationId: null,
          MetaUiResourceId: allowB,
        },
        {
          RoleId: { Id: role.id } as any,
          Mode: 'allow',
          MetaApplicationId: null,
          MetaUiResourceId: null,
        },
      ] as any,
      ['Id'] as any
    );

    const updated = (await Role.UpdateById(
      role.id,
      {
        AccessUiResourceIds: [],
      } as any,
      ['Id', 'AccessUiResourceIds', { UiResources: ['Mode', 'MetaApplicationId', 'MetaUiResourceId'] }] as any
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
      const uiId = String(row?.MetaUiResourceId || '').trim();
      const appId = String(row?.MetaApplicationId || '').trim();
      return mode === 'allow' && !!uiId && !appId;
    });
    const hasGlobalAllow = rows.some((row: any) => {
      const mode = String(row?.Mode || '')
        .trim()
        .toLowerCase();
      const uiId = String(row?.MetaUiResourceId || '').trim();
      const appId = String(row?.MetaApplicationId || '').trim();
      return mode === 'allow' && !uiId && !appId;
    });

    expect(hasAllowResourceScope).toBe(false);
    expect(hasGlobalAllow).toBe(true);
  });
});
