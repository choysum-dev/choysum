// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
import type MetaModelModel from '@/meta/service/models/model';
import type MetaServiceModel from '@/meta/service/models/service';
import RoleMethodAccess from './role_method_access';
import { maybeId } from './_user_authz_shared';
import { normalizeLogicalMethods } from './_logical_model_registry';

const MetaService = createServiceByModel<typeof MetaServiceModel>('meta.MetaService');
const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');
const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');

type ServiceAgg = {
  allow: Set<string>;
  deny: Set<string>;
  allowAll: boolean;
  denyAll: boolean;
};

export type AclAggregationResult = {
  requiresAllowKeysByCompany: Map<string, Set<string>>;
  requiresDenyKeysByCompany: Map<string, Set<string>>;
  companyGlobalAllow: Set<string>;
  companyGlobalDeny: Set<string>;
};

/**
 * Map a role id to its scope keys ('*' or per-company).
 */
export function applyToScope(roleId: string, roleScopesById: Record<string, { global: boolean; companies: string[] }>, fn: (companyKey: string) => void): void {
  const scope = roleScopesById?.[roleId];
  if (!scope) return;
  if (scope.global) fn('*');
  else for (const cid of scope.companies || []) fn(cid);
}

/**
 * Aggregate RoleMethodAccess entries into per-company RPC allow/deny sets and
 * synthesize requires keys for downstream UI whitelist evaluation.
 */
export async function buildAclAggregation(
  roleIds: string[],
  roleScopesById: Record<string, { global: boolean; companies: string[] }>
): Promise<AclAggregationResult> {
  const accessesRaw = await RoleMethodAccess.Search(['RoleId', 'in', roleIds] as any, {
    fields: ['RoleId', 'MetaServiceId', 'MetaModelId', 'MetaApplicationId', 'LogicalModelName', 'LogicalMethods', 'Mode', 'Source'],
    limit: 50000,
  });
  // UI-Option-A: ignore legacy Source=ui rows in PermissionState ACL aggregation.
  const accesses = (accessesRaw || []).filter(a => String((a as any).Source || 'manual').toLowerCase() !== 'ui');

  const aggByCompany = new Map<string, Map<string, ServiceAgg>>();
  const companyGlobalAllow = new Set<string>();
  const companyGlobalDeny = new Set<string>();

  const ensureAgg = (companyKey: string, serviceFullName: string): ServiceAgg => {
    let m = aggByCompany.get(companyKey);
    if (!m) {
      m = new Map();
      aggByCompany.set(companyKey, m);
    }
    let a = m.get(serviceFullName);
    if (!a) {
      a = { allow: new Set(), deny: new Set(), allowAll: false, denyAll: false };
      m.set(serviceFullName, a);
    }
    return a;
  };

  const irServiceIds = Array.from(new Set(accesses.map(a => String((a as any).MetaServiceId || '').trim()).filter(Boolean)));
  const irModelIds = Array.from(new Set(accesses.map(a => String((a as any).MetaModelId || '').trim()).filter(Boolean)));
  const irApplicationIds = Array.from(new Set(accesses.map(a => String((a as any).MetaApplicationId || '').trim()).filter(Boolean)));

  const serviceById = new Map<string, { modelId: string; name: string }>();
  const modelById = new Map<string, { app: string; name: string }>();

  // 4.1) Resolve service -> model + method & 4.3) Resolve applicationId -> applicationName
  const [services, apps] = await Promise.all([
    irServiceIds.length > 0 ? MetaService.Search(['Id', 'in', irServiceIds] as any, { fields: ['Id', 'ModelId', 'Name'], limit: 50000 }) : Promise.resolve([]),
    irApplicationIds.length > 0
      ? MetaApplication.Search(['Id', 'in', irApplicationIds] as any, { fields: ['Id', 'Name'], limit: 50000 } as any)
      : Promise.resolve([]),
  ]);

  const modelIdsFromServices = new Set<string>();
  for (const s of services || []) {
    const sid = String((s as any).Id || '').trim();
    const mid = maybeId((s as any).ModelId);
    const name = String((s as any).Name || '').trim();
    if (!sid || !mid || !name) continue;
    serviceById.set(sid, { modelId: mid, name });
    modelIdsFromServices.add(mid);
  }

  const appNameById = new Map<string, string>();
  for (const a of apps || []) {
    const id = String((a as any).Id || '').trim();
    const name = String((a as any).Name || '').trim();
    if (!id || !name) continue;
    appNameById.set(id, name);
  }

  // 4.2) Resolve model -> app + name
  const needModelIds = Array.from(new Set([...irModelIds, ...Array.from(modelIdsFromServices)]));
  if (needModelIds.length > 0) {
    const models = await MetaModel.Search(['Id', 'in', needModelIds] as any, { fields: ['Id', 'Name', 'Application'], limit: 50000 });
    for (const m of models || []) {
      const mid = String((m as any).Id || '').trim();
      const app = String((m as any).Application || '').trim();
      const name = String((m as any).Name || '').trim();
      if (!mid || !app || !name) continue;
      modelById.set(mid, { app, name });
    }
  }

  // Effective projections: at most one live row per (application, name).
  // Deduplicate by name in case legacy shells still coexist.
  const trimMetaLabel = (v: any): string => {
    if (v == null) return '';
    return String(v).trim();
  };
  const modelsByApp = new Map<string, Array<{ app: string; name: string }>>();
  const appNames = Array.from(new Set(appNameById.values()));
  if (appNames.length > 0) {
    const rows = await MetaModel.Search(['Application', 'in', appNames] as any, {
      fields: ['Application', 'Name', 'ModuleId', 'UpdatedAt'],
      orderBy: { field: 'UpdatedAt', order: 'desc' },
      limit: 50000,
    } as any);
    const seen = new Set<string>();
    const list = rows == null ? [] : rows;
    for (const r of list) {
      const app = trimMetaLabel((r as any).Application);
      const name = trimMetaLabel((r as any).Name);
      if (!app) continue;
      if (!name) continue;
      const key = `${app}\0${name}`;
      if (seen.has(key)) continue;
      seen.add(key);
      if (!modelsByApp.has(app)) modelsByApp.set(app, []);
      modelsByApp.get(app)!.push({ app, name });
    }
  }

  const getModelsForApp = (appName: string): Array<{ app: string; name: string }> => {
    const k = String(appName || '').trim();
    return modelsByApp.get(k) || [];
  };

  let allModels: Array<{ app: string; name: string }> | undefined;
  const getAllModels = async (): Promise<Array<{ app: string; name: string }>> => {
    if (allModels) return allModels;
    const rows = await MetaModel.Search([] as any, {
      fields: ['Application', 'Name', 'UpdatedAt'],
      orderBy: { field: 'UpdatedAt', order: 'desc' },
      limit: 50000,
    } as any);
    const seen = new Set<string>();
    const out: Array<{ app: string; name: string }> = [];
    const list = rows == null ? [] : rows;
    for (const r of list) {
      const app = trimMetaLabel((r as any).Application);
      const name = trimMetaLabel((r as any).Name);
      if (!app) continue;
      if (!name) continue;
      const key = `${app}\0${name}`;
      if (seen.has(key)) continue;
      seen.add(key);
      out.push({ app, name });
    }
    allModels = out;
    return out;
  };

  const scope = (rid: string) => applyToScope.bind(null, rid, roleScopesById);

  // 4.4) Apply rules into per-company aggregates
  for (const a of accesses || []) {
    const roleId = maybeId((a as any).RoleId);
    const sid = String((a as any).MetaServiceId || '').trim();
    const mid = String((a as any).MetaModelId || '').trim();
    const aid = String((a as any).MetaApplicationId || '').trim();
    const logicalName = String((a as any).LogicalModelName || '').trim();
    const mode = String((a as any).Mode || '').toLowerCase();
    if (!roleId || (mode !== 'allow' && mode !== 'deny')) continue;

    // LogicalModel scope: all host apps whose @Model short name matches.
    if (!sid && !mid && !aid && logicalName) {
      const models = (await getAllModels()).filter(m => m.name === logicalName);
      if (models.length === 0) continue;
      let methods: string[] | null;
      try {
        methods = normalizeLogicalMethods((a as any).LogicalMethods);
      } catch {
        continue;
      }
      scope(roleId)(companyKey => {
        for (const m of models) {
          const serviceFullName = `${m.app}.${m.name}`;
          const agg = ensureAgg(companyKey, serviceFullName);
          if (methods == null) {
            if (mode === 'allow') agg.allowAll = true;
            else agg.denyAll = true;
          } else {
            for (const methodName of methods) {
              if (mode === 'allow') agg.allow.add(methodName);
              else agg.deny.add(methodName);
            }
          }
        }
      });
      continue;
    }

    // global scope (Meta* and LogicalModelName all empty)
    if (!sid && !mid && !aid && !logicalName) {
      const models = await getAllModels();
      scope(roleId)(companyKey => {
        if (mode === 'allow') companyGlobalAllow.add(companyKey);
        else companyGlobalDeny.add(companyKey);
        for (const m of models) {
          const serviceFullName = `${m.app}.${m.name}`;
          const agg = ensureAgg(companyKey, serviceFullName);
          if (mode === 'allow') agg.allowAll = true;
          else agg.denyAll = true;
        }
      });
      continue;
    }

    // application scope
    if (!sid && !mid && aid && !logicalName) {
      const appName = appNameById.get(aid);
      if (!appName) continue;
      const models = getModelsForApp(appName);
      scope(roleId)(companyKey => {
        for (const m of models) {
          const serviceFullName = `${m.app}.${m.name}`;
          const agg = ensureAgg(companyKey, serviceFullName);
          if (mode === 'allow') agg.allowAll = true;
          else agg.denyAll = true;
        }
      });
      continue;
    }

    // model scope
    if (!sid && mid && !aid && !logicalName) {
      const mdl = modelById.get(mid);
      if (!mdl) continue;
      const serviceFullName = `${mdl.app}.${mdl.name}`;
      scope(roleId)(companyKey => {
        const agg = ensureAgg(companyKey, serviceFullName);
        if (mode === 'allow') agg.allowAll = true;
        else agg.denyAll = true;
      });
      continue;
    }

    // service(method) scope
    if (sid && !mid && !aid && !logicalName) {
      const svc = serviceById.get(sid);
      if (!svc) continue;
      const mdl = modelById.get(svc.modelId);
      if (!mdl) continue;
      const serviceFullName = `${mdl.app}.${mdl.name}`;
      const methodName = svc.name;
      scope(roleId)(companyKey => {
        const agg = ensureAgg(companyKey, serviceFullName);
        if (mode === 'allow') agg.allow.add(methodName);
        else agg.deny.add(methodName);
      });
      continue;
    }
  }

  // 2.5) Synthesize requires keys per company for internal evaluation only.
  const requiresAllowKeysByCompany = new Map<string, Set<string>>();
  const requiresDenyKeysByCompany = new Map<string, Set<string>>();
  for (const [companyKey, svcMap] of aggByCompany.entries()) {
    if (!requiresAllowKeysByCompany.has(companyKey)) requiresAllowKeysByCompany.set(companyKey, new Set<string>());
    if (!requiresDenyKeysByCompany.has(companyKey)) requiresDenyKeysByCompany.set(companyKey, new Set<string>());
    const requiresAllowSet = requiresAllowKeysByCompany.get(companyKey)!;
    const requiresDenySet = requiresDenyKeysByCompany.get(companyKey)!;

    for (const [serviceFullName, agg] of svcMap.entries()) {
      const serviceWildcard = `rpc:/${serviceFullName}/*`;

      if (agg.denyAll) {
        requiresDenySet.add(serviceWildcard);
        continue;
      }

      if (agg.allowAll) {
        requiresAllowSet.add(serviceWildcard);
      } else {
        for (const m of agg.allow) {
          requiresAllowSet.add(`rpc:/${serviceFullName}/${m}`);
        }
      }

      if (agg.deny.size > 0) {
        for (const m of agg.deny) requiresDenySet.add(`rpc:/${serviceFullName}/${m}`);
      }
    }
  }

  return {
    requiresAllowKeysByCompany,
    requiresDenyKeysByCompany,
    companyGlobalAllow,
    companyGlobalDeny,
  };
}
