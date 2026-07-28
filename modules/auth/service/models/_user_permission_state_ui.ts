// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import IrUiResource from '@/meta/service/models/ir_ui_resource';
import IrUiResourceMenuRoute from '@/meta/service/models/ir_ui_resource_menu_route';
import IrUiResourceRouteAction from '@/meta/service/models/ir_ui_resource_route_action';
import RoleUiResource from './role_ui_resource';
import { isUiResourceAllowed, maybeId, normalizeScopeRefId, normalizeUiResourceId, parseJsonStringArray, sortStrings } from './_user_authz_shared';
import { normalizeRpcRequireKey } from '@/core/service/utils/normalization';
import { applyToScope, type AclAggregationResult } from './_user_permission_state_acl';

type UiResourceMeta = {
  dbId: string;
  resourceId: string;
  type: string;
  parentId: string;
  appId: string;
  requires: string[];
};

export type PermissionStateUiOutput = Record<string, { ui?: { routes?: string[]; menus?: string[]; actions?: string[] } }>;

function ensureBucket(byCompany: PermissionStateUiOutput, k: string): Required<NonNullable<PermissionStateUiOutput[string]>> {
  if (!byCompany[k]) byCompany[k] = { ui: { routes: [], menus: [], actions: [] } };
  if (!byCompany[k].ui) byCompany[k].ui = { routes: [], menus: [], actions: [] };
  if (!byCompany[k].ui!.routes) byCompany[k].ui!.routes = [];
  if (!byCompany[k].ui!.menus) byCompany[k].ui!.menus = [];
  if (!byCompany[k].ui!.actions) byCompany[k].ui!.actions = [];
  return byCompany[k] as Required<NonNullable<PermissionStateUiOutput[string]>>;
}

/**
 * Aggregate UI resource grants and compute the per-company ui.routes/menus/actions projection.
 */
export async function buildUiPermissionProjection(
  roleIds: string[],
  roleScopesById: Record<string, { global: boolean; companies: string[] }>,
  enabledCompanyIds: string[],
  acl: AclAggregationResult
): Promise<PermissionStateUiOutput> {
  const byCompany: PermissionStateUiOutput = {};

  ensureBucket(byCompany, '*');
  for (const cid of enabledCompanyIds || []) ensureBucket(byCompany, cid);

  if (roleIds.length === 0) return byCompany;

  const scope = (rid: string) => applyToScope.bind(null, rid, roleScopesById);

  // 3) Generate ui.routes, ui.menus, and ui.actions from UI grants and RPC requires.
  const uiGrants = await RoleUiResource.Search(['RoleId', 'in', roleIds] as any, {
    fields: ['RoleId', 'IrApplicationId', 'IrUiResourceId', 'Mode'],
    limit: 50000,
  });

  const explicitGlobalUiAllowByCompany = new Set<string>();
  const explicitGlobalUiDenyByCompany = new Set<string>();
  const explicitAppUiAllowByCompany = new Map<string, Set<string>>();
  const explicitAppUiDenyByCompany = new Map<string, Set<string>>();
  const explicitResourceUiAllowByCompany = new Map<string, Set<string>>();
  const explicitResourceUiDenyByCompany = new Map<string, Set<string>>();

  const ensureExplicitAppSet = (companyKey: string): Set<string> => {
    let set = explicitAppUiAllowByCompany.get(companyKey);
    if (!set) {
      set = new Set<string>();
      explicitAppUiAllowByCompany.set(companyKey, set);
    }
    return set;
  };
  const ensureExplicitAppDenySet = (companyKey: string): Set<string> => {
    let set = explicitAppUiDenyByCompany.get(companyKey);
    if (!set) {
      set = new Set<string>();
      explicitAppUiDenyByCompany.set(companyKey, set);
    }
    return set;
  };
  const ensureExplicitResourceSet = (companyKey: string): Set<string> => {
    let set = explicitResourceUiAllowByCompany.get(companyKey);
    if (!set) {
      set = new Set<string>();
      explicitResourceUiAllowByCompany.set(companyKey, set);
    }
    return set;
  };
  const ensureExplicitResourceDenySet = (companyKey: string): Set<string> => {
    let set = explicitResourceUiDenyByCompany.get(companyKey);
    if (!set) {
      set = new Set<string>();
      explicitResourceUiDenyByCompany.set(companyKey, set);
    }
    return set;
  };

  for (const g of uiGrants || []) {
    const roleId = maybeId((g as any).RoleId);
    const appId = normalizeScopeRefId((g as any).IrApplicationId);
    const resourceId = normalizeUiResourceId((g as any).IrUiResourceId);
    const mode = String((g as any).Mode ?? 'allow')
      .trim()
      .toLowerCase();
    if (!roleId) continue;
    if (mode !== 'allow' && mode !== 'deny') continue;

    if (!appId && !resourceId) {
      scope(roleId)(companyKey => {
        if (mode === 'allow') explicitGlobalUiAllowByCompany.add(companyKey);
        else explicitGlobalUiDenyByCompany.add(companyKey);
      });
      continue;
    }

    if (appId && !resourceId) {
      scope(roleId)(companyKey => {
        if (mode === 'allow') ensureExplicitAppSet(companyKey).add(appId);
        else ensureExplicitAppDenySet(companyKey).add(appId);
      });
      continue;
    }

    if (!appId && resourceId) {
      scope(roleId)(companyKey => {
        if (mode === 'allow') ensureExplicitResourceSet(companyKey).add(resourceId);
        else ensureExplicitResourceDenySet(companyKey).add(resourceId);
      });
      continue;
    }
  }

  const resources = await IrUiResource.Search(
    [] as any,
    {
      fields: ['Id', 'Name', 'Type', 'ParentId', 'IrApplicationId', 'Requires'],
      limit: 100000,
    } as any
  );

  const resourceNameById = new Map<string, string>();
  for (const row of resources || []) {
    const id = String((row as any)?.Id ?? (row as any)?.id ?? '').trim();
    const name = String((row as any)?.Name ?? (row as any)?.name ?? '').trim();
    if (!id || !name) continue;
    resourceNameById.set(id, name);
  }

  const allResources: UiResourceMeta[] = (resources || [])
    .map((row: any) => ({
      dbId: String(row?.Id ?? row?.id ?? '').trim(),
      resourceId: String(row?.Name ?? row?.name ?? '').trim(),
      type: String(row?.Type || '')
        .trim()
        .toUpperCase(),
      parentId: (() => {
        const pid = normalizeUiResourceId(row?.ParentId ?? row?.parentId);
        return pid ? String(resourceNameById.get(pid) || '').trim() : '';
      })(),
      appId: normalizeScopeRefId(row?.IrApplicationId),
      requires: parseJsonStringArray((row as any)?.Requires ?? (row as any)?.requires),
    }))
    .filter(r => !!r.resourceId && (r.type === 'ROUTE' || r.type === 'MENU' || r.type === 'ACTION'));

  const [menuRouteRows, routeActionRows] = await Promise.all([
    IrUiResourceMenuRoute.Search([] as any, { fields: ['MenuUiResourceId', 'RouteUiResourceId'], limit: 100000 } as any),
    IrUiResourceRouteAction.Search([] as any, { fields: ['RouteUiResourceId', 'ActionUiResourceId'], limit: 100000 } as any),
  ]);

  const resourceMetaById = new Map<string, UiResourceMeta>();
  for (const resource of allResources) {
    resourceMetaById.set(resource.resourceId, resource);
  }

  const menuParentById = new Map<string, string>();
  for (const r of allResources) {
    if (r.type === 'MENU' && r.parentId) menuParentById.set(r.resourceId, r.parentId);
  }

  const menuIdsByRouteId = new Map<string, Set<string>>();
  for (const row of menuRouteRows || []) {
    const menuDbId = normalizeUiResourceId((row as any)?.MenuUiResourceId);
    const routeDbId = normalizeUiResourceId((row as any)?.RouteUiResourceId);
    const menuId = menuDbId ? String(resourceNameById.get(menuDbId) || '').trim() : '';
    const routeId = routeDbId ? String(resourceNameById.get(routeDbId) || '').trim() : '';
    if (!menuId || !routeId) continue;
    let menuIds = menuIdsByRouteId.get(routeId);
    if (!menuIds) {
      menuIds = new Set<string>();
      menuIdsByRouteId.set(routeId, menuIds);
    }
    menuIds.add(menuId);
  }

  const routeIdsByActionId = new Map<string, Set<string>>();
  for (const row of routeActionRows || []) {
    const routeDbId = normalizeUiResourceId((row as any)?.RouteUiResourceId);
    const actionDbId = normalizeUiResourceId((row as any)?.ActionUiResourceId);
    const routeId = routeDbId ? String(resourceNameById.get(routeDbId) || '').trim() : '';
    const actionId = actionDbId ? String(resourceNameById.get(actionDbId) || '').trim() : '';
    if (!routeId || !actionId) continue;
    let routeIds = routeIdsByActionId.get(actionId);
    if (!routeIds) {
      routeIds = new Set<string>();
      routeIdsByActionId.set(actionId, routeIds);
    }
    routeIds.add(routeId);
  }

  const companyKeys = Array.from(new Set(['*', ...(enabledCompanyIds || [])]));
  for (const companyKey of companyKeys) {
    const bucket = ensureBucket(byCompany, companyKey);
    const ui = bucket.ui!;

    const hasGlobalAllow = acl.companyGlobalAllow.has(companyKey) || (companyKey !== '*' && acl.companyGlobalAllow.has('*'));
    const hasGlobalDeny = acl.companyGlobalDeny.has(companyKey) || (companyKey !== '*' && acl.companyGlobalDeny.has('*'));
    const hasExplicitGlobalUiAllow = explicitGlobalUiAllowByCompany.has(companyKey) || (companyKey !== '*' && explicitGlobalUiAllowByCompany.has('*'));
    const hasExplicitGlobalUiDeny = explicitGlobalUiDenyByCompany.has(companyKey) || (companyKey !== '*' && explicitGlobalUiDenyByCompany.has('*'));
    const explicitAppAllow = new Set<string>([
      ...(explicitAppUiAllowByCompany.get(companyKey) ?? []),
      ...(companyKey !== '*' ? (explicitAppUiAllowByCompany.get('*') ?? []) : []),
    ]);
    const explicitAppDeny = new Set<string>([
      ...(explicitAppUiDenyByCompany.get(companyKey) ?? []),
      ...(companyKey !== '*' ? (explicitAppUiDenyByCompany.get('*') ?? []) : []),
    ]);
    const explicitResourceAllow = new Set<string>([
      ...(explicitResourceUiAllowByCompany.get(companyKey) ?? []),
      ...(companyKey !== '*' ? (explicitResourceUiAllowByCompany.get('*') ?? []) : []),
    ]);
    const explicitResourceDeny = new Set<string>([
      ...(explicitResourceUiDenyByCompany.get(companyKey) ?? []),
      ...(companyKey !== '*' ? (explicitResourceUiDenyByCompany.get('*') ?? []) : []),
    ]);
    const hasAnyExplicitUiDeny = hasExplicitGlobalUiDeny || explicitAppDeny.size > 0 || explicitResourceDeny.size > 0;

    const isExplicitUiDenied = (resource: { dbId: string; resourceId: string; appId: string | null }): boolean =>
      hasExplicitGlobalUiDeny ||
      (!!resource.appId && explicitAppDeny.has(resource.appId)) ||
      explicitResourceDeny.has(resource.dbId) ||
      explicitResourceDeny.has(resource.resourceId);

    const isExplicitUiAllowed = (resource: { dbId: string; resourceId: string; appId: string | null }): boolean =>
      !isExplicitUiDenied(resource) &&
      (hasExplicitGlobalUiAllow ||
        (!!resource.appId && explicitAppAllow.has(resource.appId)) ||
        explicitResourceAllow.has(resource.dbId) ||
        explicitResourceAllow.has(resource.resourceId));

    if (!hasAnyExplicitUiDeny && ((hasGlobalAllow && !hasGlobalDeny) || hasExplicitGlobalUiAllow)) {
      ui.routes = ['*'];
      ui.menus = ['*'];
      ui.actions = ['*'];
      continue;
    }

    const requiresAllowSet = new Set<string>([
      ...(acl.requiresAllowKeysByCompany.get(companyKey) ?? []),
      ...(companyKey !== '*' ? (acl.requiresAllowKeysByCompany.get('*') ?? []) : []),
    ]);
    const requiresDenySet = new Set<string>([
      ...(acl.requiresDenyKeysByCompany.get(companyKey) ?? []),
      ...(companyKey !== '*' ? (acl.requiresDenyKeysByCompany.get('*') ?? []) : []),
    ]);

    // UI-Option-A Method half for ACTION only: explicit UI allow derives Method allow for Requires.
    const actionMethodAllowSet = new Set<string>(requiresAllowSet);
    for (const r of allResources) {
      if (isExplicitUiDenied(r) || !isExplicitUiAllowed(r)) continue;
      for (const req of r.requires) {
        const k = normalizeRpcRequireKey(req);
        if (k) actionMethodAllowSet.add(k);
      }
    }

    const routeSet = new Set<string>();
    const menuSet = new Set<string>();
    const actionSet = new Set<string>();
    const explicitRouteSet = new Set<string>();
    const explicitActionSet = new Set<string>();

    for (const r of allResources) {
      const explicitlyDenied = isExplicitUiDenied(r);
      if (explicitlyDenied) continue;
      const allowedByExplicit = isExplicitUiAllowed(r);
      const allowedByRequires = isUiResourceAllowed(r.requires, requiresAllowSet, requiresDenySet);
      if (!allowedByExplicit && !allowedByRequires) continue;
      if (r.type === 'ROUTE') {
        routeSet.add(r.resourceId);
        if (allowedByExplicit) explicitRouteSet.add(r.resourceId);
      } else if (r.type === 'MENU') menuSet.add(r.resourceId);
      else if (r.type === 'ACTION') {
        // Write Action visibility ⇔ UI ∧ Method (Requires). Method deny-wins hides the button.
        const methodOk = isUiResourceAllowed(r.requires, actionMethodAllowSet, requiresDenySet);
        if (!methodOk) continue;
        actionSet.add(r.resourceId);
        if (allowedByExplicit) explicitActionSet.add(r.resourceId);
      }
    }

    for (const actionId of explicitActionSet) {
      for (const routeId of routeIdsByActionId.get(actionId) ?? []) {
        const route = resourceMetaById.get(routeId);
        if (!route || route.type !== 'ROUTE' || isExplicitUiDenied(route)) continue;
        routeSet.add(routeId);
        explicitRouteSet.add(routeId);
      }
    }

    for (const routeId of explicitRouteSet) {
      for (const menuId of menuIdsByRouteId.get(routeId) ?? []) {
        const menu = resourceMetaById.get(menuId);
        if (!menu || isExplicitUiDenied(menu)) continue;
        menuSet.add(menuId);
      }
    }

    for (const menuId of Array.from(menuSet)) {
      let parentId = menuParentById.get(menuId);
      let deniedByAncestor = false;
      const visited = new Set<string>();
      while (parentId && !visited.has(parentId)) {
        visited.add(parentId);
        const parent = resourceMetaById.get(parentId);
        if (parent && isExplicitUiDenied(parent)) {
          deniedByAncestor = true;
          break;
        }
        parentId = menuParentById.get(parentId);
      }
      if (deniedByAncestor) menuSet.delete(menuId);
    }

    const pending = Array.from(menuSet);
    while (pending.length > 0) {
      const cur = pending.pop()!;
      const parentId = menuParentById.get(cur);
      if (!parentId) continue;
      if (menuSet.has(parentId)) continue;
      const parent = resourceMetaById.get(parentId);
      if (parent && isExplicitUiDenied(parent)) {
        menuSet.delete(cur);
        continue;
      }
      menuSet.add(parentId);
      pending.push(parentId);
    }

    ui.routes = sortStrings(Array.from(routeSet));
    ui.menus = sortStrings(Array.from(menuSet));
    ui.actions = sortStrings(Array.from(actionSet));
  }

  return byCompany;
}
