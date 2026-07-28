// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getCurrentReq, getOrInitReqServiceState, memoizeInReqState } from '@/core/service/api/context';
import { createServiceByModel } from '@/core/service/rpc';
import type IrApplicationModel from '@/meta/service/models/ir_application';
import type IrModelModel from '@/meta/service/models/ir_model';
import type IrServiceModel from '@/meta/service/models/ir_service';
import IrUiResource from '@/meta/service/models/ir_ui_resource';
import { uniqStrings } from '@/core/service/utils/normalization';
import { buildUiGrantCacheKey } from './_request_cache_invalidation';
import RoleMethodAccess from './role_method_access';
import RoleUiResource from './role_ui_resource';
import { normalizeScopeRefId, normalizeUiResourceId, parseJsonStringArray, requireMatchesMethod, sortStrings } from './_user_authz_shared';

const IrApplication = createServiceByModel<typeof IrApplicationModel>('meta.IrApplication');
const IrModel = createServiceByModel<typeof IrModelModel>('meta.IrModel');
const IrService = createServiceByModel<typeof IrServiceModel>('meta.IrService');

export type UiGrantExpansion = {
  resources: any[];
  hasGlobalAllow: boolean;
  hasGlobalDeny: boolean;
  appModesById: Record<string, Array<'allow' | 'deny'>>;
  resourceModesByKey: Record<string, Array<'allow' | 'deny'>>;
};

/**
 * Method ACL decision envelope shared by CheckMethodAccess and UI-derived eval (PR-E-5 / W10).
 */
export type MethodAccessDecision = {
  allowed: boolean;
  reason: string;
  hitRuleIds: string[];
};

function getMethodAccessReqState(): Record<string, unknown> | undefined {
  const req = getCurrentReq();
  return req ? getOrInitReqServiceState(req) : undefined;
}

function buildMethodAccessMetaCacheKey(appName: string, modelName: string, methodName: string): string {
  return `methodAccessMeta::${String(appName || '').trim()}::${String(modelName || '').trim()}::${String(methodName || '')
    .trim()
    .toLowerCase()}`;
}

/**
 * Resolve model/service/application metadata needed for method-access checks.
 */
export async function resolveMethodAccessMeta(
  appName: string,
  modelName: string,
  methodName: string
): Promise<
  | {
      modelId: string;
      irServiceId: string;
      irApplicationId: string;
      scopeOr: any[];
      modelKey: string;
      methodLower: string;
    }
  | undefined
> {
  const state = getMethodAccessReqState();
  const key = buildMethodAccessMetaCacheKey(appName, modelName, methodName);
  return await memoizeInReqState(state, key, async () => {
    const models = await IrModel.Search(
      {
        And: [
          ['Name', '=', modelName],
          ['Application', '=', appName],
        ],
      } as any,
      { fields: ['Id'], limit: 1 }
    );
    const modelId = String(models?.[0]?.Id || '').trim();
    if (!modelId) return undefined;

    const serviceRows = await IrService.Search({ And: [['ModelId', '=', modelId]] } as any, { fields: ['Id', 'Name'], limit: 5000 } as any);
    const methodLower = String(methodName || '')
      .trim()
      .toLowerCase();
    const matched = (serviceRows || []).find(
      (r: any) =>
        String((r as any).Name || '')
          .trim()
          .toLowerCase() === methodLower
    ) as any;
    const irServiceId = String(matched?.Id || '').trim();
    if (!irServiceId) return undefined;

    const apps = await IrApplication.Search({ And: [['Name', '=', appName]] } as any, { fields: ['Id'], limit: 1 } as any);
    const irApplicationId = String((apps?.[0] as any)?.Id || '').trim();

    const scopeOr: any[] = [
      {
        And: [
          ['IrServiceId', '=', irServiceId],
          ['IrModelId', 'is', null],
          ['IrApplicationId', 'is', null],
        ],
      },
      {
        And: [
          ['IrServiceId', 'is', null],
          ['IrModelId', '=', modelId],
          ['IrApplicationId', 'is', null],
        ],
      },
      {
        And: [
          ['IrServiceId', 'is', null],
          ['IrModelId', 'is', null],
          ['IrApplicationId', 'is', null],
        ],
      },
    ];

    if (irApplicationId) {
      scopeOr.splice(2, 0, {
        And: [
          ['IrServiceId', 'is', null],
          ['IrModelId', 'is', null],
          ['IrApplicationId', '=', irApplicationId],
        ],
      });
    }

    return {
      modelId,
      irServiceId,
      irApplicationId,
      scopeOr,
      modelKey: `${appName}.${modelName}`,
      methodLower,
    };
  });
}

/**
 * Evaluate explicit RoleMethodAccess rules with deny-wins semantics.
 */
export async function evaluateRoleMethodAccess(
  roleIds: string[],
  scopeOr: any[]
): Promise<{ denied: boolean; allowed: boolean; hitRuleIds: string[]; reason: string }> {
  const accessesRaw = await RoleMethodAccess.Search(
    {
      And: [['RoleId', 'in', roleIds], { Or: scopeOr } as any],
    } as any,
    { fields: ['Id', 'Mode', 'Source'], limit: 5000 }
  );
  // UI-Option-A: Source=ui rows are not manual ACL (runtime ui-derived path owns UI→Method).
  const accesses = (accessesRaw || []).filter(a => String((a as any).Source || 'manual').toLowerCase() !== 'ui');

  let allowed = false;
  const allowHitRuleIds: string[] = [];
  for (const a of accesses || []) {
    const mode = String((a as any).Mode || '').toLowerCase();
    const ruleId = String((a as any)?.Id ?? '').trim();
    if (mode === 'deny') {
      // Deny wins: only the deciding deny rule participates in diagnostics.
      const hitRuleIds = ruleId ? [ruleId] : [];
      return { denied: true, allowed: false, hitRuleIds, reason: 'method_access_deny' };
    }
    if (mode === 'allow') {
      allowed = true;
      if (ruleId) allowHitRuleIds.push(ruleId);
    }
  }

  const uniqueHitRuleIds = Array.from(new Set(allowHitRuleIds)).sort();
  if (allowed) return { denied: false, allowed: true, hitRuleIds: uniqueHitRuleIds, reason: 'method_access_allow' };
  return { denied: false, allowed: false, hitRuleIds: uniqueHitRuleIds, reason: 'method_access_no_manual_rule' };
}

/**
 * Expand role UI grants into materialized resource rows and lookup tables.
 */
export async function loadUiGrantExpansionForRoles(roleIds: string[]): Promise<UiGrantExpansion> {
  const ids = sortStrings(uniqStrings(roleIds || []));
  if (ids.length === 0) {
    return { resources: [], hasGlobalAllow: false, hasGlobalDeny: false, appModesById: {}, resourceModesByKey: {} };
  }

  const req = getCurrentReq();
  const state = getOrInitReqServiceState(req);
  const roleSig = ids.join(',');
  const cacheKey = buildUiGrantCacheKey(roleSig);
  const cached = state?.[cacheKey];
  if (cached && Array.isArray((cached as any).resources)) {
    return cached as UiGrantExpansion;
  }

  const grants = await RoleUiResource.Search(
    {
      And: [['RoleId', 'in', ids]],
    } as any,
    { fields: ['IrApplicationId', 'IrUiResourceId', 'Mode'], limit: 100000 } as any
  );

  let hasGlobalAllow = false;
  let hasGlobalDeny = false;
  const appIDs = new Set<string>();
  const resourceIDs = new Set<string>();
  const appModesById = new Map<string, Set<'allow' | 'deny'>>();
  const resourceModesByKey = new Map<string, Set<'allow' | 'deny'>>();
  const sortGrantModes = (modes: Iterable<'allow' | 'deny'>): Array<'allow' | 'deny'> =>
    Array.from(new Set(modes)).sort((a, b) => a.localeCompare(b)) as Array<'allow' | 'deny'>;

  for (const g of grants || []) {
    const mode =
      String((g as any).Mode ?? 'allow')
        .trim()
        .toLowerCase() === 'deny'
        ? 'deny'
        : 'allow';
    const appID = normalizeScopeRefId((g as any).IrApplicationId);
    const uiID = normalizeUiResourceId((g as any).IrUiResourceId);
    if (!appID && !uiID) {
      if (mode === 'deny') hasGlobalDeny = true;
      else hasGlobalAllow = true;
      continue;
    }

    if (appID) {
      appIDs.add(appID);
      if (!appModesById.has(appID)) appModesById.set(appID, new Set());
      appModesById.get(appID)!.add(mode);
    }
    if (uiID) {
      resourceIDs.add(uiID);
      if (!resourceModesByKey.has(uiID)) resourceModesByKey.set(uiID, new Set());
      resourceModesByKey.get(uiID)!.add(mode);
    }
  }

  if (!hasGlobalAllow && !hasGlobalDeny && appIDs.size === 0 && resourceIDs.size === 0) {
    const empty: UiGrantExpansion = {
      resources: [],
      hasGlobalAllow: false,
      hasGlobalDeny: false,
      appModesById: {},
      resourceModesByKey: {},
    };
    if (state) state[cacheKey] = empty;
    return empty;
  }

  const byId = new Map<string, any>();
  const mergeRows = (rows: any[]) => {
    for (const row of rows || []) {
      const id = normalizeUiResourceId((row as any)?.Id ?? (row as any)?.id);
      if (!id) continue;
      if (!byId.has(id)) byId.set(id, row);
    }
  };

  if (hasGlobalAllow || hasGlobalDeny) {
    const allRows = await IrUiResource.Search([] as any, { fields: ['Id', 'Name', 'IrApplicationId', 'Requires'], limit: 100000 } as any);
    mergeRows(allRows as any[]);
  } else {
    const appIDList = uniqStrings(Array.from(appIDs));
    const resourceIDList = uniqStrings(Array.from(resourceIDs));
    const promises: Array<Promise<any[]>> = [];

    if (appIDList.length > 0) {
      promises.push(
        IrUiResource.Search(
          { And: [['IrApplicationId', 'in', appIDList]] } as any,
          { fields: ['Id', 'Name', 'IrApplicationId', 'Requires'], limit: 100000 } as any
        )
      );
    }

    if (resourceIDList.length > 0) {
      promises.push(
        IrUiResource.Search(
          {
            Or: [
              ['Id', 'in', resourceIDList],
              ['Name', 'in', resourceIDList],
            ],
          } as any,
          { fields: ['Id', 'Name', 'IrApplicationId', 'Requires'], limit: 100000 } as any
        )
      );
    }

    const results = await Promise.all(promises);
    for (const rows of results) {
      mergeRows(rows as any[]);
    }
  }

  const resources = Array.from(byId.values());
  const appModesRecord: Record<string, Array<'allow' | 'deny'>> = {};
  for (const [key, value] of appModesById.entries()) {
    appModesRecord[key] = sortGrantModes(value);
  }

  const resourceModesRecord: Record<string, Array<'allow' | 'deny'>> = {};
  for (const [key, value] of resourceModesByKey.entries()) {
    resourceModesRecord[key] = sortGrantModes(value);
  }

  const out: UiGrantExpansion = {
    resources,
    hasGlobalAllow,
    hasGlobalDeny,
    appModesById: appModesRecord,
    resourceModesByKey: resourceModesRecord,
  };
  if (state) state[cacheKey] = out;
  return out;
}

/**
 * Evaluate allow and deny decisions derived from UI resource requires.
 */
export async function evaluateUiDerivedMethodDecision(
  roleIds: string[],
  modelKey: string,
  methodLower: string
): Promise<{ allowed: boolean; denied: boolean; hitRuleIds: string[]; reason: string }> {
  const mkey = String(modelKey || '').trim();
  const m = String(methodLower || '')
    .trim()
    .toLowerCase();
  if (!mkey || !m) {
    return { allowed: false, denied: false, hitRuleIds: [], reason: 'method_access_ui_no_match' };
  }

  const expansion = await loadUiGrantExpansionForRoles(roleIds);
  const resources = expansion.resources || [];
  if (resources.length === 0) {
    return { allowed: false, denied: false, hitRuleIds: [], reason: 'method_access_ui_no_match' };
  }

  let allowed = false;
  let denied = false;
  const allowHitRuleIds: string[] = [];
  const denyHitRuleIds: string[] = [];

  for (const row of resources) {
    const requires = parseJsonStringArray((row as any)?.Requires ?? (row as any)?.requires);
    if (requires.length === 0) continue;

    let matchesMethod = false;
    for (const req of requires) {
      if (requireMatchesMethod(req, mkey, m)) {
        matchesMethod = true;
        break;
      }
    }
    if (!matchesMethod) continue;

    const matchedModes = new Set<'allow' | 'deny'>();
    if (expansion.hasGlobalAllow) matchedModes.add('allow');
    if (expansion.hasGlobalDeny) matchedModes.add('deny');

    const appId = normalizeScopeRefId((row as any)?.IrApplicationId);
    for (const mode of expansion.appModesById?.[appId || ''] || []) matchedModes.add(mode);

    const resourceKeys = uniqStrings([
      normalizeUiResourceId((row as any)?.Id ?? (row as any)?.id),
      String((row as any)?.Name ?? (row as any)?.name ?? '').trim(),
    ]);
    for (const key of resourceKeys) {
      for (const mode of expansion.resourceModesByKey?.[key] || []) matchedModes.add(mode);
    }

    // Rows without Id/id are dropped by loadUiGrantExpansionForRoles; Id or id is enough here.
    const resourceId = normalizeUiResourceId((row as any)?.Id ?? (row as any)?.id);
    if (matchedModes.has('deny')) {
      denied = true;
      if (resourceId) denyHitRuleIds.push(resourceId);
      continue;
    }
    if (matchedModes.has('allow')) {
      allowed = true;
      if (resourceId) allowHitRuleIds.push(resourceId);
    }
  }

  if (denied) {
    return {
      allowed: false,
      denied: true,
      hitRuleIds: Array.from(new Set(denyHitRuleIds)).sort(),
      reason: 'method_access_ui_deny',
    };
  }
  if (allowed) {
    return {
      allowed: true,
      denied: false,
      hitRuleIds: Array.from(new Set(allowHitRuleIds)).sort(),
      reason: 'method_access_ui_allow',
    };
  }
  return { allowed: false, denied: false, hitRuleIds: [], reason: 'method_access_ui_no_match' };
}
