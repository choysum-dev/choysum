// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getCurrentReq, getOrInitReqServiceState, memoizeInReqState } from '@/core/service/api/context';
import { createServiceByModel } from '@/core/service/rpc';
import type { ConditionEnvelope, RecordRuleOp } from '@/core/service/api/authz';
import type IrApplicationModel from '@/meta/service/models/ir_application';
import type IrFieldModel from '@/meta/service/models/ir_field';
import type IrModelModel from '@/meta/service/models/ir_model';
import RoleRecordRule from './role_record_rule';
import type { RoleRecordRuleKind } from './role_record_rule';
import { maybeId, withPermissionGraphBypass } from './_user_authz_shared';

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

function normalizeKind(raw: unknown): RoleRecordRuleKind {
  const kind = String(raw ?? 'grant')
    .trim()
    .toLowerCase();
  return kind === 'restrict' ? 'restrict' : 'grant';
}

function isTrueCondition(cond: unknown): boolean {
  if (cond === undefined || cond === null) return true;
  if (Array.isArray(cond) && cond.length === 0) return true;
  if (typeof cond === 'string' && !String(cond).trim()) return true;
  if (typeof cond === 'object' && !Array.isArray(cond)) {
    const keys = Object.keys(cond as object);
    if (keys.length === 0) return true;
    if (keys.length === 1) {
      const key = keys[0];
      const val = (cond as Record<string, unknown>)[key];
      if ((key === 'And' || key === 'Or') && Array.isArray(val) && val.length === 0) return true;
    }
  }
  return false;
}

function ruleAudienceScope(roleId: string, roleScopesById: Record<string, RoleScope>): RoleScope {
  if (!roleId) {
    // Everyone rules (RoleId null): no role-company gate; companyFilter stays orthogonal if added later.
    return { global: true, companies: [] };
  }
  const mapped = roleScopesById?.[roleId];
  if (mapped) return mapped;
  // Missing scope for a concrete role: deny-leaning (empty company gate) rather than global.
  return { global: false, companies: [] };
}

function buildRuleExpr(rule: any, companyGateEnabled: boolean, roleScopesById: Record<string, RoleScope>): any {
  const roleId = maybeId(rule?.RoleId) || '';
  const scope = ruleAudienceScope(roleId, roleScopesById);
  const gate = buildCompanyGateExpr(scope, companyGateEnabled);
  const cond = rule?.Condition;
  if (isTrueCondition(cond)) {
    return gate; // null ⇒ unconstrained TRUE for this rule
  }
  if (!gate) return cond;
  return { And: [gate, cond] } as any;
}

function orMerge(exprs: any[]): any {
  if (exprs.length === 1) return exprs[0];
  return { Or: exprs } as any;
}

/**
 * Core RecordRule evaluation (Security Algebra §5.4):
 * matching Kind=grant → OR; Kind=restrict → AND onto grants; no grant ⇒ DENY.
 * RoleId null = everyone; all matching scopes participate (no pick-one).
 *
 * Permission-graph reads run under bypass so direct callers (e.g. document owner
 * auth) are not blocked by deny-default on RoleRecordRule itself. Repository
 * entry already wraps GetRecordRuleCondition in record-rule bypass; this keeps
 * both paths consistent.
 */
export async function evaluateRecordRuleCondition(input: RecordRuleEvalInput): Promise<ConditionEnvelope> {
  return await withPermissionGraphBypass(async () => {
    const { irApplicationId, modelHit, modelId } = await resolveRecordRuleMetaCached(input.appName, input.modelName);
    if (!modelId) return { kind: 'false', reason: 'model_not_found' };

    const companyGate = await computeCompanyGateMode(modelId, Boolean(modelHit?.CompanyScoped), input.hasCompany);
    const permField = PERM_FIELD_BY_OP[input.opValue];
    const roleIds = (input.roleIds || []).map(id => String(id || '').trim()).filter(Boolean);

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

    // Audience: everyone (RoleId null) OR any of the caller's effective roles.
    const audienceOr: any[] = [['RoleId', 'is', null] as any];
    if (roleIds.length > 0) {
      audienceOr.push(['RoleId', 'in', roleIds] as any);
    }

    const RULE_FETCH_LIMIT = 5000;
    const allRules = await RoleRecordRule.Search(
      {
        And: [{ Or: audienceOr }, [permField as any, '=', true], { Or: scopeOr }],
      } as any,
      { fields: ['RoleId', 'Kind', 'Condition', 'IrModelId', 'IrApplicationId'], limit: RULE_FETCH_LIMIT + 1 }
    );

    if ((allRules || []).length > RULE_FETCH_LIMIT) {
      // Fail closed: omitted restrict rows would otherwise silently widen access.
      console.error(
        `RoleRecordRule evaluation truncated for ${input.appName}.${input.modelName} op=${input.opValue}: matched >${RULE_FETCH_LIMIT} rows`
      );
      return { kind: 'false', reason: `record_rule_truncated_${input.opValue}_deny` };
    }

    const grantExprs: any[] = [];
    const restrictExprs: any[] = [];
    let hasUnconstrainedGrant = false;

    for (const r of allRules || []) {
      const rModelId = maybeId((r as any).IrModelId);
      const rAppId = maybeId((r as any).IrApplicationId);
      const modelScoped = rModelId === modelId && !rAppId;
      const appScoped = !rModelId && !!irApplicationId && rAppId === irApplicationId;
      const globalScoped = !rModelId && !rAppId;
      if (!modelScoped && !appScoped && !globalScoped) continue;

      const kind = normalizeKind((r as any).Kind);
      const expr = buildRuleExpr(r, companyGate.enabled, input.roleScopesById || {});
      if (kind === 'restrict') {
        if (expr != null) restrictExprs.push(expr);
        // Unconstrained restrict (TRUE) is a no-op AND — skip.
        continue;
      }

      if (expr == null) {
        hasUnconstrainedGrant = true;
        continue;
      }
      grantExprs.push(expr);
    }

    const hasGrant = hasUnconstrainedGrant || grantExprs.length > 0;
    if (!hasGrant) {
      return { kind: 'false', reason: `no_grant_${input.opValue}_deny` };
    }

    const parts: any[] = [];
    if (!hasUnconstrainedGrant) {
      parts.push(orMerge(grantExprs));
    }
    // Unconstrained grant ⇒ grant domain is TRUE; only restricts (if any) remain.
    for (const r of restrictExprs) {
      parts.push(r);
    }

    if (parts.length === 0) {
      return { kind: 'true', reason: 'grant_unconstrained' };
    }
    if (parts.length === 1) {
      return { kind: 'expr', expr: parts[0], reason: restrictExprs.length ? 'grant_and_restrict' : 'grant_domain' };
    }
    // parts.length > 1 ⇒ AND-compose (never call a 1-element helper).
    return { kind: 'expr', expr: { And: parts } as any, reason: 'grant_or_and_restricts' };
  });
}
