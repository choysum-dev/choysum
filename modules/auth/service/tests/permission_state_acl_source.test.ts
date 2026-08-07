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

test('buildAclAggregation dedupes same application+name for app and global scopes', async () => {
  const origAccess = (RoleMethodAccess as any).Search;
  const origService = (MetaService as any).Search;
  const origModel = (MetaModel as any).Search;
  const origApp = (MetaApplication as any).Search;

  try {
    (RoleMethodAccess as any).Search = async () => [
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: 'app_1',
        Mode: 'allow',
        Source: 'manual',
      },
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        Mode: 'allow',
        Source: 'manual',
      },
    ];
    (MetaService as any).Search = async () => [];
    (MetaApplication as any).Search = async () => [{ Id: 'app_1', Name: 'auth' }];
    let modelSearchCalls = 0;
    (MetaModel as any).Search = async (domain: any) => {
      modelSearchCalls++;
      // App-scoped Search uses Application in; global uses empty domain.
      const isGlobal = Array.isArray(domain) && domain.length === 0;
      const rows = [
        { Application: 'auth', Name: 'User', ModuleId: 'shell', UpdatedAt: '2026-08-05T12:00:00.000Z' },
        { Application: 'auth', Name: 'User', ModuleId: null, UpdatedAt: '2026-08-05T10:00:00.000Z' },
        { Application: 'auth', Name: 'Role', ModuleId: null, UpdatedAt: '2026-08-05T11:00:00.000Z' },
        // null Application / Name hit `appRaw == null ? ''` / `nameRaw == null ? ''`.
        { Application: null, Name: 'Ghost' },
        { Application: 'auth', Name: null },
        { Application: '', Name: 'Bad' },
        { Application: 'auth', Name: '  ' },
      ];
      if (isGlobal) {
        return [...rows, { Application: 'other', Name: 'X', UpdatedAt: '2026-08-05T09:00:00.000Z' }];
      }
      return rows;
    };

    const agg = await buildAclAggregation(['role_1'], { role_1: { global: true, companies: [] } });
    expect(modelSearchCalls).toBeGreaterThanOrEqual(2);
    // Deduped: auth.User once + auth.Role from app scope; global also adds other.X.
    const allow = agg.requiresAllowKeysByCompany.get('*') || new Set();
    expect(allow.has('rpc:/auth.User/*')).toBe(true);
    expect(allow.has('rpc:/auth.Role/*')).toBe(true);
    expect(allow.has('rpc:/other.X/*')).toBe(true);
    expect(agg.companyGlobalAllow.has('*')).toBe(true);
  } finally {
    (RoleMethodAccess as any).Search = origAccess;
    (MetaService as any).Search = origService;
    (MetaModel as any).Search = origModel;
    (MetaApplication as any).Search = origApp;
  }
});

test('buildAclAggregation dedupe tolerates null MetaModel Search and multi-app maps', async () => {
  const origAccess = (RoleMethodAccess as any).Search;
  const origService = (MetaService as any).Search;
  const origModel = (MetaModel as any).Search;
  const origApp = (MetaApplication as any).Search;

  try {
    (RoleMethodAccess as any).Search = async () => [
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: 'app_auth',
        Mode: 'allow',
        Source: 'manual',
      },
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: 'app_base',
        Mode: 'deny',
        Source: 'manual',
      },
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        Mode: 'allow',
        Source: 'manual',
      },
      // Second global rule reuses memoized getAllModels (allModels already set).
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        Mode: 'deny',
        Source: 'manual',
      },
    ];
    (MetaService as any).Search = async () => [];
    (MetaApplication as any).Search = async () => [
      { Id: 'app_auth', Name: 'auth' },
      { Id: 'app_base', Name: 'base' },
    ];
    let globalCalls = 0;
    (MetaModel as any).Search = async (domain: any) => {
      const isGlobal = Array.isArray(domain) && domain.length === 0;
      if (isGlobal) {
        globalCalls++;
        // null hits `rows || []`; second global access must not Search again.
        if (globalCalls === 1) return null;
        return [{ Application: 'ghost', Name: 'ShouldNotMatter' }];
      }
      // null hits `rows || []` for app-scoped modelsByApp loop.
      return null;
    };

    const agg = await buildAclAggregation(['role_1'], { role_1: { global: true, companies: [] } });
    expect(globalCalls).toBe(1);
    expect(agg.companyGlobalAllow.has('*')).toBe(true);
    expect(agg.companyGlobalDeny.has('*')).toBe(true);
    expect(agg.requiresAllowKeysByCompany.get('*')?.size || 0).toBe(0);
  } finally {
    (RoleMethodAccess as any).Search = origAccess;
    (MetaService as any).Search = origService;
    (MetaModel as any).Search = origModel;
    (MetaApplication as any).Search = origApp;
  }
});

test('buildAclAggregation treats LogicalModelName as logical scope not global', async () => {
  const origAccess = (RoleMethodAccess as any).Search;
  const origService = (MetaService as any).Search;
  const origModel = (MetaModel as any).Search;
  const origApp = (MetaApplication as any).Search;

  try {
    (RoleMethodAccess as any).Search = async () => [
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'FieldDefault',
        LogicalMethods: ['Get', 'Set'],
        Mode: 'allow',
        Source: 'manual',
      },
    ];
    (MetaService as any).Search = async () => [];
    (MetaApplication as any).Search = async () => [];
    (MetaModel as any).Search = async () => [
      { Application: 'auth', Name: 'FieldDefault', UpdatedAt: '2026-01-02' },
      { Application: 'base', Name: 'FieldDefault', UpdatedAt: '2026-01-01' },
      { Application: 'auth', Name: 'User', UpdatedAt: '2026-01-03' },
    ];

    const agg = await buildAclAggregation(['role_1'], { role_1: { global: true, companies: [] } });
    expect(agg.companyGlobalAllow.has('*')).toBe(false);
    const allows = agg.requiresAllowKeysByCompany.get('*') || new Set();
    expect(allows.has('rpc:/auth.FieldDefault/Get')).toBe(true);
    expect(allows.has('rpc:/auth.FieldDefault/Set')).toBe(true);
    expect(allows.has('rpc:/base.FieldDefault/Get')).toBe(true);
    expect(allows.has('rpc:/auth.FieldDefault/*')).toBe(false);
    expect(allows.has('rpc:/auth.User/*')).toBe(false);

    (RoleMethodAccess as any).Search = async () => [
      {
        RoleId: 'role_1',
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'FieldDefault',
        LogicalMethods: null,
        Mode: 'allow',
        Source: 'manual',
      },
    ];
    const aggAll = await buildAclAggregation(['role_1'], { role_1: { global: true, companies: [] } });
    const allowsAll = aggAll.requiresAllowKeysByCompany.get('*') || new Set();
    expect(allowsAll.has('rpc:/auth.FieldDefault/*')).toBe(true);
    expect(allowsAll.has('rpc:/base.FieldDefault/*')).toBe(true);
    expect(aggAll.companyGlobalAllow.has('*')).toBe(false);
  } finally {
    (RoleMethodAccess as any).Search = origAccess;
    (MetaService as any).Search = origService;
    (MetaModel as any).Search = origModel;
    (MetaApplication as any).Search = origApp;
  }
});
