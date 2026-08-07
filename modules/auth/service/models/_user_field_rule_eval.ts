// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import { getCurrentReq, getOrInitReqServiceState, memoizeInReqState } from '@/core/service/api/context';
import { newAuthError, AuthErrCode, GrpcCode } from '../error';
import { _t } from '../i18n';
import RoleFieldRule from './role_field_rule';
import type MetaFieldModel from '@/meta/service/models/field';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { resolveEffectiveApplicationId, resolveEffectiveModelId } from './_resolve_effective_model';

const MetaField = createServiceByModel<typeof MetaFieldModel>('meta.MetaField');

function normalizeFieldPerm(v: any): 'allow' | 'deny' | null {
  if (v == null) return null;
  if (typeof v === 'object') {
    const raw = (v as any)?.value ?? (v as any)?.Value ?? (v as any)?.id ?? (v as any)?.Id;
    if (raw != null && raw !== v) return normalizeFieldPerm(raw);
  }
  const s = String(v ?? '')
    .trim()
    .toLowerCase();
  if (!s) return null;
  if (s === 'allow' || s === 'deny') return s;
  return null;
}

function pickField(obj: any, keys: string[]): any {
  if (!obj || (typeof obj !== 'object' && typeof obj !== 'function')) return undefined;

  for (const k of keys) {
    if (k in obj) return obj[k];
  }

  const norm = (s: string): string => s.toLowerCase().replace(/[^a-z0-9]/g, '');
  const normalizedWants = keys.map(norm);

  for (const k of Object.keys(obj)) {
    if (normalizedWants.includes(norm(k))) {
      return obj[k];
    }
  }

  return undefined;
}

const SYSTEM_FIELDS = new Set(['Id', 'CreatedAt', 'UpdatedAt', 'DeletedAt', 'DisplayName']);

export type FieldRuleEvalInput = {
  appName: string;
  modelName: string;
  modelFullName: string;
  roleIds: string[];
};

export type FieldRuleEvalResult = {
  denyReadFields: string[];
  denyWriteFields: string[];
  reason: string;
  hitRuleIds?: string[];
};

function getFieldRuleReqState(): Record<string, unknown> | undefined {
  const req = getCurrentReq();
  return req ? getOrInitReqServiceState(req) : undefined;
}

function buildFieldRuleMetaCacheKey(type: 'app' | 'model', appName: string, modelName?: string): string {
  return `fieldRuleMeta::${type}::${String(appName || '').trim()}::${String(modelName || '').trim()}`;
}

/**
 * Resolve meta application id by name (single effective row).
 */
async function resolveApplicationId(appName: string): Promise<string> {
  const state = getFieldRuleReqState();
  const key = buildFieldRuleMetaCacheKey('app', appName);
  return await memoizeInReqState(state, key, async () => resolveEffectiveApplicationId(appName));
}

/**
 * Resolve the single effective meta model id for (application, name).
 */
async function resolveModelId(appName: string, modelName: string): Promise<string> {
  const state = getFieldRuleReqState();
  const key = buildFieldRuleMetaCacheKey('model', appName, modelName);
  return await memoizeInReqState(state, key, async () => resolveEffectiveModelId(appName, modelName));
}

function denyAllNonSystemFields(fieldNames: string[], reason: string, hitRuleIds?: string[]): FieldRuleEvalResult {
  const names = [...fieldNames].sort();
  return { denyReadFields: names, denyWriteFields: [...names], reason, hitRuleIds };
}

/**
 * Core FieldRule evaluation: resolve meta, load rules, partition by scope, and decide per-field.
 *
 * Deny-by-default (§5.5 / PR-C-1): no matching allow ⇒ field enters deny lists.
 * More-specific scope wins; same-scope deny-wins; read-deny ⇒ write-deny.
 */
export async function evaluateFieldRules(input: FieldRuleEvalInput): Promise<FieldRuleEvalResult> {
  const [applicationId, modelId] = await Promise.all([
    resolveApplicationId(input.appName),
    resolveModelId(input.appName, input.modelName),
  ]);
  if (!modelId) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: _t('Model does not exist', { scope: 'service/models/_user_field_rule_eval' }),
    })
      .withGrpcCode(GrpcCode.InvalidArgument)
      .withMetadata({ model: input.modelFullName });
  }

  // Load meta fields (needed for deny-all early exits and per-field decisions).
  const fields = await MetaField.Search(['ModelId', '=', modelId] as any, { fields: ['Id', 'Name'], limit: 5000 } as any);
  const fieldNameById = new Map<string, string>();
  const fieldIdsByName = new Map<string, string[]>();
  const fieldIdSet = new Set<string>();
  for (const f of fields || []) {
    const id = String((f as any)?.Id || '').trim();
    const name = String((f as any)?.Name || '').trim();
    if (!id || !name) continue;
    if (SYSTEM_FIELDS.has(name)) continue;
    fieldNameById.set(id, name);
    if (!fieldIdSet.has(id)) fieldIdSet.add(id);
    const xs = fieldIdsByName.get(name) || [];
    xs.push(id);
    fieldIdsByName.set(name, xs);
  }

  const fieldNames = Array.from(fieldIdsByName.keys()).sort();

  if (fieldIdsByName.size === 0) {
    // Nothing to deny (system fields are never listed).
    return { denyReadFields: [], denyWriteFields: [], reason: 'no_fields_deny_by_default' };
  }

  if (input.roleIds.length === 0) {
    return denyAllNonSystemFields(fieldNames, 'no_roles_deny_by_default');
  }

  // Load rules
  const rules = await RoleFieldRule.Search(
    {
      And: [
        ['RoleId', 'in', input.roleIds],
        {
          Or: [
            {
              And: [
                ['MetaModelId', '=', modelId],
                ['MetaFieldId', 'in', Array.from(fieldIdSet)],
                ['MetaApplicationId', 'is', null],
                ['LogicalModelName', 'is', null],
              ],
            },
            {
              And: [
                ['MetaModelId', '=', modelId],
                ['MetaFieldId', 'is', null],
                ['MetaApplicationId', 'is', null],
                ['LogicalModelName', 'is', null],
              ],
            },
            ...(applicationId
              ? [
                  {
                    And: [
                      ['MetaApplicationId', '=', applicationId],
                      ['MetaModelId', 'is', null],
                      ['MetaFieldId', 'is', null],
                      ['LogicalModelName', 'is', null],
                    ],
                  },
                ]
              : []),
            {
              And: [
                ['MetaApplicationId', 'is', null],
                ['MetaModelId', 'is', null],
                ['MetaFieldId', 'is', null],
                ['LogicalModelName', '=', input.modelName],
              ],
            },
            {
              And: [
                ['MetaApplicationId', 'is', null],
                ['MetaModelId', 'is', null],
                ['MetaFieldId', 'is', null],
                ['LogicalModelName', 'is', null],
              ],
            },
          ],
        },
      ],
    } as any,
    {
      fields: ['Id', 'MetaApplicationId', 'MetaModelId', 'MetaFieldId', 'LogicalModelName', 'PermRead', 'PermWrite'],
      limit: 5000,
    } as any
  );

  if (!rules || rules.length === 0) {
    return denyAllNonSystemFields(fieldNames, 'no_field_rules_deny_by_default');
  }

  // Partition by scope
  const fieldRulesByFieldName = new Map<string, any[]>();
  const modelRules: any[] = [];
  const appRules: any[] = [];
  const logicalRules: any[] = [];
  const globalRules: any[] = [];
  const modelNameWant = String(input.modelName || '').trim();

  for (const r of rules || []) {
    const rid = String((r as any)?.Id ?? '').trim();
    const irApp = normalizeRefId(pickField(r, ['MetaApplicationId', 'meta_application_id', 'irApplicationId']));
    const irModel = normalizeRefId(pickField(r, ['MetaModelId', 'meta_model_id', 'irModelId']));
    const irField = normalizeRefId(pickField(r, ['MetaFieldId', 'meta_field_id', 'irFieldId']));
    const logicalName = String(pickField(r, ['LogicalModelName', 'logical_model_name']) ?? '').trim() || null;
    const permRead = normalizeFieldPerm(pickField(r, ['PermRead', 'perm_read', 'permRead']));
    const permWrite = normalizeFieldPerm(pickField(r, ['PermWrite', 'perm_write', 'permWrite']));

    const rule: Record<string, unknown> = { irApp, irModel, irField, logicalName, permRead, permWrite };
    if (rid) rule.__rid = rid;

    const isField = irField != null && irModel != null && irApp == null && logicalName == null;
    const isModel = irField == null && irModel != null && irApp == null && logicalName == null;
    const isApp = irField == null && irModel == null && irApp != null && logicalName == null;
    const isLogical = irField == null && irModel == null && irApp == null && logicalName != null;
    const isGlobal = irField == null && irModel == null && irApp == null && logicalName == null;

    if (isField) {
      if (!fieldIdSet.has(irField)) continue;
      if (irModel !== modelId) continue;
      const fieldName = fieldNameById.get(irField);
      if (!fieldName) continue;
      const xs = fieldRulesByFieldName.get(fieldName) || [];
      xs.push(rule);
      fieldRulesByFieldName.set(fieldName, xs);
    } else if (isModel) {
      if (irModel !== modelId) continue;
      modelRules.push(rule);
    } else if (isApp) {
      if (!applicationId || irApp !== applicationId) continue;
      appRules.push(rule);
    } else if (isLogical) {
      if (logicalName !== modelNameWant) continue;
      logicalRules.push(rule);
    } else if (isGlobal) {
      globalRules.push(rule);
    }
  }

  function decideInScope(xs: any[], dim: 'read' | 'write'): 'allow' | 'deny' | undefined {
    let hasAllow = false;
    for (const r of xs || []) {
      const v = dim === 'read' ? (r as any)?.permRead : (r as any)?.permWrite;
      if (v === 'deny') return 'deny';
      if (v === 'allow') hasAllow = true;
    }
    return hasAllow ? 'allow' : undefined;
  }

  function decideEffective(fieldName: string, dim: 'read' | 'write'): 'allow' | 'deny' {
    // Field > MetaModel > Application > LogicalModel > Global
    const buckets: any[][] = [
      fieldRulesByFieldName.get(fieldName) || [],
      modelRules,
      appRules,
      logicalRules,
      globalRules,
    ];
    for (const b of buckets) {
      const d = decideInScope(b, dim);
      if (d) return d;
    }
    return 'deny';
  }

  const denyReadFields: string[] = [];
  const denyWriteFields: string[] = [];
  for (const name of fieldNames) {
    const readDecision = decideEffective(name, 'read');
    let writeDecision = decideEffective(name, 'write');
    // read-deny ⇒ write-deny (cannot write what you cannot read).
    if (readDecision === 'deny') writeDecision = 'deny';
    if (readDecision === 'deny') denyReadFields.push(name);
    if (writeDecision === 'deny') denyWriteFields.push(name);
  }

  denyReadFields.sort();
  denyWriteFields.sort();

  const hitRuleIds = Array.from(
    new Set(
      [
        ...Array.from(fieldRulesByFieldName.values()).flat(),
        ...modelRules,
        ...appRules,
        ...logicalRules,
        ...globalRules,
      ]
        .map(r => String((r as any)?.__rid ?? '').trim())
        .filter(Boolean)
    )
  ).sort();

  return { denyReadFields, denyWriteFields, reason: 'ok', hitRuleIds };
}
