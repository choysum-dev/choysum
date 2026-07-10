// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage, type ModelMetadata } from '../../metadata';
import type BaseModel from '../../model/model';
import type { ModelCtor } from '../../metadata/field';
import type { Entity } from '../types';
import { RepositoryFactory } from '../repository_factory';
import { AuthUserService, isAuthServiceUnavailable } from './auth_user_service';
import { getRepositoryCurrentReq, getRepositoryFieldRuleBypassDepth } from './authz_runtime';
import type { SelectionNode } from '../projection';
import type { RepositoryPermissionDeniedFn } from './types';
import { getRuntimeEnvFlag } from '@/core/utils/env';
import { asObjectRecord } from '../../../../utils/object';
import type { ObjectRecord } from '../../../../utils/types';

export type RepositoryFieldRuleSpec = {
  denyReadFields: string[];
  denyWriteFields: string[];
  reason?: string;
};

export type RepositoryFieldRuleDeps = {
  meta: ModelMetadata;
  userId?: string;
  requestContext: unknown;
  normalizeCompanyIds: () => string[];
  isControlPlaneMetaModel: () => boolean;
  isFieldRuleControlPlaneModel: () => boolean;
  withRecordRuleBypass: <T>(fn: () => Promise<T>) => Promise<T>;
  withFieldRuleBypass: <T>(fn: () => Promise<T>) => Promise<T>;
  permissionDenied: RepositoryPermissionDeniedFn;
};

export function repositoryFieldRuleEnabled(): boolean {
  return getRuntimeEnvFlag('CHOYSUM_GRPC_FIELD_RULE_ENABLED', true);
}

export function getRepositoryTopLevelFieldRuleMode(): string {
  const req = getRepositoryCurrentReq();
  const depth = typeof req?.depth === 'number' ? req.depth : 0;
  if (depth !== 0) return '';
  return typeof req?.fieldRuleMode === 'string' ? String(req.fieldRuleMode).trim() : '';
}

export function repositoryFieldRuleLayerSkipped(): boolean {
  return getRepositoryTopLevelFieldRuleMode() === 'skip';
}

function resolveRepositoryFieldRuleJsCtxRoot(): ObjectRecord | undefined {
  const root = asObjectRecord((globalThis as { $choysum?: unknown }).$choysum);
  if (!root) return undefined;
  const request = asObjectRecord(root.request);
  return asObjectRecord(request?.context) ?? asObjectRecord(root.context) ?? root;
}

function getRepositoryFieldRuleCache(): Map<string, RepositoryFieldRuleSpec> {
  const CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
  const jsCtx = resolveRepositoryFieldRuleJsCtxRoot();
  const jsCtxCarrier = jsCtx as Record<PropertyKey, unknown> | undefined;

  if (jsCtxCarrier) {
    if (!(jsCtxCarrier[CACHE_KEY] instanceof Map)) {
      jsCtxCarrier[CACHE_KEY] = new Map<string, RepositoryFieldRuleSpec>();
    }
    return jsCtxCarrier[CACHE_KEY] as Map<string, RepositoryFieldRuleSpec>;
  }

  const PROC_KEY = Symbol.for('choysum.fieldrule.cache.process');
  const globalCarrier = globalThis as unknown as Record<PropertyKey, unknown>;
  if (!(globalCarrier[PROC_KEY] instanceof Map)) {
    globalCarrier[PROC_KEY] = new Map<string, RepositoryFieldRuleSpec>();
  }
  return globalCarrier[PROC_KEY] as Map<string, RepositoryFieldRuleSpec>;
}

function buildRepositoryFieldRuleCacheKey(params: Pick<RepositoryFieldRuleDeps, 'meta' | 'userId' | 'normalizeCompanyIds'>): string {
  const model = (params.meta.fullModelName || params.meta.modelName || params.meta.name || 'Unknown') as string;
  const userId = String(params.userId || '').trim();
  const companyIds = params.normalizeCompanyIds().slice().sort().join(',');
  const req = getRepositoryCurrentReq();
  const method = typeof req?.method === 'string' ? req.method : '';
  const mode = typeof req?.fieldRuleMode === 'string' ? req.fieldRuleMode : '';
  const enabled = '1';

  return `fr|${model}|u=${userId}|c=${companyIds}|m=${method}|frm=${mode}|g=${enabled}`;
}

function normalizeRepositoryFieldRuleSpec(input: unknown): RepositoryFieldRuleSpec {
  const value = asObjectRecord(input) ?? {};

  const toStringArray = (raw: unknown): string[] => {
    if (!Array.isArray(raw)) return [];
    const output: string[] = [];
    for (const item of raw) {
      const legacyNormalized = String(item ?? '').trim();
      if (legacyNormalized) output.push(legacyNormalized);
    }
    return Array.from(new Set(output)).sort();
  };

  return {
    denyReadFields: toStringArray(value.denyReadFields),
    denyWriteFields: toStringArray(value.denyWriteFields),
    reason: typeof value.reason === 'string' ? value.reason : undefined,
  };
}

export async function getRepositoryFieldRuleSpec(params: RepositoryFieldRuleDeps): Promise<RepositoryFieldRuleSpec> {
  if (!repositoryFieldRuleEnabled()) {
    return { denyReadFields: [], denyWriteFields: [], reason: 'field_rule_disabled' };
  }

  if (params.isControlPlaneMetaModel()) {
    return { denyReadFields: [], denyWriteFields: [], reason: 'control_plane_meta_model' };
  }

  if (params.isFieldRuleControlPlaneModel()) {
    return { denyReadFields: [], denyWriteFields: [], reason: 'field_rule_control_plane_model' };
  }

  if (repositoryFieldRuleLayerSkipped()) {
    return { denyReadFields: [], denyWriteFields: [], reason: 'entry_field_rule_skip' };
  }

  const cache = getRepositoryFieldRuleCache();
  const key = buildRepositoryFieldRuleCacheKey(params);
  const cached = cache.get(key);
  if (cached) return cached;

  const model = (params.meta.fullModelName || params.meta.modelName || params.meta.name || 'Unknown') as string;
  let result: unknown;
  try {
    result = await params.withRecordRuleBypass(async () => params.withFieldRuleBypass(async () => AuthUserService.GetFieldRuleSpec(model)));
  } catch (error) {
    if (isAuthServiceUnavailable(error)) {
      const spec = { denyReadFields: [], denyWriteFields: [], reason: 'auth_service_unavailable' };
      cache.set(key, spec);
      return spec;
    }
    throw params.permissionDenied('field_rule_fetch_failed', 'failed to fetch field rule spec', { model });
  }

  const spec = normalizeRepositoryFieldRuleSpec(result);
  cache.set(key, spec);
  return spec;
}

export async function assertRepositoryFieldRuleWriteAllowed(params: RepositoryFieldRuleDeps & { payload: Entity }): Promise<void> {
  if (!repositoryFieldRuleEnabled()) return;
  if (params.isControlPlaneMetaModel()) return;
  if (params.isFieldRuleControlPlaneModel()) return;
  if (getRepositoryFieldRuleBypassDepth() > 0) return;

  const { payload } = params;
  if (!payload || typeof payload !== 'object') return;
  const proto = Object.getPrototypeOf(payload);
  const isPlain = proto === Object.prototype || proto === null;
  if (!isPlain) return;

  const req = getRepositoryCurrentReq();
  const spec = await getRepositoryFieldRuleSpec(params);
  if (!spec.denyWriteFields.length) return;

  const deniedFields: string[] = [];
  for (const field of spec.denyWriteFields) {
    if (Object.prototype.hasOwnProperty.call(payload, field)) deniedFields.push(field);
  }
  if (!deniedFields.length) return;

  const method = typeof req?.method === 'string' ? req.method : '';
  const model = (params.meta.fullModelName || params.meta.modelName || params.meta.name || 'Unknown') as string;
  throw params.permissionDenied('field_rule_readonly_violation', 'field rule readonly violation', {
    model,
    access: 'write',
    field: deniedFields[0],
    fields: deniedFields.join(','),
    method,
    reason: spec.reason || 'denied',
  });
}

export type RepositoryFieldRuleSelectionDeps = Pick<RepositoryFieldRuleDeps, 'isControlPlaneMetaModel'>;

export async function pruneRepositorySelectionTreeForFieldRule(
  params: RepositoryFieldRuleSelectionDeps,
  meta: ModelMetadata,
  node: SelectionNode,
  denyCache: Map<unknown, string[]>
): Promise<void> {
  if (!node) return;
  if (!repositoryFieldRuleEnabled()) return;
  if (params.isControlPlaneMetaModel()) return;

  const getDeny = async (ctor: unknown): Promise<string[]> => {
    if (!ctor || typeof ctor !== 'function') return [];
    const cached = denyCache.get(ctor);
    if (cached !== undefined) return cached;
    try {
      const repo = RepositoryFactory.getRepository(ctor as never);
      const spec = await repo.getDenyReadFields();
      const denyRaw = asObjectRecord(spec)?.denyReadFields;
      const deny = Array.isArray(denyRaw) ? denyRaw.map(item => String(item ?? '').trim()).filter(Boolean) : [];
      denyCache.set(ctor, deny);
      return deny;
    } catch {
      denyCache.set(ctor, []);
      return [];
    }
  };

  const ctor = meta?.type;
  const deny = await getDeny(ctor);
  if (deny.length) {
    for (const field of deny) {
      const key = String(field ?? '').trim();
      if (!key) continue;
      if (key !== 'Id') node.columns.delete(key);
      node.relations.delete(key);
    }
  }

  for (const [, entry] of Array.from(node.relations.entries())) {
    const targetModelFn = entry.relation?.targetModel;
    if (typeof targetModelFn !== 'function') continue;
    let targetCtor: unknown;
    try {
      targetCtor = targetModelFn();
    } catch {
      targetCtor = undefined;
    }
    if (typeof targetCtor !== 'function') continue;

    const targetMeta = MetadataStorage.instance.getModelMetadata(targetCtor as ModelCtor<BaseModel>);
    await pruneRepositorySelectionTreeForFieldRule(params, targetMeta, entry.node, denyCache);

    if (!entry.node.columns.size) entry.node.columns.add('Id');
  }
}
