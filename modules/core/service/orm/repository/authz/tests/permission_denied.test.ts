// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode } from '@/core/service/error';
import { createRepositoryPermissionDeniedError, resolveRepositoryPermissionDeniedLayer } from '../permission_denied';

test('repository permission denied maps deny layer prefixes to authz layers', () => {
  expect(resolveRepositoryPermissionDeniedLayer('company_scope_violation')).toBe('company_filter');
  expect(resolveRepositoryPermissionDeniedLayer('record_rule_denied')).toBe('record_rule');
  expect(resolveRepositoryPermissionDeniedLayer('field_rule_readonly_violation')).toBe('field_rule');
  expect(resolveRepositoryPermissionDeniedLayer('something_else')).toBe('unknown');
});

test('repository permission denied builds PermissionDenied error and emits deny summary with request facts', () => {
  const summaries: Array<Record<string, any>> = [];

  const error = createRepositoryPermissionDeniedError(
    {
      meta: {
        fullModelName: 'auth.Role',
        modelName: 'Role',
        name: 'Role',
      } as any,
      userId: ' user_1 ',
      getReqMethodMeta() {
        return {
          fullMethod: '/auth.Role/UpdateById',
          method: 'UpdateById',
          companyMode: 'strict',
          recordRuleMode: 'default',
          fieldRuleMode: 'default',
        };
      },
      getCompanyScopeFacts() {
        return {
          activeCompanyId: 'company_a',
          enabledCompanyIds: ['company_a', 'company_b'],
        };
      },
      emitAuthzDecisionSummary(summary) {
        summaries.push(summary);
      },
    },
    'record_rule_denied',
    'record rule denied',
    { targetCount: '2' }
  );

  expect(error.domain).toBe('core.repository');
  expect(error.code).toBe('record_rule_denied');
  expect(error.message).toBe('record rule denied');
  expect(error.grpcCode).toBe(GrpcCode.PermissionDenied);
  expect(error.metadata).toEqual({ targetCount: '2' });

  expect(summaries).toEqual([
    {
      layer: 'record_rule',
      decision: 'deny',
      basis: 'record_rule_denied',
      fullMethod: '/auth.Role/UpdateById',
      method: 'UpdateById',
      model: 'auth.Role',
      userId: 'user_1',
      activeCompanyId: 'company_a',
      enabledCompanyIds: ['company_a', 'company_b'],
      companyMode: 'strict',
      recordRuleMode: 'default',
      fieldRuleMode: 'default',
      message: 'record rule denied',
      metadata: { targetCount: '2' },
    },
  ]);
});

test('repository permission denied uses empty metadata summary when metadata is omitted', () => {
  const summaries: Array<Record<string, any>> = [];

  const error = createRepositoryPermissionDeniedError(
    {
      meta: {
        fullModelName: 'demo.Model',
        modelName: 'Model',
        name: 'Model',
      } as any,
      userId: 'user_2',
      getReqMethodMeta() {
        return {
          fullMethod: '/demo.Model/Delete',
          method: 'Delete',
          companyMode: 'strict',
          recordRuleMode: 'default',
          fieldRuleMode: 'default',
        };
      },
      getCompanyScopeFacts() {
        return {
          activeCompanyId: 'company_a',
          enabledCompanyIds: ['company_a'],
        };
      },
      emitAuthzDecisionSummary(summary) {
        summaries.push(summary);
      },
    },
    'company_scope_violation',
    'company denied'
  );

  expect(error.code).toBe('company_scope_violation');
  expect(error.metadata).toEqual({});
  expect(summaries.length).toBe(1);
  expect(summaries[0]?.metadata).toEqual({});
  expect(summaries[0]?.layer).toBe('company_filter');
});

test('repository permission denied forwards metadata keyset to summary and error metadata', () => {
  const summaries: Array<Record<string, any>> = [];

  const error = createRepositoryPermissionDeniedError(
    {
      meta: {
        fullModelName: 'demo.Model',
        modelName: 'Model',
        name: 'Model',
      } as any,
      userId: 'user_3',
      getReqMethodMeta() {
        return {
          fullMethod: '/demo.Model/Create',
          method: 'Create',
          companyMode: 'strict',
          recordRuleMode: 'default',
          fieldRuleMode: 'default',
        };
      },
      getCompanyScopeFacts() {
        return {
          activeCompanyId: 'company_b',
          enabledCompanyIds: ['company_b'],
        };
      },
      emitAuthzDecisionSummary(summary) {
        summaries.push(summary);
      },
    },
    'record_rule_violation',
    'target set violates record rule',
    {
      targetCount: '3',
      allowedCount: '2',
      reason: 'policy',
    }
  );

  expect(error.code).toBe('record_rule_violation');
  expect(error.metadata).toEqual({
    targetCount: '3',
    allowedCount: '2',
    reason: 'policy',
  });
  expect(summaries.length).toBe(1);
  expect(Object.keys(summaries[0]?.metadata || {}).sort()).toEqual(['allowedCount', 'reason', 'targetCount']);
});

test('repository permission denied resolves model fallback from modelName to name', () => {
  const summaries: Array<Record<string, any>> = [];

  createRepositoryPermissionDeniedError(
    {
      meta: {
        fullModelName: '',
        modelName: 'RoleAlias',
        name: 'RoleName',
      } as any,
      userId: 'user_model_name',
      getReqMethodMeta() {
        return {
          fullMethod: '/demo.Model/Read',
          method: 'Read',
          companyMode: 'strict',
          recordRuleMode: 'default',
          fieldRuleMode: 'default',
        };
      },
      getCompanyScopeFacts() {
        return {
          activeCompanyId: 'company_a',
          enabledCompanyIds: ['company_a'],
        };
      },
      emitAuthzDecisionSummary(summary) {
        summaries.push(summary);
      },
    },
    'record_rule_denied',
    'denied by modelName fallback'
  );

  createRepositoryPermissionDeniedError(
    {
      meta: {
        fullModelName: '',
        modelName: '',
        name: 'RoleOnlyName',
      } as any,
      userId: 'user_name_only',
      getReqMethodMeta() {
        return {
          fullMethod: '/demo.Model/Read',
          method: 'Read',
          companyMode: 'strict',
          recordRuleMode: 'default',
          fieldRuleMode: 'default',
        };
      },
      getCompanyScopeFacts() {
        return {
          activeCompanyId: 'company_b',
          enabledCompanyIds: ['company_b'],
        };
      },
      emitAuthzDecisionSummary(summary) {
        summaries.push(summary);
      },
    },
    'record_rule_denied',
    'denied by name fallback'
  );

  expect(summaries).toHaveLength(2);
  expect(summaries[0]?.model).toBe('RoleAlias');
  expect(summaries[1]?.model).toBe('RoleOnlyName');
});
