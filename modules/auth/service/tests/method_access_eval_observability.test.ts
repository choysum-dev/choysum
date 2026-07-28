// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { evaluateRoleMethodAccess } from '@/auth/service/models/_user_method_access';
import RoleMethodAccess from '@/auth/service/models/role_method_access';

test('evaluateRoleMethodAccess returns deny allow and empty diagnostics with hitRuleIds', async () => {
  const orig = (RoleMethodAccess as any).Search;

  try {
    (RoleMethodAccess as any).Search = async () => [{ Id: 'ma_deny', Mode: 'deny' }, { Id: 'ma_allow', Mode: 'allow' }];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['IrModelId', '=', 'm1']] as any])).toEqual({
      denied: true,
      allowed: false,
      hitRuleIds: ['ma_deny'],
      reason: 'method_access_deny',
    });

    (RoleMethodAccess as any).Search = async () => [
      { Id: 'ma_2', Mode: 'allow' },
      { Id: '', Mode: 'allow' },
      { Id: 'ma_1', Mode: 'allow' },
      { Mode: 'other' },
    ];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['IrModelId', '=', 'm1']] as any])).toEqual({
      denied: false,
      allowed: true,
      hitRuleIds: ['ma_1', 'ma_2'],
      reason: 'method_access_allow',
    });

    (RoleMethodAccess as any).Search = async () => [{ Id: null, Mode: 'allow' }, { Mode: 'deny' }];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['IrModelId', '=', 'm1']] as any])).toEqual({
      denied: true,
      allowed: false,
      hitRuleIds: [],
      reason: 'method_access_deny',
    });

    (RoleMethodAccess as any).Search = async () => [];
    expect(await evaluateRoleMethodAccess(['role_1'], [[['IrModelId', '=', 'm1']] as any])).toEqual({
      denied: false,
      allowed: false,
      hitRuleIds: [],
      reason: 'method_access_no_manual_rule',
    });

    (RoleMethodAccess as any).Search = async () => null;
    expect(await evaluateRoleMethodAccess(['role_1'], [[['IrModelId', '=', 'm1']] as any])).toEqual({
      denied: false,
      allowed: false,
      hitRuleIds: [],
      reason: 'method_access_no_manual_rule',
    });
  } finally {
    (RoleMethodAccess as any).Search = orig;
  }
});
