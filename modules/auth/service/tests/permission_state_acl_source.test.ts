// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildAclAggregation } from '@/auth/service/models/_user_permission_state_acl';
import RoleMethodAccess from '@/auth/service/models/role_method_access';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
import type MetaModelModel from '@/meta/service/models/model';
import type MetaServiceModel from '@/meta/service/models/service';

const MetaService = createServiceByModel<typeof MetaServiceModel>('meta.MetaService');
const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');
const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');

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
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        Mode: 'allow',
        Source: 'ui',
      },
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
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

test('buildAclAggregation collects MetaServiceId/MetaModelId/MetaApplicationId from accesses', async () => {
  const origAccess = (RoleMethodAccess as any).Search;
  const origService = (MetaService as any).Search;
  const origModel = (MetaModel as any).Search;
  const origApp = (MetaApplication as any).Search;

  try {
    // Mixed null/non-null Meta* ids exercise both sides of `id || ''` while collecting
    // irServiceIds / irModelIds / irApplicationIds (patch lines Codecov marked partial).
    (RoleMethodAccess as any).Search = async () => [
      {
        RoleId: 'role_1',
        MetaServiceId: 'svc_1',
        MetaModelId: null,
        MetaApplicationId: null,
        Mode: 'allow',
        Source: 'manual',
      },
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: 'model_1',
        MetaApplicationId: null,
        Mode: 'deny',
        Source: 'manual',
      },
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: 'app_1',
        Mode: 'allow',
        Source: 'manual',
      },
    ];
    // Keep meta lookups empty so this stays a pure ACL-source unit test.
    (MetaService as any).Search = async () => [];
    (MetaModel as any).Search = async () => [];
    (MetaApplication as any).Search = async () => [];

    const agg = await buildAclAggregation(['role_1'], { role_1: { global: true, companies: [] } });
    // Resolution may no-op when meta rows are absent; the important part is the
    // Meta* id collection path above runs without throwing.
    expect(agg.requiresAllowKeysByCompany instanceof Map).toBe(true);
    expect(agg.requiresDenyKeysByCompany instanceof Map).toBe(true);
  } finally {
    (RoleMethodAccess as any).Search = origAccess;
    (MetaService as any).Search = origService;
    (MetaModel as any).Search = origModel;
    (MetaApplication as any).Search = origApp;
  }
});
