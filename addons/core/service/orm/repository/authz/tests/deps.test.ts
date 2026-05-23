// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createRepositoryAuthzContextDeps,
  createRepositoryCompanyScopeDeps,
  createRepositoryCompanyScopePolicyDeps,
  createRepositoryCompanyScopeQueryDeps,
  createRepositoryFieldRuleDeps,
  createRepositoryFieldRulePolicyDeps,
  createRepositoryFieldRuleSelectionDeps,
  createRepositoryRecordRuleCoordinatorDeps,
  createRepositoryRecordRuleCoordinatorPolicyDeps,
  createRepositoryRecordRuleDeps,
  createRepositoryRecordRulePolicyDeps,
} from '..';

test('repository authz deps delegate request/company/audit callbacks unchanged', () => {
  const summaries: Array<Record<string, any>> = [];
  let reqCalls = 0;
  let scopeCalls = 0;

  const deps = createRepositoryAuthzContextDeps({
    meta: { fullModelName: 'auth.Role' } as any,
    userId: 'user_1',
    getReqMethodMeta() {
      reqCalls += 1;
      return {
        fullMethod: '/auth.Role/Search',
        method: 'Search',
        companyMode: 'strict',
        recordRuleMode: 'default',
        fieldRuleMode: 'default',
      };
    },
    getCompanyScopeFacts() {
      scopeCalls += 1;
      return { activeCompanyId: 'company_a', enabledCompanyIds: ['company_a'] };
    },
    emitAuthzDecisionSummary(summary) {
      summaries.push(summary);
    },
  });

  expect(deps.meta).toEqual({ fullModelName: 'auth.Role' });
  expect(deps.userId).toBe('user_1');
  expect(deps.getReqMethodMeta()).toEqual({
    fullMethod: '/auth.Role/Search',
    method: 'Search',
    companyMode: 'strict',
    recordRuleMode: 'default',
    fieldRuleMode: 'default',
  });
  expect(deps.getCompanyScopeFacts()).toEqual({ activeCompanyId: 'company_a', enabledCompanyIds: ['company_a'] });
  deps.emitAuthzDecisionSummary({ layer: 'record_rule', decision: 'allow' });

  expect(reqCalls).toBe(1);
  expect(scopeCalls).toBe(1);
  expect(summaries).toEqual([{ layer: 'record_rule', decision: 'allow' }]);
});

test('repository authz deps delegate record rule deps unchanged', async () => {
  const calls = {
    companies: 0,
    companyId: 0,
    control: 0,
    enabled: 0,
    bypassDepth: 0,
    bypass: 0,
    denied: 0,
  };
  const permissionError = new Error('denied');

  const deps = createRepositoryRecordRuleDeps({
    meta: { fullModelName: 'auth.Role' } as any,
    userId: 'user_1',
    requestContext: { activeCompanyId: 'company_a' },
    normalizeCompanyIds() {
      calls.companies += 1;
      return ['company_a'];
    },
    normalizeCompanyIdForWrite() {
      calls.companyId += 1;
      return 'company_a';
    },
    isControlPlaneMetaModel() {
      calls.control += 1;
      return false;
    },
    recordRuleEnabled() {
      calls.enabled += 1;
      return true;
    },
    getRecordRuleBypassDepth() {
      calls.bypassDepth += 1;
      return 0;
    },
    async withRecordRuleBypass(fn) {
      calls.bypass += 1;
      return await fn();
    },
    permissionDenied() {
      calls.denied += 1;
      return permissionError;
    },
  });

  expect(deps.normalizeCompanyIds()).toEqual(['company_a']);
  expect(deps.normalizeCompanyIdForWrite()).toBe('company_a');
  expect(deps.isControlPlaneMetaModel()).toBe(false);
  expect(deps.recordRuleEnabled()).toBe(true);
  expect(deps.getRecordRuleBypassDepth()).toBe(0);
  expect(await deps.withRecordRuleBypass(async () => 'ok')).toBe('ok');
  expect(deps.permissionDenied('record_rule_denied', 'denied')).toBe(permissionError);
  expect(calls).toEqual({
    companies: 1,
    companyId: 1,
    control: 1,
    enabled: 1,
    bypassDepth: 1,
    bypass: 1,
    denied: 1,
  });
});

test('repository authz deps delegate record rule policy deps unchanged', async () => {
  const calls = {
    normalizeCompanies: 0,
    normalizeCompanyId: 0,
    control: 0,
    enabled: 0,
    bypassDepth: 0,
    bypass: 0,
    denied: 0,
  };
  const denied = new Error('denied');
  const deps = createRepositoryRecordRulePolicyDeps({
    meta: { fullModelName: 'auth.Role' } as any,
    userId: 'user_1',
    requestContext: { activeCompanyId: 'company_a' },
    normalizeCompanyIds() {
      calls.normalizeCompanies += 1;
      return ['company_a'];
    },
    normalizeCompanyIdForWrite() {
      calls.normalizeCompanyId += 1;
      return 'company_a';
    },
    isControlPlaneMetaModel() {
      calls.control += 1;
      return false;
    },
    recordRuleEnabled() {
      calls.enabled += 1;
      return true;
    },
    getRecordRuleBypassDepth() {
      calls.bypassDepth += 1;
      return 0;
    },
    async withRecordRuleBypass(fn) {
      calls.bypass += 1;
      return await fn();
    },
    permissionDenied() {
      calls.denied += 1;
      return denied;
    },
  });

  expect(deps.meta).toEqual({ fullModelName: 'auth.Role' });
  expect(deps.userId).toBe('user_1');
  expect(deps.requestContext).toEqual({ activeCompanyId: 'company_a' });
  expect(deps.normalizeCompanyIds()).toEqual(['company_a']);
  expect(deps.normalizeCompanyIdForWrite()).toBe('company_a');
  expect(deps.isControlPlaneMetaModel()).toBe(false);
  expect(deps.recordRuleEnabled()).toBe(true);
  expect(deps.getRecordRuleBypassDepth()).toBe(0);
  expect(await deps.withRecordRuleBypass(async () => 'ok')).toBe('ok');
  expect(deps.permissionDenied('record_rule_denied', 'denied')).toBe(denied);

  expect(calls).toEqual({
    normalizeCompanies: 1,
    normalizeCompanyId: 1,
    control: 1,
    enabled: 1,
    bypassDepth: 1,
    bypass: 1,
    denied: 1,
  });
});

test('repository authz deps delegate record rule coordinator deps unchanged', async () => {
  const summaries: Array<Record<string, any>> = [];
  const calls = {
    envelope: 0,
    replace: 0,
    req: 0,
    scope: 0,
    denied: 0,
    count: 0,
  };
  const permissionError = new Error('denied');

  const deps = createRepositoryRecordRuleCoordinatorDeps({
    meta: { fullModelName: 'auth.Role' } as any,
    userId: 'user_1',
    recordRuleEnabled() {
      return true;
    },
    async getRecordRuleEnvelope() {
      calls.envelope += 1;
      return { kind: 'true', reason: 'ok' } as any;
    },
    replaceRecordRuleTokens(condition) {
      calls.replace += 1;
      return { And: [condition, ['Id', '!=', '0'] as any] } as any;
    },
    getReqMethodMeta() {
      calls.req += 1;
      return {
        fullMethod: '/auth.Role/Search',
        method: 'Search',
        companyMode: 'strict',
        recordRuleMode: 'default',
        fieldRuleMode: 'default',
      };
    },
    getCompanyScopeFacts() {
      calls.scope += 1;
      return { activeCompanyId: 'company_a', enabledCompanyIds: ['company_a'] };
    },
    emitAuthzDecisionSummary(summary) {
      summaries.push(summary);
    },
    permissionDenied() {
      calls.denied += 1;
      return permissionError;
    },
    async countConditionMatches() {
      calls.count += 1;
      return 2;
    },
  });

  expect(await deps.getRecordRuleEnvelope('read')).toEqual({ kind: 'true', reason: 'ok' });
  expect(deps.replaceRecordRuleTokens(['Status', '=', 'ready'] as any)).toEqual({
    And: [
      ['Status', '=', 'ready'],
      ['Id', '!=', '0'],
    ],
  });
  expect(deps.getReqMethodMeta()).toEqual({
    fullMethod: '/auth.Role/Search',
    method: 'Search',
    companyMode: 'strict',
    recordRuleMode: 'default',
    fieldRuleMode: 'default',
  });
  expect(deps.getCompanyScopeFacts()).toEqual({ activeCompanyId: 'company_a', enabledCompanyIds: ['company_a'] });
  deps.emitAuthzDecisionSummary({ layer: 'record_rule', decision: 'allow' });
  expect(deps.permissionDenied('record_rule_denied', 'denied')).toBe(permissionError);
  expect(await deps.countConditionMatches(['Status', '=', 'ready'] as any)).toBe(2);
  expect(calls).toEqual({
    envelope: 1,
    replace: 1,
    req: 1,
    scope: 1,
    denied: 1,
    count: 1,
  });
  expect(summaries).toEqual([{ layer: 'record_rule', decision: 'allow' }]);
});

test('repository authz deps delegate record rule coordinator policy deps unchanged', async () => {
  const summaries: Array<Record<string, any>> = [];
  const denied = new Error('denied');
  const deps = createRepositoryRecordRuleCoordinatorPolicyDeps({
    meta: { fullModelName: 'auth.Role' } as any,
    userId: 'user_1',
    getReqMethodMeta() {
      return {
        fullMethod: '/auth.Role/Search',
        method: 'Search',
        companyMode: 'strict',
        recordRuleMode: 'default',
        fieldRuleMode: 'default',
      };
    },
    getCompanyScopeFacts() {
      return { activeCompanyId: 'company_a', enabledCompanyIds: ['company_a'] };
    },
    emitAuthzDecisionSummary(summary) {
      summaries.push(summary);
    },
    recordRuleEnabled() {
      return true;
    },
    async getRecordRuleEnvelope() {
      return { kind: 'true', reason: 'ok' } as any;
    },
    replaceRecordRuleTokens(condition) {
      return condition;
    },
    permissionDenied() {
      return denied;
    },
    async countConditionMatches() {
      return 1;
    },
  });

  expect(await deps.getRecordRuleEnvelope('read')).toEqual({ kind: 'true', reason: 'ok' });
  expect(deps.replaceRecordRuleTokens(['Id', '=', '1'] as any)).toEqual(['Id', '=', '1']);
  expect(deps.permissionDenied('record_rule_denied', 'denied')).toBe(denied);
  expect(await deps.countConditionMatches(['Id', '=', '1'] as any)).toBe(1);
  deps.emitAuthzDecisionSummary({ layer: 'record_rule', decision: 'allow' });
  expect(summaries).toEqual([{ layer: 'record_rule', decision: 'allow' }]);
});

test('repository authz deps delegate field rule deps unchanged', async () => {
  const calls = {
    normalize: 0,
    control: 0,
    fieldControl: 0,
    recordBypass: 0,
    fieldBypass: 0,
    denied: 0,
  };
  const permissionError = new Error('denied');
  const deps = createRepositoryFieldRuleDeps({
    meta: { fullModelName: 'auth.Role' } as any,
    userId: 'user_1',
    requestContext: { activeCompanyId: 'company_a' },
    normalizeCompanyIds() {
      calls.normalize += 1;
      return ['company_a'];
    },
    isControlPlaneMetaModel() {
      calls.control += 1;
      return false;
    },
    isFieldRuleControlPlaneModel() {
      calls.fieldControl += 1;
      return true;
    },
    async withRecordRuleBypass(fn) {
      calls.recordBypass += 1;
      return await fn();
    },
    async withFieldRuleBypass(fn) {
      calls.fieldBypass += 1;
      return await fn();
    },
    permissionDenied() {
      calls.denied += 1;
      return permissionError;
    },
  });

  expect(deps.normalizeCompanyIds()).toEqual(['company_a']);
  expect(deps.isControlPlaneMetaModel()).toBe(false);
  expect(deps.isFieldRuleControlPlaneModel()).toBe(true);
  expect(await deps.withRecordRuleBypass(async () => 'rr')).toBe('rr');
  expect(await deps.withFieldRuleBypass(async () => 'fr')).toBe('fr');
  expect(deps.permissionDenied('field_rule_fetch_failed', 'failed')).toBe(permissionError);

  expect(calls).toEqual({
    normalize: 1,
    control: 1,
    fieldControl: 1,
    recordBypass: 1,
    fieldBypass: 1,
    denied: 1,
  });
});

test('repository authz deps delegate field rule policy and selection deps unchanged', async () => {
  const summaries: Array<Record<string, any>> = [];
  const denied = new Error('denied');
  const fieldDeps = createRepositoryFieldRulePolicyDeps({
    meta: { fullModelName: 'auth.Role' } as any,
    userId: 'user_1',
    requestContext: { activeCompanyId: 'company_a' },
    normalizeCompanyIds() {
      return ['company_a'];
    },
    isControlPlaneMetaModel() {
      return false;
    },
    isFieldRuleControlPlaneModel() {
      return true;
    },
    async withRecordRuleBypass(fn) {
      return await fn();
    },
    async withFieldRuleBypass(fn) {
      return await fn();
    },
    permissionDenied() {
      return denied;
    },
  });
  const selectionDeps = createRepositoryFieldRuleSelectionDeps({
    isControlPlaneMetaModel() {
      return true;
    },
  });

  expect(fieldDeps.normalizeCompanyIds()).toEqual(['company_a']);
  expect(fieldDeps.isControlPlaneMetaModel()).toBe(false);
  expect(fieldDeps.isFieldRuleControlPlaneModel()).toBe(true);
  expect(await fieldDeps.withRecordRuleBypass(async () => 'rr')).toBe('rr');
  expect(await fieldDeps.withFieldRuleBypass(async () => 'fr')).toBe('fr');
  expect(fieldDeps.permissionDenied('field_rule_denied', 'denied')).toBe(denied);
  expect(selectionDeps.isControlPlaneMetaModel()).toBe(true);
  expect(summaries).toEqual([]);
});

test('repository authz deps delegate company scope deps unchanged', async () => {
  const summaries: Array<Record<string, any>> = [];
  const permissionError = new Error('denied');
  const calls = {
    skipped: 0,
    req: 0,
    scope: 0,
    denied: 0,
  };

  const deps = createRepositoryCompanyScopeDeps({
    meta: { fullModelName: 'auth.Role' } as any,
    ctx: { activeCompanyId: 'company_a' },
    userId: 'user_1',
    companyLayerSkipped() {
      calls.skipped += 1;
      return true;
    },
    getReqMethodMeta() {
      calls.req += 1;
      return {
        fullMethod: '/auth.Role/Search',
        method: 'Search',
        companyMode: 'strict',
        recordRuleMode: 'default',
        fieldRuleMode: 'default',
      };
    },
    getCompanyScopeFacts() {
      calls.scope += 1;
      return { activeCompanyId: 'company_a', enabledCompanyIds: ['company_a'] };
    },
    emitAuthzDecisionSummary(summary) {
      summaries.push(summary);
    },
    permissionDenied() {
      calls.denied += 1;
      return permissionError;
    },
  });

  expect(deps.companyLayerSkipped()).toBe(true);
  expect(deps.getReqMethodMeta()).toEqual({
    fullMethod: '/auth.Role/Search',
    method: 'Search',
    companyMode: 'strict',
    recordRuleMode: 'default',
    fieldRuleMode: 'default',
  });
  expect(deps.getCompanyScopeFacts()).toEqual({ activeCompanyId: 'company_a', enabledCompanyIds: ['company_a'] });
  deps.emitAuthzDecisionSummary({ layer: 'company_filter', decision: 'allow' });
  expect(deps.permissionDenied('company_scope_violation', 'denied')).toBe(permissionError);

  expect(calls).toEqual({ skipped: 1, req: 1, scope: 1, denied: 1 });
  expect(summaries).toEqual([{ layer: 'company_filter', decision: 'allow' }]);
});

test('repository authz deps delegate company scope policy and query deps unchanged', async () => {
  const summaries: Array<Record<string, any>> = [];
  const denied = new Error('denied');
  const companyDeps = createRepositoryCompanyScopePolicyDeps({
    meta: { fullModelName: 'auth.Role' } as any,
    userId: 'user_1',
    getReqMethodMeta() {
      return {
        fullMethod: '/auth.Role/Search',
        method: 'Search',
        companyMode: 'strict',
        recordRuleMode: 'default',
        fieldRuleMode: 'default',
      };
    },
    getCompanyScopeFacts() {
      return { activeCompanyId: 'company_a', enabledCompanyIds: ['company_a'] };
    },
    emitAuthzDecisionSummary(summary) {
      summaries.push(summary);
    },
    ctx: { activeCompanyId: 'company_a' },
    companyLayerSkipped() {
      return false;
    },
    permissionDenied() {
      return denied;
    },
  });

  const executed: any[] = [];
  const query = { kind: 'select' };
  const queryDeps = createRepositoryCompanyScopeQueryDeps({
    meta: { fullModelName: 'auth.Role' } as any,
    ctx: { activeCompanyId: 'company_a' },
    userId: 'user_1',
    companyLayerSkipped() {
      return false;
    },
    getReqMethodMeta() {
      return {
        fullMethod: '/auth.Role/Update',
        method: 'Update',
        companyMode: 'strict',
        recordRuleMode: 'default',
        fieldRuleMode: 'default',
      };
    },
    getCompanyScopeFacts() {
      return { activeCompanyId: 'company_a', enabledCompanyIds: ['company_a'] };
    },
    emitAuthzDecisionSummary() {},
    permissionDenied(code, message) {
      return new Error(`${code}:${message}`);
    },
    db: { name: 'db' },
    table: 'demo_table',
    applySoftLayer(condition) {
      return { And: [condition, ['DeletedAt', 'is', null] as any] } as any;
    },
    isEmptyCondition(condition) {
      return Array.isArray(condition) && condition.length === 0;
    },
    convertCondition(eb, condition, selfTable) {
      return { eb, condition, selfTable };
    },
    async execute(input) {
      executed.push(input);
      return [{ Id: 'row_1' }] as any;
    },
  });

  expect(companyDeps.ctx).toEqual({ activeCompanyId: 'company_a' });
  expect(companyDeps.companyLayerSkipped()).toBe(false);
  companyDeps.emitAuthzDecisionSummary({ layer: 'company_filter', decision: 'allow' });
  expect(summaries).toEqual([{ layer: 'company_filter', decision: 'allow' }]);

  expect(queryDeps.db).toEqual({ name: 'db' });
  expect(queryDeps.table).toBe('demo_table');
  expect(queryDeps.applySoftLayer(['Status', '=', 'ready'] as any)).toEqual({
    And: [
      ['Status', '=', 'ready'],
      ['DeletedAt', 'is', null],
    ],
  });
  expect(queryDeps.isEmptyCondition([] as any)).toBe(true);
  expect(queryDeps.convertCondition('EB', ['Status', '=', 'ready'] as any, 'demo_table')).toEqual({
    eb: 'EB',
    condition: ['Status', '=', 'ready'],
    selfTable: 'demo_table',
  });
  await queryDeps.execute(query);
  expect(executed).toEqual([query]);
});
