// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getCurrentReq, getOrInitReqServiceState, memoizeInReqState } from '@/core/service/api/context';
import { createServiceByModel } from '@/core/service/rpc';
import type { ConditionEnvelope, RecordRuleOp } from '@/core/service/api/authz';
import type IrApplicationModel from '@/meta/service/models/ir_application';
import type IrFieldModel from '@/meta/service/models/ir_field';
import type IrModelModel from '@/meta/service/models/ir_model';
import RoleRecordRule from './role_record_rule';
import { maybeId } from './_user_authz_shared';

const IrApplication = createServiceByModel<typeof IrApplicationModel>('meta.IrApplication');
const IrModel = createServiceByModel<typeof IrModelModel>('meta.IrModel');
const IrField = createServiceByModel<typeof IrFieldModel>('meta.IrField');

type RoleScope = { global: boolean; companies: string[] };

const PERM_FIELD_BY_OP: Record<RecordRuleOp, keyof RoleRecordRule> = {
  read: 'PermRead',
  write: 'PermWrite',
  create: 'PermCreate',
  delete: 'PermDelete',
};

export type RecordRuleEvalInput = {
  appName: string;
  modelName: string;
  hasCompany: boolean;
  opValue: RecordRuleOp;
  roleIds: string[];
  roleScopesById: Record<string, RoleScope>;
};

function getRecordRuleReqState(): Record<string, unknown> | undefined {
  const req = getCurrentReq();
  return req ? getOrInitReqServiceState(req) : undefined;
}

function buildRecordRuleMetaCacheKey(appName: string, modelName: string): string {
  return `recordRuleMeta::${String(appName || '').trim()}::${String(modelName || '').trim()}`;
}

async function resolveRecordRuleMetaCached(appName: string, modelName: string): Promise<{ irApplicationId: string; modelHit: any; modelId: string }> {
  const state = getRecordRuleReqState();
  const key = buildRecordRuleMetaCacheKey(appName, modelName);
  return await memoizeInReqState(state, key, async () => {
    const [apps, models] = await Promise.all([
      IrApplication.Search({ And: [['Name', '=', appName]] } as any, { fields: ['Id'], limit: 1 } as any),
      IrModel.Search(
        {
          And: [
            ['Name', '=', modelName],
            ['Application', '=', appName],
          ],
        } as any,
        { fields: ['Id', 'CompanyScoped'], limit: 1 }
      ),
    ]);

    const irApplicationId = String((apps as any)?.[0]?.Id || '').trim();
    const modelHit: any = (models as any)?.[0] as any;
    const modelId = String(modelHit?.Id || '').trim();
    return { irApplicationId, modelHit, modelId };
  });
}

async function computeCompanyGateMode(modelId: string, companyScoped: boolean, hasCompany: boolean): Promise<{ enabled: boolean; reason?: string }> {
  if (!hasCompany) return { enabled: false, reason: 'no_company_context' };
  if (!companyScoped) return { enabled: false, reason: 'model_not_company_scoped' };

  const req = getCurrentReq();
  const state = req ? getOrInitReqServiceState(req) : undefined;
  const key = `companyGateMode::${modelId}`;
  return await memoizeInReqState(state, key, async (): Promise<{ enabled: boolean; reason?: string }> => {
    try {
      const hasCompanyIdField =
        Number(
          await IrField.Count({
            And: [
              ['ModelId', '=', modelId],
              ['Name', '=', 'CompanyId'],
            ],
          } as any)
        ) > 0;
      if (!hasCompanyIdField) {
        return { enabled: false, reason: 'company_scoped_missing_company_id_field' } as const;
      }

      return { enabled: true } as const;
    } catch {
      return { enabled: false, reason: 'meta_company_gate_error' };
    }
  });
}

function buildCompanyGateExpr(scope: RoleScope, companyGateEnabled: boolean): any {
  if (!companyGateEnabled) return null;
  if (scope.global) return null;
  const ids = scope.companies || [];
  const companyIn: any = ['CompanyId', 'in', ids] as any;
  const shared: any = ['CompanyId', 'is', null] as any;
  return { Or: [companyIn, shared] } as any;
}

/**
 * Core RecordRule evaluation: resolve meta, pick-one scope, load rules, and merge conditions.
 */
export async function evaluateRecordRuleCondition(input: RecordRuleEvalInput): Promise<ConditionEnvelope> {
  const { irApplicationId, modelHit, modelId } = await resolveRecordRuleMetaCached(input.appName, input.modelName);
  if (!modelId) return { kind: 'false', reason: 'model_not_found' };

  const companyGate = await computeCompanyGateMode(modelId, Boolean(modelHit?.CompanyScoped), input.hasCompany);

  if (input.roleIds.length === 0) {
    return { kind: 'true', reason: `no_roles_${input.opValue}_allow` };
  }

  const permField = PERM_FIELD_BY_OP[input.opValue];

  const scopeOr: any[] = [
    {
      And: [
        ['IrModelId', '=', modelId],
        ['IrApplicationId', 'is', null],
      ],
    },
    ...(irApplicationId
      ? [
          {
            And: [
              ['IrModelId', 'is', null],
              ['IrApplicationId', '=', irApplicationId],
            ],
          } as any,
        ]
      : []),
    {
      And: [
        ['IrModelId', 'is', null],
        ['IrApplicationId', 'is', null],
      ],
    },
  ];

  const allRules = await RoleRecordRule.Search(
    {
      And: [['RoleId', 'in', input.roleIds], [permField as any, '=', true], { Or: scopeOr }],
    } as any,
    { fields: ['RoleId', 'Condition', 'IrModelId', 'IrApplicationId'], limit: 5000 }
  );

  const modelRules: any[] = [];
  const appRules: any[] = [];
  const globalRules: any[] = [];

  for (const r of allRules || []) {
    const rModelId = maybeId((r as any).IrModelId);
    const rAppId = maybeId((r as any).IrApplicationId);

    if (rModelId === modelId && !rAppId) {
      modelRules.push(r);
    } else if (!rModelId && rAppId === irApplicationId) {
      appRules.push(r);
    } else if (!rModelId && !rAppId) {
      globalRules.push(r);
    }
  }

  const rules = modelRules.length > 0 ? modelRules : appRules.length > 0 ? appRules : globalRules;

  if (!rules || rules.length === 0) {
    return { kind: 'true', reason: `no_rules_${input.opValue}_allow` };
  }

  const exprs: any[] = [];
  for (const r of rules || []) {
    const roleId = maybeId((r as any).RoleId) || '';
    const scope = input.roleScopesById?.[roleId] || { global: true, companies: [] };
    const gate = buildCompanyGateExpr(scope, companyGate.enabled);
    const cond = (r as any).Condition;

    const isTrueCond = cond === undefined || cond === null || (Array.isArray(cond) && cond.length === 0);
    if (isTrueCond && !gate) {
      return { kind: 'true', reason: 'global_allow_rule' };
    }

    const expr = isTrueCond ? gate : !gate ? cond : ({ And: [gate, cond] } as any);
    if (expr !== undefined && expr !== null) {
      exprs.push(expr);
    }
  }

  if (exprs.length === 0) {
    return { kind: 'true', reason: companyGate.enabled ? 'rules_with_empty_condition_or_company_gate' : 'rules_with_empty_condition' };
  }
  if (exprs.length === 1) {
    return { kind: 'expr', expr: exprs[0], reason: 'single_rule' };
  }
  return { kind: 'expr', expr: { Or: exprs } as any, reason: 'or_merged' };
}
