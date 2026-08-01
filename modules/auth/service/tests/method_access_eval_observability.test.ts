// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { evaluateRoleMethodAccess } from '@/auth/service/models/_user_method_access';
import RoleMethodAccess from '@/auth/service/models/role_method_access';

test('evaluateRoleMethodAccess returns deny allow and empty diagnostics with hitRuleIds', async () => {
  const orig = (RoleMethodAccess as any).Search;

  try {
    (RoleMethodAccess as any).Search = async () => [{ Id: 'ma_deny', Mode: 'deny' }, { Id: 'ma_allow', Mode: 'allow' }];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['MetaModelId', '=', 'm1']] as any])).toEqual({
      denied: true,
      allowed: false,
      hitRuleIds: ['ma_deny'],
      reason: 'method_access_deny',
    });

    // Allow rows before deny must not leak into deny diagnostics.
    (RoleMethodAccess as any).Search = async () => [
      { Id: 'ma_allow_first', Mode: 'allow' },
      { Id: 'ma_deny_wins', Mode: 'deny' },
      { Id: 'ma_allow_after', Mode: 'allow' },
    ];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['MetaModelId', '=', 'm1']] as any])).toEqual({
      denied: true,
      allowed: false,
      hitRuleIds: ['ma_deny_wins'],
      reason: 'method_access_deny',
    });

    (RoleMethodAccess as any).Search = async () => [
      { Id: 'ma_2', Mode: 'allow' },
      { Id: '', Mode: 'allow' },
      { Id: 'ma_1', Mode: 'allow' },
      { Mode: 'other' },
    ];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['MetaModelId', '=', 'm1']] as any])).toEqual({
      denied: false,
      allowed: true,
      hitRuleIds: ['ma_1', 'ma_2'],
      reason: 'method_access_allow',
    });

    (RoleMethodAccess as any).Search = async () => [{ Id: null, Mode: 'allow' }, { Mode: 'deny' }];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['MetaModelId', '=', 'm1']] as any])).toEqual({
      denied: true,
      allowed: false,
      hitRuleIds: [],
      reason: 'method_access_deny',
    });

    (RoleMethodAccess as any).Search = async () => [];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['MetaModelId', '=', 'm1']] as any])).toEqual({
      denied: false,
      allowed: false,
      hitRuleIds: [],
      reason: 'method_access_no_manual_rule',
    });

    (RoleMethodAccess as any).Search = async () => null;
    expect(await evaluateRoleMethodAccess(['role_1'], [[['MetaModelId', '=', 'm1']] as any])).toEqual({
      denied: false,
      allowed: false,
      hitRuleIds: [],
      reason: 'method_access_no_manual_rule',
    });

    // UI-Option-A: Source=ui rows are ignored by the manual ACL evaluator.
    (RoleMethodAccess as any).Search = async () => [
      { Id: 'ma_ui_only', Mode: 'allow', Source: 'ui' },
      { Id: 'ma_manual', Mode: 'allow', Source: 'manual' },
    ];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['MetaModelId', '=', 'm1']] as any])).toEqual({
      denied: false,
      allowed: true,
      hitRuleIds: ['ma_manual'],
      reason: 'method_access_allow',
    });

    (RoleMethodAccess as any).Search = async () => [{ Id: 'ma_ui_deny', Mode: 'deny', Source: 'ui' }];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['MetaModelId', '=', 'm1']] as any])).toEqual({
      denied: false,
      allowed: false,
      hitRuleIds: [],
      reason: 'method_access_no_manual_rule',
    });
  } finally {
    (RoleMethodAccess as any).Search = orig;
  }
});

test('evaluateUiDerivedMethodDecision returns reason and hitRuleIds', async () => {
  const { evaluateUiDerivedMethodDecision } = await import('@/auth/service/models/_user_method_access');
  const RoleUiResource = (await import('@/auth/service/models/role_ui_resource')).default;
  const MetaUiResource = (await import('@/meta/service/models/ui_resource')).default;

  // Isolate from sibling tests that may have warmed UI-grant request caches.
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  root.request.context = { ctx: {}, req: { depth: 0 }, identity: {} };
  (globalThis as any).$choysum = root;

  const originalRoleUiSearch = (RoleUiResource as any).Search;
  const originalIrUiSearch = (MetaUiResource as any).Search;

  (RoleUiResource as any).Search = async () => [
    { MetaApplicationId: null, MetaUiResourceId: null, Mode: 'allow' },
    { MetaApplicationId: null, MetaUiResourceId: 'RES-E5-ALLOW', Mode: 'allow' },
  ];
  (MetaUiResource as any).Search = async () => [
    {
      Id: 'RES-E5-ALLOW',
      Name: 'res-e5-allow',
      MetaApplicationId: 'APP-E5',
      Requires: ['rpc:/auth.User/browse'],
    },
  ];

  try {
    const allowed = await evaluateUiDerivedMethodDecision(['ROLE-E5-ALLOW'], 'auth.User', 'browse');
    expect(allowed).toEqual({
      allowed: true,
      denied: false,
      hitRuleIds: ['RES-E5-ALLOW'],
      reason: 'method_access_ui_allow',
    });

    const empty = await evaluateUiDerivedMethodDecision(['ROLE-E5-ALLOW'], 'auth.User', 'missing');
    expect(empty.reason).toBe('method_access_ui_no_match');
    expect(empty.hitRuleIds).toEqual([]);
  } finally {
    (RoleUiResource as any).Search = originalRoleUiSearch;
    (MetaUiResource as any).Search = originalIrUiSearch;
  }
});
