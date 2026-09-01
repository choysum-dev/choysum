// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { evaluateFieldRules } from '@/auth/service/models/user/_field_rule_eval';
import RoleFieldRule from '@/auth/service/models/role_field_rule';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
import type MetaFieldModel from '@/meta/service/models/field';
import type MetaModelModel from '@/meta/service/models/model';

const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');
const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');
const MetaField = createServiceByModel<typeof MetaFieldModel>('meta.MetaField');

function ensureRequestContext(): void {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  // Fresh ctx/req each call so memoizeInReqState does not leak across tests.
  jsCtx.ctx = {};
  jsCtx.req = { depth: 0 };
  if (!jsCtx.identity) jsCtx.identity = {};
  (globalThis as any).$choysum = root;
}

test('evaluateFieldRules throws when effective model missing', async () => {
  ensureRequestContext();
  const origModel = (MetaModel as any).Search;
  const origApp = (MetaApplication as any).Search;
  try {
    (MetaModel as any).Search = async () => [];
    (MetaApplication as any).Search = async () => [{ Id: 'app-1' }];
    let caught: any;
    try {
      await evaluateFieldRules({
        appName: 'auth-missing-model',
        modelName: 'Missing',
        modelFullName: 'auth.Missing',
        roleIds: ['r1'],
      });
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeTruthy();
    expect(caught instanceof ChoysumError).toBe(true);
    expect(String((caught as any)?.code || '')).toBe('VALIDATION_FAILED');
    expect(String((caught as any)?.message || '').includes('Model does not exist')).toBe(true);
  } finally {
    (MetaModel as any).Search = origModel;
    (MetaApplication as any).Search = origApp;
  }
});

test('evaluateFieldRules skips app-scope Or when applicationId empty and drops mismatched model rules', async () => {
  ensureRequestContext();
  const origModel = (MetaModel as any).Search;
  const origApp = (MetaApplication as any).Search;
  const origField = (MetaField as any).Search;
  const origRules = (RoleFieldRule as any).Search;

  try {
    (MetaModel as any).Search = async () => [{ Id: 'model-1', ModuleId: null, UpdatedAt: '2026-08-05T12:00:00.000Z' }];
    (MetaApplication as any).Search = async () => []; // empty applicationId
    (MetaField as any).Search = async () => [
      { Id: 'f1', Name: 'Login' },
      { Id: 'f2', Name: 'PasswordHash' },
    ];

    let capturedDomain: any;
    (RoleFieldRule as any).Search = async (domain: any) => {
      capturedDomain = domain;
      return [
        // Field rule for wrong model — should be skipped by irModel !== modelId.
        {
          Id: 'rule-mismatch-field',
          MetaModelId: 'other-model',
          MetaFieldId: 'f1',
          MetaApplicationId: null,
          PermRead: 'deny',
          PermWrite: 'deny',
        },
        // Model rule for wrong model — skipped.
        {
          Id: 'rule-mismatch-model',
          MetaModelId: 'other-model',
          MetaFieldId: null,
          MetaApplicationId: null,
          PermRead: 'allow',
          PermWrite: 'allow',
        },
        // App rule with non-empty irApp while applicationId is empty — skipped.
        {
          Id: 'rule-app',
          MetaModelId: null,
          MetaFieldId: null,
          MetaApplicationId: 'app-ghost',
          PermRead: 'allow',
          PermWrite: 'allow',
        },
        // Matching model allow so evaluation completes.
        {
          Id: 'rule-ok',
          MetaModelId: 'model-1',
          MetaFieldId: null,
          MetaApplicationId: null,
          PermRead: 'allow',
          PermWrite: 'allow',
        },
      ];
    };

    const out = await evaluateFieldRules({
      appName: 'no-such-meta-app-for-field-rule',
      modelName: 'User',
      modelFullName: 'auth.User',
      roleIds: ['r1'],
    });

    // App-scoped Or branch omitted when applicationId is empty.
    const orBranches = capturedDomain?.And?.[1]?.Or || [];
    const hasAppBranch = orBranches.some((b: any) => {
      const and = b?.And || [];
      return and.some((c: any) => Array.isArray(c) && c[0] === 'MetaApplicationId' && c[1] === '=');
    });
    expect(hasAppBranch).toBe(false);

    expect(out.denyReadFields).toEqual([]);
    expect(out.hitRuleIds || []).toContain('rule-ok');
    expect(out.hitRuleIds || []).not.toContain('rule-mismatch-field');
    expect(out.hitRuleIds || []).not.toContain('rule-app');
  } finally {
    (MetaModel as any).Search = origModel;
    (MetaApplication as any).Search = origApp;
    (MetaField as any).Search = origField;
    (RoleFieldRule as any).Search = origRules;
  }
});

test('evaluateFieldRules app rule continues when irApp mismatches resolved applicationId', async () => {
  ensureRequestContext();
  const origModel = (MetaModel as any).Search;
  const origApp = (MetaApplication as any).Search;
  const origField = (MetaField as any).Search;
  const origRules = (RoleFieldRule as any).Search;

  try {
    (MetaModel as any).Search = async () => [{ Id: 'model-1', ModuleId: null, UpdatedAt: '2026-08-05T12:00:00.000Z' }];
    (MetaApplication as any).Search = async () => [{ Id: 'app-1' }];
    (MetaField as any).Search = async () => [{ Id: 'f1', Name: 'Login' }];
    (RoleFieldRule as any).Search = async () => [
      {
        Id: 'rule-wrong-app',
        MetaModelId: null,
        MetaFieldId: null,
        MetaApplicationId: 'app-other',
        PermRead: 'deny',
        PermWrite: 'deny',
      },
      {
        Id: 'rule-global',
        MetaModelId: null,
        MetaFieldId: null,
        MetaApplicationId: null,
        PermRead: 'allow',
        PermWrite: 'allow',
      },
    ];

    const out = await evaluateFieldRules({
      appName: 'auth-app-mismatch-fr',
      modelName: 'User',
      modelFullName: 'auth.User',
      roleIds: ['r1'],
    });
    expect(out.denyReadFields).toEqual([]);
    expect(out.hitRuleIds || []).toContain('rule-global');
    expect(out.hitRuleIds || []).not.toContain('rule-wrong-app');
  } finally {
    (MetaModel as any).Search = origModel;
    (MetaApplication as any).Search = origApp;
    (MetaField as any).Search = origField;
    (RoleFieldRule as any).Search = origRules;
  }
});
