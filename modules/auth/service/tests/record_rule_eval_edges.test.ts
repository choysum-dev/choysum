// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import RoleRecordRule from '@/auth/service/models/role_record_rule';
import { buildCompanyGateExpr, evaluateRecordRuleCondition } from '@/auth/service/models/_user_record_rule_eval';
import { createServiceByModel } from '@/core/service/rpc';
import type IrApplicationModel from '@/meta/service/models/ir_application';
import type IrFieldModel from '@/meta/service/models/ir_field';
import type IrModelModel from '@/meta/service/models/ir_model';

const IrApplication = createServiceByModel<typeof IrApplicationModel>('meta.IrApplication');
const IrModel = createServiceByModel<typeof IrModelModel>('meta.IrModel');
const IrField = createServiceByModel<typeof IrFieldModel>('meta.IrField');

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};
  (globalThis as any).$choysum = root;
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = { depth: 0, fieldRuleMode: 'skip' };
  jsCtx.identity = {};
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

async function resolveModelId(appName: string, modelName: string): Promise<string> {
  const rows = await IrModel.Search(
    {
      And: [
        ['Name', '=', modelName],
        ['Application', '=', appName],
      ],
    } as any,
    { fields: ['Id'], limit: 1 }
  );
  const id = String((rows[0] as any)?.Id || '').trim();
  if (!id) throw new Error(`meta model not found: ${appName}.${modelName}`);
  return id;
}

test('P2-2 eval edges: model_not_found deny', async () => {
  resetRequestContext();
  const env = await evaluateRecordRuleCondition({
    appName: 'auth',
    modelName: uid('MissingModel'),
    hasCompany: false,
    opValue: 'read',
    roleIds: [],
    roleScopesById: {},
  });
  expect(env.kind).toBe('false');
  expect(String(env.reason || '')).toBe('model_not_found');
});

test('P2-2 eval edges: Search truncation fail-closed', async () => {
  resetRequestContext();
  const orig = (RoleRecordRule as any).Search;
  const fakeRows = Array.from({ length: 5001 }, () => ({
    RoleId: null,
    Kind: 'grant',
    Condition: null,
    IrModelId: 'x',
    IrApplicationId: null,
  }));
  (RoleRecordRule as any).Search = async () => fakeRows;
  const prevError = console.error;
  const errors: string[] = [];
  console.error = ((...args: unknown[]) => {
    errors.push(args.map(a => String(a)).join(' '));
  }) as typeof console.error;
  try {
    const env = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'User',
      hasCompany: false,
      opValue: 'read',
      roleIds: ['r1'],
      roleScopesById: {},
    });
    expect(env.kind).toBe('false');
    expect(String(env.reason || '')).toContain('record_rule_truncated_read_deny');
    expect(errors.some(e => e.includes('truncated'))).toBe(true);
  } finally {
    (RoleRecordRule as any).Search = orig;
    console.error = prevError;
  }
});

test('P2-2 eval edges: missing roleScopesById applies deny-lean company gate', async () => {
  resetRequestContext();
  // Isolate from suite-seeded RoleRecordRule rows (full CI unit shard shares one DB).
  const modelId = await resolveModelId('auth', 'CompanyScopedResource');
  const roleId = uid('role');
  const origSearch = (RoleRecordRule as any).Search;
  const origCount = (IrField as any).Count;
  (IrField as any).Count = async () => 1; // force company gate enabled
  (RoleRecordRule as any).Search = async () => [
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: { And: [['Name', '=', 'x']] },
      IrModelId: modelId,
      IrApplicationId: null,
    },
  ];
  try {
    // roleIds present but roleScopesById omits the role → deny-lean gate (empty company in).
    const env = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: {},
    });
    expect(env.kind).toBe('expr');
    const json = JSON.stringify((env as any).expr || {});
    expect(json).toContain('CompanyId');
    expect(json).toContain('Name');
    // Deny-lean scope → CompanyId in [].
    expect(json.includes('[]')).toBe(true);
  } finally {
    (RoleRecordRule as any).Search = origSearch;
    (IrField as any).Count = origCount;
  }
});

test('P2-2 eval edges: true-condition variants and multi grant+restrict compose', async () => {
  resetRequestContext();
  const modelId = await resolveModelId('auth', 'CompanyScopedResource');
  const roleId = uid('role');
  const origSearch = (RoleRecordRule as any).Search;

  const readRows = [
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: { And: [['Name', '=', 'a']] },
      IrModelId: modelId,
      IrApplicationId: null,
    },
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: { Or: [['Name', '=', 'b']] },
      IrModelId: modelId,
      IrApplicationId: null,
    },
    {
      RoleId: { Id: roleId },
      Kind: 'restrict',
      Condition: { And: [['Name', '!=', 'z']] },
      IrModelId: modelId,
      IrApplicationId: null,
    },
  ];
  const writeRows = [
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: { Or: [] },
      IrModelId: modelId,
      IrApplicationId: null,
    },
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: {},
      IrModelId: modelId,
      IrApplicationId: null,
    },
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: '',
      IrModelId: modelId,
      IrApplicationId: null,
    },
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: [],
      IrModelId: modelId,
      IrApplicationId: null,
    },
  ];

  try {
    (RoleRecordRule as any).Search = async () => readRows;
    const composed = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: false,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: true, companies: [] } },
    });
    expect(composed.kind).toBe('expr');
    expect(String((composed as any).reason || '')).toBe('grant_or_and_restricts');

    (RoleRecordRule as any).Search = async () => writeRows;
    const writeTrue = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: false,
      opValue: 'write',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: true, companies: [] } },
    });
    expect(writeTrue.kind).toBe('true');
    expect(String((writeTrue as any).reason || '')).toBe('grant_unconstrained');
  } finally {
    (RoleRecordRule as any).Search = origSearch;
  }
});

test('P2-2 eval edges: mismatched rule scopes skipped', async () => {
  resetRequestContext();
  const modelId = await resolveModelId('auth', 'CompanyScopedResource');
  const roleId = uid('role');
  const companyId = uid('C1');
  const origSearch = (RoleRecordRule as any).Search;
  const origCount = (IrField as any).Count;
  (RoleRecordRule as any).Search = async () => [
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: { And: [['Name', '=', 'ok']] },
      IrModelId: modelId,
      IrApplicationId: null,
    },
    {
      // Mismatched scope: model id for a different model with an app id set → skipped.
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: null,
      IrModelId: 'not-the-model',
      IrApplicationId: 'not-the-app',
    },
  ];
  (IrField as any).Count = async () => 1;

  try {
    const env = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: false, companies: [companyId] } },
    });
    expect(env.kind).toBe('expr');
    expect(String((env as any).reason || '')).toBe('grant_domain');
  } finally {
    (RoleRecordRule as any).Search = origSearch;
    (IrField as any).Count = origCount;
  }
});

test('P2-2 eval edges: company-isolated model without ownership field fail-closes gate', async () => {
  resetRequestContext();
  const modelId = await resolveModelId('auth', 'CompanyScopedResource');
  const roleId = uid('role');
  const companyId = uid('C1');
  const origSearch = (RoleRecordRule as any).Search;
  const origCount = (IrField as any).Count;
  (IrField as any).Count = async () => 0;
  (RoleRecordRule as any).Search = async () => [
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: { And: [['Name', '=', 'nogate']] },
      IrModelId: modelId,
      IrApplicationId: null,
    },
  ];
  try {
    const env = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: false, companies: [companyId] } },
    });
    expect(env.kind).toBe('false');
    expect(String((env as any).reason || '')).toBe('company_isolated_missing_ownership_field');
  } finally {
    (RoleRecordRule as any).Search = origSearch;
    (IrField as any).Count = origCount;
  }
});

test('P2-2 eval edges: unconstrained grant with one restrict uses grant_and_restrict', async () => {
  resetRequestContext();
  const modelId = await resolveModelId('auth', 'CompanyScopedResource');
  const userModelId = await resolveModelId('auth', 'User');
  const roleId = uid('role');
  const companyId = uid('C1');
  const origSearch = (RoleRecordRule as any).Search;

  try {
    (RoleRecordRule as any).Search = async () => [
      {
        RoleId: { Id: roleId },
        Kind: 'grant',
        Condition: null,
        IrModelId: modelId,
        IrApplicationId: null,
      },
      {
        RoleId: { Id: roleId },
        Kind: 'restrict',
        Condition: { And: [['Name', '!=', 'nope']] },
        IrModelId: modelId,
        IrApplicationId: null,
      },
    ];
    const andRestrict = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: false,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: true, companies: [] } },
    });
    expect(andRestrict.kind).toBe('expr');
    expect(String((andRestrict as any).reason || '')).toBe('grant_and_restrict');

    (RoleRecordRule as any).Search = async () => [
      {
        RoleId: { Id: roleId },
        Kind: 'grant',
        Condition: { And: [['Id', '!=', '']] },
        IrModelId: userModelId,
        IrApplicationId: null,
      },
    ];
    // User is not company-scoped → gate stays off even with hasCompany.
    const userEnv = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'User',
      hasCompany: true,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: false, companies: [companyId] } },
    });
    expect(userEnv.kind).toBe('expr');
    expect(JSON.stringify((userEnv as any).expr || {})).not.toContain('CompanyId');
  } finally {
    (RoleRecordRule as any).Search = origSearch;
  }
});

test('P2-2 eval edges: app-scoped grant participates when IrApplication resolves', async () => {
  resetRequestContext();
  const apps = await IrApplication.Search({ And: [['Name', '=', 'auth']] } as any, {
    fields: ['Id'],
    limit: 1,
  } as any);
  const appId = String((apps[0] as any)?.Id || '').trim();
  expect(appId.length > 0).toBe(true);
  const roleId = uid('role');
  const origSearch = (RoleRecordRule as any).Search;
  (RoleRecordRule as any).Search = async () => [
    {
      RoleId: { Id: roleId },
      Kind: 'grant',
      Condition: { And: [['Name', '=', 'from_app']] },
      IrModelId: null,
      IrApplicationId: appId,
    },
  ];
  try {
    const env = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: false,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: true, companies: [] } },
    });
    expect(env.kind).toBe('expr');
    expect(JSON.stringify((env as any).expr || {})).toContain('from_app');
  } finally {
    (RoleRecordRule as any).Search = origSearch;
  }
});

test('P2-2 eval edges: defensive fallbacks for empty/null inputs', async () => {
  resetRequestContext();
  const modelId = await resolveModelId('auth', 'CompanyScopedResource');
  const roleId = uid('role');
  const origSearch = (RoleRecordRule as any).Search;
  const origAppSearch = (IrApplication as any).Search;
  const origCount = (IrField as any).Count;
  const origGetReq = (globalThis as any).$choysum?.request;

  try {
    // Empty app/model names hit String(x || '') arms in the meta cache key.
    const env0 = await evaluateRecordRuleCondition({
      appName: '',
      modelName: '',
      hasCompany: false,
      opValue: 'read',
      roleIds: [],
      roleScopesById: {},
    });
    expect(env0.kind).toBe('false');
    expect(String(env0.reason || '')).toBe('model_not_found');

    // Empty role id entries filtered; null roleIds/roleScopes use || defaults.
    (RoleRecordRule as any).Search = async () => [
      {
        RoleId: { Id: roleId },
        Kind: null, // normalizeKind → grant
        Condition: { And: [['Name', '=', 'z']] },
        IrModelId: modelId,
        IrApplicationId: null,
      },
    ];
    const env = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: false,
      opValue: 'read',
      roleIds: null as any,
      roleScopesById: undefined as any,
    });
    // No roles + only RoleId-null audience in query, but mock returns role-scoped row → still accepted by loop.
    // With roleIds null, audience is everyone-only; mock still returns our row and eval uses it.
    expect(['expr', 'false']).toContain(env.kind);

    const env2 = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: ['', '  ', roleId],
      roleScopesById: { [roleId]: { global: false, companies: undefined as any } },
    });
    expect(env2.kind).toBe('expr');
    expect(JSON.stringify((env2 as any).expr || {})).toContain('CompanyId');

    // IrApplication empty ⇒ irApplicationId '' ⇒ skip app-scoped arm in scopeOr.
    (IrApplication as any).Search = async () => [];
    (RoleRecordRule as any).Search = async () => null; // allRules || []
    const env3 = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: false,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: {},
    });
    expect(env3.kind).toBe('false');
    expect(String(env3.reason || '')).toContain('no_grant');

    // Everyone RoleId-null hits ruleAudienceScope empty-roleId arm with company gate on.
    (RoleRecordRule as any).Search = async () => [
      {
        RoleId: null,
        Kind: 'grant',
        Condition: { And: [['Name', '=', 'everyone']] },
        IrModelId: modelId,
        IrApplicationId: null,
      },
    ];
    (IrField as any).Count = async () => 1;
    resetRequestContext();
    const envEveryone = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: [],
      roleScopesById: {},
    });
    expect(envEveryone.kind).toBe('expr');
    // Everyone scope is treated as global ⇒ no CompanyId gate clause.
    expect(JSON.stringify((envEveryone as any).expr || {})).not.toContain('CompanyId');

    // Missing request while company-scoped gate runs ⇒ req? undefined arm in computeCompanyGateMode.
    const root = (globalThis as any).$choysum;
    const prevRequest = root.request;
    root.request = undefined;
    (IrField as any).Count = async () => 1;
    (RoleRecordRule as any).Search = async () => [
      {
        RoleId: { Id: roleId },
        Kind: 'grant',
        Condition: { And: [['Name', '=', 'gated']] },
        IrModelId: modelId,
        IrApplicationId: null,
      },
    ];
    const env4 = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: [roleId],
      // global:true hits buildCompanyGateExpr early return when gate enabled.
      roleScopesById: { [roleId]: { global: true, companies: [] } },
    });
    expect(env4.kind).toBe('expr');
    expect(JSON.stringify((env4 as any).expr || {})).not.toContain('CompanyId');
    root.request = prevRequest;
  } finally {
    (RoleRecordRule as any).Search = origSearch;
    (IrApplication as any).Search = origAppSearch;
    (IrField as any).Count = origCount;
    if (origGetReq !== undefined) {
      (globalThis as any).$choysum.request = origGetReq;
    }
  }
});

test('P2-2 eval edges: companyField gate fail-closes when ownership field missing or meta errors', async () => {
  resetRequestContext();
  const modelId = await resolveModelId('auth', 'CompanyScopedResource');
  const roleId = uid('role');
  const origSearch = (RoleRecordRule as any).Search;
  const origModelSearch = (IrModel as any).Search;
  const origCount = (IrField as any).Count;

  try {
    (RoleRecordRule as any).Search = async () => [
      {
        RoleId: { Id: roleId },
        Kind: 'grant',
        Condition: { And: [['Name', '=', 'gated']] },
        IrModelId: modelId,
        IrApplicationId: null,
      },
    ];

    // Empty CompanyField ⇒ model_not_company_isolated (gate off, grant still applies).
    resetRequestContext();
    (IrModel as any).Search = async () => [{ Id: modelId, CompanyField: null }];
    (IrField as any).Count = async () => 1;
    const notIsolated = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: false, companies: ['company_a'] } },
    });
    expect(notIsolated.kind).toBe('expr');
    expect(JSON.stringify((notIsolated as any).expr || {})).not.toContain('CompanyId');

    // CompanyField set but IrField missing ⇒ fail-closed deny.
    resetRequestContext();
    (IrModel as any).Search = async () => [{ Id: modelId, CompanyField: 'CompanyId' }];
    (IrField as any).Count = async () => 0;
    const missingField = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: false, companies: ['company_a'] } },
    });
    expect(missingField.kind).toBe('false');
    expect(String((missingField as any).reason || '')).toBe('company_isolated_missing_ownership_field');

    // IrField.Count throws ⇒ fail-closed deny.
    resetRequestContext();
    (IrModel as any).Search = async () => [{ Id: modelId, CompanyField: 'CompanyId' }];
    (IrField as any).Count = async () => {
      throw new Error('meta boom');
    };
    const gateError = await evaluateRecordRuleCondition({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      hasCompany: true,
      opValue: 'read',
      roleIds: [roleId],
      roleScopesById: { [roleId]: { global: false, companies: ['company_a'] } },
    });
    expect(gateError.kind).toBe('false');
    expect(String((gateError as any).reason || '')).toBe('meta_company_gate_error');
  } finally {
    (RoleRecordRule as any).Search = origSearch;
    (IrModel as any).Search = origModelSearch;
    (IrField as any).Count = origCount;
  }
});

test('buildCompanyGateExpr covers disabled, empty ownership, and global scope', () => {
  expect(buildCompanyGateExpr({ global: false, companies: ['c1'] }, { enabled: false })).toBeNull();
  expect(buildCompanyGateExpr({ global: false, companies: ['c1'] }, { enabled: true, ownershipField: '  ' })).toBeNull();
  expect(buildCompanyGateExpr({ global: false, companies: ['c1'] }, { enabled: true })).toBeNull();
  expect(buildCompanyGateExpr({ global: true, companies: ['c1'] }, { enabled: true, ownershipField: 'CompanyId' })).toBeNull();
  expect(buildCompanyGateExpr({ global: false, companies: ['c1'] }, { enabled: true, ownershipField: 'CompanyId' })).toEqual({
    Or: [
      ['CompanyId', 'in', ['c1']],
      ['CompanyId', 'is', null],
    ],
  });
  expect(buildCompanyGateExpr({ global: false } as any, { enabled: true, ownershipField: 'OwningCompanyId' })).toEqual({
    Or: [
      ['OwningCompanyId', 'in', []],
      ['OwningCompanyId', 'is', null],
    ],
  });
});
