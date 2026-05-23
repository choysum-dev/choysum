// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Company scope selector used by permission lookups.
 */
export type CompanyScope = 'enabled' | 'active';

/**
 * Company context carried by the auth token or current request state.
 */
export type PermissionCtx = {
  activeCompanyId?: string;
  enabledCompanyIds?: string[];
};

/**
 * UI resource buckets materialized from the backend permission state.
 */
export type PermissionUiSet = {
  routes?: string[];
  menus?: string[];
  actions?: string[];
};

/**
 * Client-side permission snapshot returned by auth.User.GetPermissionState.
 */
export type PermissionState = {
  permStateVersion: number;
  byCompany: Record<
    string,
    {
      ui?: PermissionUiSet;
    }
  >;
};

/**
 * Normalize a company identifier into a trimmed string.
 */
function normalizeCompanyId(v: unknown): string {
  return String(v ?? '').trim();
}

/**
 * Normalize a UI resource identifier into a trimmed string.
 */
function normalizeResourceId(v: unknown): string {
  return String(v ?? '').trim();
}

/**
 * Remove duplicate strings while preserving first-seen order.
 */
function uniq(arr: string[]): string[] {
  return Array.from(new Set(arr));
}

/**
 * Build the effective UI resource set for one company bucket.
 */
function effectiveUiSetForCompany(state: PermissionState, companyId: string, kind: keyof PermissionUiSet): Set<string> {
  const by = state?.byCompany ?? {};
  const g = by['*']?.ui ?? {};
  const c = by[companyId]?.ui ?? {};

  const globalValues = Array.isArray(g[kind]) ? g[kind].map(normalizeResourceId).filter(Boolean) : [];
  const companyValues = Array.isArray(c[kind]) ? c[kind].map(normalizeResourceId).filter(Boolean) : [];
  return new Set(uniq([...globalValues, ...companyValues]));
}

/**
 * Merge effective UI resource sets across multiple enabled companies.
 */
function effectiveUiSetForCompanies(state: PermissionState, companyIds: string[], kind: keyof PermissionUiSet): Set<string> {
  const out = new Set<string>();
  for (const cid of companyIds) {
    for (const key of effectiveUiSetForCompany(state, cid, kind)) out.add(key);
  }
  return out;
}

/**
 * Resolve the effective UI set for the requested company scope.
 */
function effectiveUiSet(state: PermissionState, ctx: PermissionCtx, scope: CompanyScope, kind: keyof PermissionUiSet): Set<string> {
  const activeCompanyId = normalizeCompanyId(ctx.activeCompanyId);
  const enabledCompanyIds = (ctx.enabledCompanyIds ?? []).map(normalizeCompanyId).filter(Boolean);

  if (scope === 'active') {
    if (!activeCompanyId) return new Set<string>();
    return effectiveUiSetForCompany(state, activeCompanyId, kind);
  }
  if (enabledCompanyIds.length > 0) return effectiveUiSetForCompanies(state, enabledCompanyIds, kind);
  if (!activeCompanyId) return new Set<string>();
  return effectiveUiSetForCompany(state, activeCompanyId, kind);
}

/**
 * Check whether one resource Id is allowed within the chosen UI bucket.
 */
function canResource(
  id: string | undefined,
  kind: keyof PermissionUiSet,
  state: PermissionState | null | undefined,
  ctx: PermissionCtx,
  scope: CompanyScope = 'enabled'
): boolean {
  const rid = String(id || '').trim();
  if (!rid) return true;
  if (!state) return false;
  const set = effectiveUiSet(state, ctx, scope, kind);
  if (set.has('*')) return true;
  return set.has(rid);
}

/**
 * Check whether a route resource is allowed.
 */
export function canRoute(id: string | undefined, state: PermissionState | null | undefined, ctx: PermissionCtx, scope: CompanyScope = 'enabled'): boolean {
  return canResource(id, 'routes', state, ctx, scope);
}

/**
 * Check whether a menu resource is allowed.
 */
export function canMenu(id: string | undefined, state: PermissionState | null | undefined, ctx: PermissionCtx, scope: CompanyScope = 'enabled'): boolean {
  return canResource(id, 'menus', state, ctx, scope);
}

/**
 * Check whether an action resource is allowed.
 */
export function hasAction(id: string | undefined, state: PermissionState | null | undefined, ctx: PermissionCtx, scope: CompanyScope = 'enabled'): boolean {
  return canResource(id, 'actions', state, ctx, scope);
}
