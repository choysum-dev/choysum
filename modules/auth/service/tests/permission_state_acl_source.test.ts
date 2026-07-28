// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildAclAggregation } from '@/auth/service/models/_user_permission_state_acl';
import RoleMethodAccess from '@/auth/service/models/role_method_access';

test('buildAclAggregation ignores Source=ui and tolerates null Search (UI-Option-A)', async () => {
  const orig = (RoleMethodAccess as any).Search;

  try {
    // Null Search result hits `accessesRaw || []`.
    (RoleMethodAccess as any).Search = async () => null;
    const empty = await buildAclAggregation(['role_1'], { role_1: { global: true, companies: [] } });
    expect(empty.companyGlobalAllow.size).toBe(0);
    expect(empty.companyGlobalDeny.size).toBe(0);

    // Source=ui global allow must not enter PermissionState ACL aggregation.
    // Missing Source defaults to manual and is kept.
    (RoleMethodAccess as any).Search = async () => [
      {
        RoleId: 'role_1',
        IrServiceId: null,
        IrModelId: null,
        IrApplicationId: null,
        Mode: 'allow',
        Source: 'ui',
      },
      {
        RoleId: 'role_1',
        IrServiceId: null,
        IrModelId: null,
        IrApplicationId: null,
        Mode: 'deny',
        // Source omitted → treated as manual
      },
    ];
    const filtered = await buildAclAggregation(['role_1'], { role_1: { global: true, companies: [] } });
    expect(filtered.companyGlobalAllow.has('*')).toBe(false);
    expect(filtered.companyGlobalDeny.has('*')).toBe(true);
  } finally {
    (RoleMethodAccess as any).Search = orig;
  }
});
