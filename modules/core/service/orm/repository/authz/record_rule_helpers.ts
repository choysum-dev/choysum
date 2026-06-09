// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import { AuthUserService, isAuthServiceUnavailable } from './auth_user_service';
import { getRepositoryCurrentReq } from './authz_runtime';
import type { RepositoryPermissionDeniedFn } from './types';
import type { BaseQueryCondition, ConditionEnvelope, RecordRuleOp } from '../types';
import { asObjectRecord, isObjectRecord } from '../../../../utils/object';
import type { UnknownRecord } from '../../../../utils/types';

export type RepositoryRecordRuleDeps = {
  meta: ModelMetadata;
  userId?: string;
  requestContext: unknown;
  normalizeCompanyIds: () => string[];
  normalizeCompanyIdForWrite: () => string | undefined;
  isControlPlaneMetaModel: () => boolean;
  recordRuleEnabled: () => boolean;
  getRecordRuleBypassDepth: () => number;
  withRecordRuleBypass: <T>(fn: () => Promise<T>) => Promise<T>;
  permissionDenied: RepositoryPermissionDeniedFn;
};

function getRepositoryRecordRuleCache(): Map<string, ConditionEnvelope> {
  const CACHE_KEY = Symbol.for('choysum.recordrule.cache');
  const root = asObjectRecord(globalThis)?.$choysum;
  const rootRecord = asObjectRecord(root);
  const request = asObjectRecord(rootRecord?.request);
  const jsCtx = asObjectRecord(request?.context) ?? asObjectRecord(rootRecord?.context) ?? rootRecord;

  if (jsCtx) {
    const cached = jsCtx[CACHE_KEY as unknown as keyof UnknownRecord];
    if (cached instanceof Map) return cached as Map<string, ConditionEnvelope>;
    const created = new Map<string, ConditionEnvelope>();
    jsCtx[CACHE_KEY as unknown as keyof UnknownRecord] = created;
    return created;
  }

  const PROC_KEY = Symbol.for('choysum.recordrule.cache.process');
  const globalRecord = asObjectRecord(globalThis) as Record<PropertyKey, unknown>;
  const existing = globalRecord[PROC_KEY];
  if (existing instanceof Map) return existing as Map<string, ConditionEnvelope>;
  const created = new Map<string, ConditionEnvelope>();
  globalRecord[PROC_KEY] = created;
  return created;
}

function buildRepositoryRecordRuleCacheKey(
  params: Pick<RepositoryRecordRuleDeps, 'meta' | 'userId' | 'requestContext' | 'normalizeCompanyIds'>,
  op: RecordRuleOp
): string {
  const model = (params.meta.fullModelName || params.meta.modelName || params.meta.name || 'Unknown') as string;
  const userId = String(params.userId || '').trim();
  const companyIds = params.normalizeCompanyIds().slice().sort().join(',');
  const ctx = asObjectRecord(params.requestContext);
  const activeCompanyId = String(ctx?.activeCompanyId ?? ctx?.ActiveCompanyId ?? '').trim();
  const req = getRepositoryCurrentReq();
  const method = typeof req?.method === 'string' ? req.method : '';
  const mode = typeof req?.recordRuleMode === 'string' ? req.recordRuleMode : '';

  return `rr|${model}|${op}|u=${userId}|c=${companyIds}|ac=${activeCompanyId}|m=${method}|rrm=${mode}`;
}

function getRepositoryTopLevelRecordRuleAllowlist(): { enabled: boolean; allow: Set<string>; method: string } {
  const req = getRepositoryCurrentReq();
  const depth = typeof req?.depth === 'number' ? req.depth : 0;
  const method = typeof req?.method === 'string' ? req.method : '';
  const mode = typeof req?.recordRuleMode === 'string' ? req.recordRuleMode : '';

  if (depth !== 0) return { enabled: false, allow: new Set<string>(), method };
  if (mode !== 'allowlist') return { enabled: false, allow: new Set<string>(), method };

  const raw = req?.recordRuleAllow;
  const allow = new Set<string>();
  if (Array.isArray(raw)) {
    for (const item of raw) {
      const normalized = String(item ?? '').trim();
      if (normalized) allow.add(normalized);
    }
  }
  return { enabled: true, allow, method };
}

function normalizeRepositoryRecordRuleEnvelope(input: unknown): ConditionEnvelope {
  const value = asObjectRecord(input) ?? {};
  const kind = String(value.kind || '').trim();
  if (kind === 'true') return { kind: 'true', reason: typeof value.reason === 'string' ? value.reason : undefined };
  if (kind === 'false') return { kind: 'false', reason: typeof value.reason === 'string' ? value.reason : undefined };
  if (kind === 'expr') {
    if (!value.expr) return { kind: 'false', reason: 'invalid_record_rule_envelope_missing_expr' };
    return { kind: 'expr', expr: value.expr as BaseQueryCondition, reason: typeof value.reason === 'string' ? value.reason : undefined };
  }
  return { kind: 'false', reason: 'invalid_record_rule_envelope_kind' };
}

export async function fetchRepositoryRecordRuleEnvelope(params: RepositoryRecordRuleDeps, op: RecordRuleOp): Promise<ConditionEnvelope> {
  if (!params.recordRuleEnabled()) return { kind: 'true', reason: 'record_rule_disabled' };
  if (params.isControlPlaneMetaModel()) return { kind: 'true', reason: 'control_plane_meta_model' };
  if (params.getRecordRuleBypassDepth() > 0) return { kind: 'true', reason: 'record_rule_bypass' };

  const { enabled, allow, method } = getRepositoryTopLevelRecordRuleAllowlist();
  if (enabled) {
    const model = (params.meta.fullModelName || params.meta.modelName || params.meta.name || 'Unknown') as string;
    const key = `${model}:${op}`;
    if (!allow.has(key)) {
      throw params.permissionDenied('record_rule_entry_allowlist_miss', 'entry record rule allowlist miss', {
        model,
        op,
        method,
      });
    }
    return { kind: 'true', reason: 'entry_record_rule_allowlist' };
  }

  const cache = getRepositoryRecordRuleCache();
  const key = buildRepositoryRecordRuleCacheKey(params, op);
  const cached = cache.get(key);
  if (cached) return cached;

  const model = (params.meta.fullModelName || params.meta.modelName || params.meta.name || 'Unknown') as string;
  let result: unknown;
  try {
    result = await params.withRecordRuleBypass(async () => AuthUserService.GetRecordRuleCondition(model, op));
  } catch (error) {
    if (isAuthServiceUnavailable(error)) {
      const env: ConditionEnvelope = { kind: 'true', reason: 'auth_service_unavailable' };
      cache.set(key, env);
      return env;
    }
    throw params.permissionDenied('record_rule_fetch_failed', 'failed to fetch record rule condition', { model, op });
  }

  const env = normalizeRepositoryRecordRuleEnvelope(result);
  cache.set(key, env);
  return env;
}

function getActiveCompanyIdForRepositoryRecordRuleToken(params: RepositoryRecordRuleDeps): string {
  const companyId = params.normalizeCompanyIdForWrite();
  if (companyId) return companyId;
  throw params.permissionDenied('record_rule_missing_company_id', 'missing ctx.activeCompanyId for $companyId token', {
    model: params.meta.fullModelName || params.meta.modelName || params.meta.name || 'unknown_model',
  });
}

function getCompanyIdsForRepositoryRecordRuleToken(params: RepositoryRecordRuleDeps): string[] {
  const companyIds = params.normalizeCompanyIds();
  if (companyIds.length) return companyIds;
  throw params.permissionDenied('record_rule_missing_company_ids', 'missing ctx.enabledCompanyIds for $companyIds token', {
    model: params.meta.fullModelName || params.meta.modelName || params.meta.name || 'unknown_model',
  });
}

export function replaceRepositoryRecordRuleConditionTokens(params: RepositoryRecordRuleDeps, condition: BaseQueryCondition): BaseQueryCondition {
  const userId = String(params.userId || '').trim();
  const companyId = () => getActiveCompanyIdForRepositoryRecordRuleToken(params);
  const companyIds = () => getCompanyIdsForRepositoryRecordRuleToken(params);

  const normalizeConditionInput = (input: unknown): unknown => {
    if (input === null || input === undefined) return [];
    if (Array.isArray(input)) return input;

    if (typeof input === 'object') {
      try {
        if (Object.keys(input as object).length === 0) return [];
      } catch {
        // ignore
      }
      return input;
    }

    if (typeof input !== 'string') return input;

    const normalized = input.trim();
    if (normalized === '') return [];

    if (normalized.startsWith('{') || normalized.startsWith('[') || normalized === 'null') {
      try {
        const parsed = JSON.parse(normalized);
        return parsed === null ? [] : parsed;
      } catch {
        throw params.permissionDenied('record_rule_invalid_condition_json', 'invalid JSON in record rule condition', {
          model: params.meta.fullModelName || params.meta.modelName || params.meta.name || 'unknown_model',
        });
      }
    }

    return input;
  };

  const invalidCondition = (got: unknown): never => {
    let preview = '';
    try {
      preview = JSON.stringify(got);
    } catch {
      try {
        preview = String(got);
      } catch {
        preview = '[unstringifiable]';
      }
    }

    const clippedPreview = preview.length > 400 ? `${preview.slice(0, 400)}…` : preview;
    const message = `invalid record rule condition shape (type=${typeof got}, isArray=${Array.isArray(got)}, preview=${clippedPreview})`;

    throw params.permissionDenied('record_rule_invalid_condition', message, {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name || 'unknown_model',
      type: typeof got,
      isArray: Array.isArray(got) ? 'true' : 'false',
      length: Array.isArray(got) ? String(got.length) : '',
      preview: clippedPreview,
    });
  };

  const replaceInAny = (value: unknown): unknown => {
    if (value === null || value === undefined) return value;
    if (typeof value === 'string') {
      const normalized = value.trim();
      if (normalized === '$userId') {
        if (!userId) {
          throw params.permissionDenied('record_rule_missing_user_id', 'missing identity.userId for $userId token', {
            model: params.meta.fullModelName || params.meta.modelName || params.meta.name || 'unknown_model',
          });
        }
        return userId;
      }
      if (normalized === '$companyId') return companyId();
      if (normalized === '$companyIds') return companyIds();

      if (/^\$[A-Za-z_][A-Za-z0-9_]*$/.test(normalized)) {
        throw params.permissionDenied('record_rule_unknown_token', 'unknown token in record rule condition', {
          model: params.meta.fullModelName || params.meta.modelName || params.meta.name || 'unknown_model',
          token: normalized,
        });
      }
      return value;
    }
    if (Array.isArray(value)) return value.map(item => replaceInAny(item));
    if (isObjectRecord(value)) {
      const output: UnknownRecord = {};
      for (const [key, item] of Object.entries(value)) output[key] = replaceInAny(item);
      return output;
    }
    return value;
  };

  const walk = (input: unknown): BaseQueryCondition => {
    const value = normalizeConditionInput(input);

    if (Array.isArray(value)) {
      if (value.length === 0) return value as unknown as BaseQueryCondition;
      if (value.length >= 3) {
        const [field, op, leaf] = value;
        return [String(field), op as BaseQueryCondition extends readonly [string, infer TOp, unknown] ? TOp : never, replaceInAny(leaf)] as BaseQueryCondition;
      }
      return invalidCondition(value);
    }

    if (isObjectRecord(value)) {
      const andValue = value.And;
      if (Array.isArray(andValue)) {
        return { And: andValue.map(item => walk(item)) };
      }
      const orValue = value.Or;
      if (Array.isArray(orValue)) {
        return { Or: orValue.map(item => walk(item)) };
      }
      return invalidCondition(value);
    }

    return invalidCondition(value);
  };

  return walk(condition);
}
