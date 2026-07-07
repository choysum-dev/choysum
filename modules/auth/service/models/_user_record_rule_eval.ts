// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getCurrentReq, getOrInitReqServiceState } from '@/core/service/api/context';
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

async function computeCompanyGateMode(modelId: string, companyScoped: boolean, hasCompany: boolean): Promise<{ enabled: boolean; reason?: string }> {
  if (!hasCompany) return { enabled: false, reason: 'no_company_context' };

  const req = getCurrentReq();
  const state = req ? getOrInitReqServiceState(req) : undefined;
  const key = `companyGateMode::${modelId}`;
  const existing = state ? state[key] : undefined;
  if (existing) {
    if (typeof existing?.then === 'function') {
      const v = await existing;
      try {
        state[key] = v;
      } catch {
        /* ignore */
      }
      return v;
    }
    return existing;
  }

  const p = (async (): Promise<{ enabled: boolean; reason?: string }> => {
    if (!companyScoped) return { enabled: false, reason: 'model_not_company_scoped' };

    const hasCompanyIdField =
      Number(
        await IrField.Count({
          And: [
            ['ModelId', '=', modelId],
            ['Name', '=', 'CompanyId'],
          ],
        } as any)
      ) > 0;
    if (!hasCompanyIdField) return { enabled: false, reason: 'company_scoped_missing_company_id_field' };

    return { enabled: true };
  })()
    .then((v: any) => {
      if (state) {
        try {
          state[key] = v;
        } catch {
          /* ignore */
        }
      }
      return v;
    })
    .catch(() => {
      if (state) {
        try {
          delete state[key];
        } catch {
          /* ignore */
        }
      }
      return { enabled: false, reason: 'meta_company_gate_error' };
    });

  if (state) state[key] = p;
  const v = await p;
  if (state) {
    try {
      state[key] = v;
    } catch {
      /* ignore */
    }
  }
  return v;
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
  // Resolve meta ids
  const apps = await IrApplication.Search({ And: [['Name', '=', input.appName]] } as any, { fields: ['Id'], limit: 1 } as any);
  const irApplicationId = String((apps as any)?.[0]?.Id || '').trim();

  const models = await IrModel.Search(
    {
      And: [
        ['Name', '=', input.modelName],
        ['Application', '=', input.appName],
      ],
    } as any,
    { fields: ['Id', 'CompanyScoped'], limit: 1 }
  );
  const modelHit: any = (models as any)?.[0] as any;
  const modelId = String(modelHit?.Id || '').trim();
  if (!modelId) return { kind: 'false', reason: 'model_not_found' };

  const companyGate = await computeCompanyGateMode(modelId, Boolean(modelHit?.CompanyScoped), input.hasCompany);

  if (input.roleIds.length === 0) {
    return { kind: 'true', reason: `no_roles_${input.opValue}_allow` };
  }

  const permField = PERM_FIELD_BY_OP[input.opValue];

  const baseAnd: any[] = [
    ['RoleId', 'in', input.roleIds],
    [permField as any, '=', true],
  ];

  const hasAnyRule = async (scopeAnd: any[]): Promise<boolean> => {
    const n = Number(
      await RoleRecordRule.Count({
        And: [...scopeAnd, ...baseAnd],
      } as any)
    );
    if (!Number.isFinite(n)) throw new Error('invalid_role_record_rule_count');
    return n > 0;
  };

  const modelScopeAnd: any[] = [
    ['IrModelId', '=', modelId],
    ['IrApplicationId', 'is', null],
  ];
  const applicationScopeAnd: any[] = [
    ['IrModelId', 'is', null],
    ['IrApplicationId', '=', irApplicationId],
  ];
  const globalScopeAnd: any[] = [
    ['IrModelId', 'is', null],
    ['IrApplicationId', 'is', null],
  ];

  let pickedScopeAnd: any[] = [];

  if (await hasAnyRule(modelScopeAnd)) {
    pickedScopeAnd = modelScopeAnd;
  } else if (irApplicationId && (await hasAnyRule(applicationScopeAnd))) {
    pickedScopeAnd = applicationScopeAnd;
  } else if (await hasAnyRule(globalScopeAnd)) {
    pickedScopeAnd = globalScopeAnd;
  }

  const rules =
    pickedScopeAnd.length === 0
      ? []
      : await RoleRecordRule.Search(
          {
            And: [...pickedScopeAnd, ...baseAnd],
          } as any,
          { fields: ['RoleId', 'Condition'], limit: 5000 }
        );

  if (!rules || rules.length === 0) {
    return { kind: 'true', reason: `no_rules_${input.opValue}_allow` };
  }

  const exprs = (rules || [])
    .map(r => {
      const roleId = maybeId((r as any).RoleId) || '';
      const scope = input.roleScopesById?.[roleId] || { global: true, companies: [] };
      const gate = buildCompanyGateExpr(scope, companyGate.enabled);
      const cond = (r as any).Condition;

      const isTrueCond = cond === undefined || cond === null || (Array.isArray(cond) && cond.length === 0);
      if (isTrueCond) return gate;
      if (!gate) return cond;
      return { And: [gate, cond] } as any;
    })
    .filter(v => v !== undefined && v !== null);

  if (exprs.length === 0) {
    return { kind: 'true', reason: companyGate.enabled ? 'rules_with_empty_condition_or_company_gate' : 'rules_with_empty_condition' };
  }
  if (exprs.length === 1) {
    return { kind: 'expr', expr: exprs[0], reason: 'single_rule' };
  }
  return { kind: 'expr', expr: { Or: exprs } as any, reason: 'or_merged' };
}
