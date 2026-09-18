// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { memoizeInReqState } from '@/core/service/api/context';
import { condition } from '@/core/service/api/query';
import type { BaseQueryCondition } from '@/core/service/api/query';
import type { Projected } from '@/core/service/api/selection';
import { uniqStrings } from '@/core/service/utils/normalization';
import { sortStrings, maybeId, withPermissionGraphBypass } from './_authz_shared';
import Role from '../role';
import RoleFieldRule from '../role_field_rule';
import RoleInheritance from '../role_inheritance';
import RoleMethodAccess from '../role_method_access';
import RoleRecordRule from '../role_record_rule';
import RoleUiResource from '../role_ui_resource';
import UserRole from '../user_role';

/** Multi-model Search surface for {@link maxUpdatedAt} (UserRole / Role / inheritance / ACL tables). */
type SearchableForMaxUpdatedAt = {
  Search: (cond: BaseQueryCondition | [], opts: object) => Promise<Array<{ UpdatedAt?: unknown }>>;
};

/**
 * Return the latest UpdatedAt timestamp matching a condition.
 */
export async function maxUpdatedAt(model: SearchableForMaxUpdatedAt, cond: BaseQueryCondition | []): Promise<number> {
  try {
    const rows = await model.Search(cond, { fields: ['UpdatedAt'], orderBy: { field: 'UpdatedAt', order: 'desc' }, limit: 1 });
    const v = rows?.[0]?.UpdatedAt;
    const n = Number(new Date((v as string | number | Date | null | undefined) || 0));
    return Number.isFinite(n) ? n : 0;
  } catch {
    return 0;
  }
}

/**
 * Expand direct roles through the full inheritance closure.
 */
export async function expandRoleClosure(directRoleIds: string[]): Promise<string[]> {
  const seed = Array.from(new Set((directRoleIds || []).map(s => String(s || '').trim()).filter(Boolean)));
  if (seed.length === 0) return [];

  const all = new Set<string>(seed);
  const pending = seed.slice();

  const edges = await RoleInheritance.Search([], { fields: ['ParentRoleId', 'ChildRoleId'], limit: 10000 });
  const adj = new Map<string, string[]>();
  for (const e of edges || []) {
    const parentId = maybeId(e.ParentRoleId);
    const childId = maybeId(e.ChildRoleId);
    if (parentId && childId) {
      if (!adj.has(parentId)) adj.set(parentId, []);
      adj.get(parentId)!.push(childId);
    }
  }

  while (pending.length > 0) {
    const current = pending.shift()!;
    const children = adj.get(current) || [];
    for (const childId of children) {
      if (!all.has(childId)) {
        all.add(childId);
        pending.push(childId);
      }
    }
  }
  return Array.from(all);
}

/**
 * Compute effective global and company-scoped role coverage from user-role assignments.
 */
export async function computeEffectiveRoleScopes(
  userRoles: Array<Projected<UserRole, ['RoleId', 'CompanyId']>>
): Promise<Map<string, { global: boolean; companies: Set<string> }>> {
  type RoleScope = { global: boolean; companies: Set<string> };
  const roleScopes = new Map<string, RoleScope>();

  const ensureScope = (roleId: string): RoleScope => {
    let s = roleScopes.get(roleId);
    if (!s) {
      s = { global: false, companies: new Set<string>() };
      roleScopes.set(roleId, s);
    }
    return s;
  };

  const mergeScope = (roleId: string, companyId: string | null | undefined): boolean => {
    const s = ensureScope(roleId);
    if (companyId === null || companyId === undefined || String(companyId ?? '').trim() === '') {
      if (s.global) return false;
      s.global = true;
      return true;
    }
    if (s.global) return false;
    const cid = String(companyId).trim();
    if (!cid || s.companies.has(cid)) return false;
    s.companies.add(cid);
    return true;
  };

  for (const ur of userRoles || []) {
    const roleId = maybeId(ur.RoleId);
    if (!roleId) continue;
    // CompanyId is ManyToOneRef (string id) or empty for a global assignment.
    const companyRaw = ur.CompanyId;
    mergeScope(roleId, companyRaw == null || String(companyRaw).trim() === '' ? null : String(companyRaw).trim());
  }

  const directRoleIds = Array.from(new Set((userRoles || []).map(ur => maybeId(ur.RoleId)).filter(Boolean) as string[]));
  if (directRoleIds.length === 0) return roleScopes;

  const edges = await RoleInheritance.Search([], { fields: ['ParentRoleId', 'ChildRoleId'], limit: 10000 });
  const adj = new Map<string, string[]>();
  for (const e of edges || []) {
    const parentId = maybeId(e.ParentRoleId);
    const childId = maybeId(e.ChildRoleId);
    if (parentId && childId) {
      if (!adj.has(parentId)) adj.set(parentId, []);
      adj.get(parentId)!.push(childId);
    }
  }

  const pending: string[] = directRoleIds.slice();
  for (const rid of pending) ensureScope(rid);

  while (pending.length > 0) {
    const parentId = pending.shift()!;
    const parentScope = roleScopes.get(parentId);
    if (!parentScope) continue;

    const children = adj.get(parentId) || [];
    for (const childId of children) {
      ensureScope(childId);
      let changed = false;
      if (parentScope.global) {
        changed = mergeScope(childId, null) || changed;
      } else {
        for (const cid of parentScope.companies) {
          changed = mergeScope(childId, cid) || changed;
        }
      }
      if (changed) pending.push(childId);
    }
  }

  return roleScopes;
}

/**
 * Compute a permission-state version from user-role and role graph timestamps.
 */
export async function computePermStateVersion(userId: string): Promise<number> {
  const uid = String(userId || '').trim();
  if (!uid) return 0;
  try {
    return await withPermissionGraphBypass(async () => {
      const asSearchable = (model: object): SearchableForMaxUpdatedAt => model as SearchableForMaxUpdatedAt;

      const urMax = await maxUpdatedAt(asSearchable(UserRole), ['UserId', '=', uid]);

      const userRoles = await UserRole.Search(['UserId', '=', uid], { fields: ['RoleId'], limit: 5000 });
      const directRoleIds = Array.from(new Set((userRoles || []).map(ur => maybeId(ur.RoleId)).filter(Boolean) as string[]));

      const effectiveRoleIds = await expandRoleClosure(directRoleIds);
      if (effectiveRoleIds.length === 0) {
        // Role-less users still consume everyone (RoleId null) record rules.
        const everyoneRrMax = await maxUpdatedAt(asSearchable(RoleRecordRule), ['RoleId', 'is', null]);
        return Math.max(urMax, everyoneRrMax);
      }

      const [roleMax, inhMax, maMax, rrMax, rfMax, ruMax] = await Promise.all([
        maxUpdatedAt(asSearchable(Role), condition(['Id', 'in', effectiveRoleIds])),
        maxUpdatedAt(
          asSearchable(RoleInheritance),
          condition({
            Or: [
              ['ParentRoleId', 'in', effectiveRoleIds],
              ['ChildRoleId', 'in', effectiveRoleIds],
            ],
          })
        ),
        maxUpdatedAt(asSearchable(RoleMethodAccess), condition(['RoleId', 'in', effectiveRoleIds])),
        maxUpdatedAt(
          asSearchable(RoleRecordRule),
          condition({
            Or: [['RoleId', 'is', null], ['RoleId', 'in', effectiveRoleIds]],
          })
        ),
        maxUpdatedAt(asSearchable(RoleFieldRule), condition(['RoleId', 'in', effectiveRoleIds])),
        maxUpdatedAt(asSearchable(RoleUiResource), condition(['RoleId', 'in', effectiveRoleIds])),
      ]);

      return Math.max(urMax, roleMax, inhMax, maMax, rrMax, rfMax, ruMax);
    });
  } catch {
    return 0;
  }
}

export type AuthzContextResult = {
  userId: string;
  activeCompanyId: string;
  enabledCompanyIds: string[];
  roleScopesById: Record<string, { global: boolean; companies: string[] }>;
  roleIds: string[];
  rolesByCompany: Record<string, string[]>;
  roles: string[];
};

/**
 * Core authz context builder (stateless).  Callers are responsible for caching.
 */
export async function buildAuthzContext(args: { userId: string; activeCompanyId: string; enabledCompanyIds: string[] }): Promise<AuthzContextResult> {
  const { activeCompanyId, enabledCompanyIds } = args;

  if (!args.userId) {
    return {
      userId: '',
      activeCompanyId,
      enabledCompanyIds,
      roleScopesById: {},
      roleIds: [],
      rolesByCompany: {},
      roles: [],
    };
  }

  return await withPermissionGraphBypass(async () => {
    const hasCompany = enabledCompanyIds.length > 0;
    const userRoleCond = hasCompany
      ? condition({
          And: [
            ['UserId', '=', args.userId],
            {
              Or: [
                ['CompanyId', 'is', null],
                ['CompanyId', 'in', enabledCompanyIds],
              ],
            },
          ],
        })
      : condition({
          And: [
            ['UserId', '=', args.userId],
            ['CompanyId', 'is', null],
          ],
        });

    const userRoles = await UserRole.Search(userRoleCond, {
      fields: ['RoleId', 'CompanyId'],
      limit: 5000,
    });
    const roleScopes = await computeEffectiveRoleScopes(userRoles || []);
    const roleIds = Array.from(roleScopes.keys());

    const roleScopesById: Record<string, { global: boolean; companies: string[] }> = {};
    for (const [rid, scope] of roleScopes.entries()) {
      roleScopesById[rid] = {
        global: !!scope.global,
        companies: scope.global ? [] : sortStrings(Array.from(scope.companies)),
      };
    }

    const globalRoleIds: string[] = [];
    const scopedByCompany = new Map<string, Set<string>>();
    for (const [rid, scope] of roleScopes.entries()) {
      if (scope.global) {
        globalRoleIds.push(rid);
      } else {
        for (const cid of scope.companies) {
          if (!cid) continue;
          let s = scopedByCompany.get(cid);
          if (!s) {
            s = new Set<string>();
            scopedByCompany.set(cid, s);
          }
          s.add(rid);
        }
      }
    }

    const rolesByCompany: Record<string, string[]> = {};
    for (const cid of enabledCompanyIds) {
      const set = new Set<string>();
      for (const rid of globalRoleIds) set.add(rid);
      const extra = scopedByCompany.get(cid);
      if (extra) for (const rid of extra) set.add(rid);
      rolesByCompany[cid] = sortStrings(Array.from(set));
    }

    const activeRoles = activeCompanyId && rolesByCompany[activeCompanyId] ? rolesByCompany[activeCompanyId] : sortStrings(globalRoleIds);

    return {
      userId: args.userId,
      activeCompanyId,
      enabledCompanyIds,
      roleScopesById,
      roleIds: sortStrings(roleIds),
      rolesByCompany,
      roles: activeRoles,
    };
  });
}

/**
 * Build or reuse the request-scoped authorization context with caching.
 */
export async function getAuthzContext(
  userIdGetter: () => string,
  deps: {
    getCurrentReq: () => unknown;
    getOrInitReqServiceState: (req: unknown) => Record<string, unknown> | undefined;
    getCompanyScopeFromRequestContext: () => { activeCompanyId: string; enabledCompanyIds: string[] };
    buildAuthzContextCacheKey: (userId: string, companyScopeKey: string) => string;
    buildAuthzContext: typeof buildAuthzContext;
  }
): Promise<AuthzContextResult> {
  const req = deps.getCurrentReq();
  const state = deps.getOrInitReqServiceState(req);

  const userId = String(userIdGetter() || '').trim();
  const companyScope = deps.getCompanyScopeFromRequestContext();
  const enabledCompanyIdsKey = sortStrings(uniqStrings(companyScope.enabledCompanyIds));
  const companyScopeKey = `${companyScope.activeCompanyId}::${enabledCompanyIdsKey.join(',')}`;

  if (!state) {
    return await deps.buildAuthzContext({ userId, activeCompanyId: companyScope.activeCompanyId, enabledCompanyIds: enabledCompanyIdsKey });
  }

  const KEY = deps.buildAuthzContextCacheKey(userId, companyScopeKey);
  return await memoizeInReqState(state, KEY, () =>
    deps.buildAuthzContext({ userId, activeCompanyId: companyScope.activeCompanyId, enabledCompanyIds: enabledCompanyIdsKey })
  );
}
