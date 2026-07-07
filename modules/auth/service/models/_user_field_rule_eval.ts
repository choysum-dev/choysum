// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import { newAuthError, AuthErrCode, GrpcCode } from '../error';
import RoleFieldRule from './role_field_rule';
import type IrApplicationModel from '@/meta/service/models/ir_application';
import type IrFieldModel from '@/meta/service/models/ir_field';
import type IrModelModel from '@/meta/service/models/ir_model';
import { normalizeRefId } from '@/core/service/utils/normalization';

const IrApplication = createServiceByModel<typeof IrApplicationModel>('meta.IrApplication');
const IrModel = createServiceByModel<typeof IrModelModel>('meta.IrModel');
const IrField = createServiceByModel<typeof IrFieldModel>('meta.IrField');

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
  rawModel: string;
  roleIds: string[];
};

export type FieldRuleEvalResult = {
  denyReadFields: string[];
  denyWriteFields: string[];
  reason: string;
};

/**
 * Resolve meta application ids by name.
 */
async function resolveApplicationIds(appName: string): Promise<string[]> {
  const apps = await IrApplication.Search(
    ['Name', '=', appName] as any,
    { fields: ['Id', 'UpdatedAt'], orderBy: { field: 'UpdatedAt', order: 'desc' }, limit: 5000 } as any
  );
  const idSet = new Set<string>();
  const ids: string[] = [];
  for (const a of apps || []) {
    const id = String((a as any)?.Id || '').trim();
    if (!id) continue;
    if (idSet.has(id)) continue;
    idSet.add(id);
    ids.push(id);
  }
  return ids;
}

/**
 * Resolve meta model ids by logical name (application-agnostic to handle re-materializations).
 */
async function resolveModelIds(modelName: string): Promise<string[]> {
  const models = await IrModel.Search(
    ['Name', '=', modelName] as any,
    { fields: ['Id', 'UpdatedAt'], orderBy: { field: 'UpdatedAt', order: 'desc' }, limit: 5000 } as any
  );
  const idSet = new Set<string>();
  const ids: string[] = [];
  for (const m of models || []) {
    const id = String((m as any)?.Id || '').trim();
    if (!id) continue;
    if (idSet.has(id)) continue;
    idSet.add(id);
    ids.push(id);
  }
  return ids;
}

/**
 * Core FieldRule evaluation: resolve meta, load rules, partition by scope, and decide per-field.
 */
export async function evaluateFieldRules(input: FieldRuleEvalInput): Promise<FieldRuleEvalResult> {
  const applicationIds = await resolveApplicationIds(input.appName);
  if (applicationIds.length === 0) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: 'Application does not exist',
    })
      .withGrpcCode(GrpcCode.InvalidArgument)
      .withMetadata({ application: input.appName, model: input.rawModel });
  }

  const modelIds = await resolveModelIds(input.modelName);
  if (modelIds.length === 0) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: 'Model does not exist',
    })
      .withGrpcCode(GrpcCode.InvalidArgument)
      .withMetadata({ model: input.rawModel });
  }

  if (input.roleIds.length === 0) {
    return { denyReadFields: [], denyWriteFields: [], reason: 'no_roles_allow_by_default' };
  }

  // Load meta fields
  const fields = await IrField.Search(['ModelId', 'in', modelIds] as any, { fields: ['Id', 'Name'], limit: 5000 } as any);
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

  if (fieldIdsByName.size === 0) {
    return { denyReadFields: [], denyWriteFields: [], reason: 'no_fields_allow_by_default' };
  }

  const modelIdSet = new Set<string>(modelIds);
  const applicationIdSet = new Set<string>(applicationIds);

  // Load rules
  const rules = await RoleFieldRule.Search(
    {
      And: [
        ['RoleId', 'in', input.roleIds],
        {
          Or: [
            {
              And: [
                ['IrModelId', 'in', modelIds],
                ['IrFieldId', 'in', Array.from(fieldIdSet)],
                ['IrApplicationId', 'is', null],
              ],
            },
            {
              And: [
                ['IrModelId', 'in', modelIds],
                ['IrFieldId', 'is', null],
                ['IrApplicationId', 'is', null],
              ],
            },
            {
              And: [
                ['IrApplicationId', 'in', applicationIds],
                ['IrModelId', 'is', null],
                ['IrFieldId', 'is', null],
              ],
            },
            {
              And: [
                ['IrApplicationId', 'is', null],
                ['IrModelId', 'is', null],
                ['IrFieldId', 'is', null],
              ],
            },
          ],
        },
      ],
    } as any,
    {
      fields: ['Id', 'IrApplicationId', 'IrModelId', 'IrFieldId', 'PermRead', 'PermWrite'],
      limit: 5000,
    } as any
  );

  if (!rules || rules.length === 0) {
    return { denyReadFields: [], denyWriteFields: [], reason: 'no_field_rules_allow_by_default' };
  }

  // Partition by scope
  const fieldRulesByFieldName = new Map<string, any[]>();
  const modelRules: any[] = [];
  const appRules: any[] = [];
  const globalRules: any[] = [];

  for (const r of rules || []) {
    const rid = String((r as any)?.Id ?? '').trim();
    const irApp = normalizeRefId(pickField(r, ['IrApplicationId', 'ir_application_id', 'irApplicationId']));
    const irModel = normalizeRefId(pickField(r, ['IrModelId', 'ir_model_id', 'irModelId']));
    const irField = normalizeRefId(pickField(r, ['IrFieldId', 'ir_field_id', 'irFieldId']));
    const permRead = normalizeFieldPerm(pickField(r, ['PermRead', 'perm_read', 'permRead']));
    const permWrite = normalizeFieldPerm(pickField(r, ['PermWrite', 'perm_write', 'permWrite']));

    const rule = { __rid: rid, irApp, irModel, irField, permRead, permWrite };

    const isField = irField != null && irModel != null && irApp == null;
    const isModel = irField == null && irModel != null && irApp == null;
    const isApp = irField == null && irModel == null && irApp != null;
    const isGlobal = irField == null && irModel == null && irApp == null;

    if (isField) {
      if (!fieldIdSet.has(irField)) continue;
      if (!modelIdSet.has(irModel)) continue;
      const fieldName = fieldNameById.get(irField);
      if (!fieldName) continue;
      const xs = fieldRulesByFieldName.get(fieldName) || [];
      xs.push(rule);
      fieldRulesByFieldName.set(fieldName, xs);
    } else if (isModel) {
      if (!modelIdSet.has(irModel)) continue;
      modelRules.push(rule);
    } else if (isApp) {
      if (!applicationIdSet.has(irApp)) continue;
      appRules.push(rule);
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
    const buckets: any[][] = [fieldRulesByFieldName.get(fieldName) || [], modelRules, appRules, globalRules];
    for (const b of buckets) {
      const d = decideInScope(b, dim);
      if (d) return d;
    }
    return 'allow';
  }

  const denyReadFields: string[] = [];
  const denyWriteFields: string[] = [];
  const fieldNames = Array.from(fieldIdsByName.keys()).sort();
  for (const name of fieldNames) {
    const readDecision = decideEffective(name, 'read');
    let writeDecision = decideEffective(name, 'write');
    if (readDecision === 'deny') writeDecision = 'deny';
    if (readDecision === 'deny') denyReadFields.push(name);
    if (writeDecision === 'deny') denyWriteFields.push(name);
  }

  denyReadFields.sort();
  denyWriteFields.sort();

  return { denyReadFields, denyWriteFields, reason: 'ok' };
}
