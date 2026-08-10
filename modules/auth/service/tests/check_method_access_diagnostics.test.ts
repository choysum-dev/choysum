// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getCurrentReq, getOrInitReqServiceState, withContext as withModelContext } from '@/core/service/api/context';
import User from '@/auth/service/models/user';
import Role from '@/auth/service/models/role';
import UserRole from '@/auth/service/models/user_role';
import RoleMethodAccess from '@/auth/service/models/role_method_access';
import RoleUiResource from '@/auth/service/models/role_ui_resource';
import { evaluateUiDerivedMethodDecision } from '@/auth/service/models/_user_method_access';
import { buildMethodAccessCacheKey } from '@/auth/service/models/_request_cache_invalidation';
import { metaModelId } from './_meta_ids';
import MetaUiResource from '@/meta/service/models/ui_resource';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaServiceModel from '@/meta/service/models/service';

const MetaService = createServiceByModel<typeof MetaServiceModel>('meta.MetaService');

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
  jsCtx.req = { depth: 0, fieldRuleMode: 'skip', recordRuleMode: 'skip' };
  jsCtx.identity = {};
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function setIdentity(userId: string): void {
  const jsCtx = ensureRequestContext();
  jsCtx.identity = { userId, tokenId: 'tok', userVersion: 1, permStateVersion: 1 };
}

function setReq(patch: Record<string, any>): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.req) jsCtx.req = {};
  Object.assign(jsCtx.req, patch);
}

function setupAllowlistForFixtures(): void {
  setReq({
    depth: 0,
    recordRuleMode: 'allowlist',
    fieldRuleMode: 'skip',
    recordRuleAllow: [
      'auth.User:read',
      'auth.User:write',
      'auth.User:create',
      'auth.User:delete',
      'User:read',
      'User:write',
      'User:create',
      'User:delete',
      'auth.Role:read',
      'auth.Role:write',
      'auth.Role:create',
      'auth.Role:delete',
      'Role:read',
      'Role:write',
      'Role:create',
      'Role:delete',
      'auth.UserRole:read',
      'auth.UserRole:write',
      'auth.UserRole:create',
      'auth.UserRole:delete',
      'UserRole:read',
      'UserRole:write',
      'UserRole:create',
      'UserRole:delete',
      'auth.RoleMethodAccess:read',
      'auth.RoleMethodAccess:write',
      'auth.RoleMethodAccess:create',
      'auth.RoleMethodAccess:delete',
      'RoleMethodAccess:read',
      'RoleMethodAccess:write',
      'RoleMethodAccess:create',
      'RoleMethodAccess:delete',
      'auth.RoleUiResource:read',
      'auth.RoleUiResource:write',
      'auth.RoleUiResource:create',
      'auth.RoleUiResource:delete',
      'RoleUiResource:read',
      'RoleUiResource:write',
      'RoleUiResource:create',
      'RoleUiResource:delete',
      'meta.MetaUiResource:read',
      'MetaUiResource:read',
      'meta.MetaModel:read',
      'MetaModel:read',
      'meta.MetaService:read',
      'MetaService:read',
      'meta.MetaApplication:read',
      'MetaApplication:read',
      'auth.RoleInheritance:read',
      'RoleInheritance:read',
    ],
  });
}

function disableAllowlist(): void {
  setReq({ depth: 0, recordRuleMode: '', fieldRuleMode: '' });
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : `${Date.now()}_${Math.random().toString(36).slice(2, 10)}`;
  return `${prefix}_${u.replace(/[^a-zA-Z0-9]/g, '').slice(0, 24)}`;
}

async function createUser(companyId: string): Promise<string> {
  const created = await User.Create(
    {
      Username: uid('u'),
      Email: `${uid('e')}@example.com`,
      PasswordHash: 'test',
      FirstName: 'T',
      LastName: 'U',
      IsActive: true,
      CompanyId: companyId,
      CompanyIds: [companyId],
    } as any,
    ['Id'] as any
  );
  return String((created as any).Id);
}

async function createRole(): Promise<string> {
  const created = await Role.Create(
    {
      Name: uid('role'),
      Code: uid('CODE'),
      Description: 'e5',
      IsActive: true,
      IsSystem: false,
    } as any,
    ['Id'] as any
  );
  return String((created as any).Id);
}

async function resolveBrowse(): Promise<{ name: string; modelId: string; serviceId: string }> {
  const modelId = await metaModelId('auth', 'User');
  const services = await MetaService.Search({ And: [['ModelId', '=', modelId]] } as any, {
    fields: ['Id', 'Name'],
    limit: 5000,
  } as any);
  const browse = (services || []).find((r: any) => String(r.Name || '').toLowerCase() === 'browse');
  return { name: String((browse as any)?.Name || 'Browse'), modelId, serviceId: String((browse as any)?.Id || '') };
}

test('PR-E-5 CheckMethodAccess returns early diagnostic reasons', async () => {
  resetRequestContext();

  expect(await User.CheckMethodAccess('c1', '/auth.User/Browse')).toEqual({
    allowed: false,
    reason: 'missing_identity_or_method',
    hitRuleIds: [],
  });

  setIdentity('user-1');
  expect(await User.CheckMethodAccess('c1', '')).toEqual({
    allowed: false,
    reason: 'missing_identity_or_method',
    hitRuleIds: [],
  });
  expect(await User.CheckMethodAccess('c1', 'not-a-method')).toEqual({
    allowed: false,
    reason: 'invalid_service_full_name',
    hitRuleIds: [],
  });
  expect(await User.CheckMethodAccess('', '/auth.User/Browse')).toEqual({
    allowed: false,
    reason: 'missing_company_id',
    hitRuleIds: [],
  });

  const jsCtx = ensureRequestContext();
  jsCtx.ctx = { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] };
  expect(await User.CheckMethodAccess('c2', '/auth.User/Browse')).toEqual({
    allowed: false,
    reason: 'company_not_in_enabled_scope',
    hitRuleIds: [],
  });
});

test('PR-E-5 CheckMethodAccess covers cache, meta-miss, ui allow and internal_error', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();
  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id, c2.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);
      const roleId = await createRole();
      // Company-scoped role: present for c1 only.
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: roleId } as any,
          CompanyId: c1.Id as any,
        } as any,
        ['Id'] as any
      );

      const browse = await resolveBrowse();
      const fullMethod = `/auth.User/${browse.name}`;
      disableAllowlist();

      // No manual ACL and no UI grant => ui no match.
      const noMatch = await User.CheckMethodAccess(c1.Id, fullMethod);
      expect(noMatch.allowed).toBe(false);
      expect(noMatch.reason).toBe('method_access_ui_no_match');

      // Object cache hit returns the same decision envelope.
      const cached = await User.CheckMethodAccess(c1.Id, fullMethod);
      expect(cached).toEqual(noMatch);

      // Legacy boolean cache entry is cleared and recomputed as an envelope.
      const req = getCurrentReq();
      const state = getOrInitReqServiceState(req) as Record<string, unknown>;
      const cacheKey = buildMethodAccessCacheKey(userId, c1.Id, fullMethod);
      state[cacheKey] = true;
      const afterLegacy = await User.CheckMethodAccess(c1.Id, fullMethod);
      expect(afterLegacy.allowed).toBe(false);
      expect(typeof afterLegacy.reason).toBe('string');
      expect(typeof (state[cacheKey] as any)?.allowed).toBe('boolean');

      // Unknown method name => method_meta_not_found.
      const metaMiss = await User.CheckMethodAccess(c1.Id, '/auth.User/DefinitelyMissingRpc_E5');
      expect(metaMiss).toEqual({
        allowed: false,
        reason: 'method_meta_not_found',
        hitRuleIds: [],
      });

      // UI allow / deny paths (no manual ACL).
      const originalRoleUiSearch = (RoleUiResource as any).Search;
      const originalIrUiSearch = (MetaUiResource as any).Search;
      const clearMethodAndUiCaches = () => {
        delete state[cacheKey];
        for (const key of Object.keys(state)) {
          if (key.startsWith('uiGrantExpansion::')) delete state[key];
        }
      };
      (RoleUiResource as any).Search = async () => [
        { MetaApplicationId: null, MetaUiResourceId: 'RES-E5-UI', Mode: 'allow' },
      ];
      (MetaUiResource as any).Search = async () => [
        {
          Id: 'RES-E5-UI',
          Name: 'res-e5-ui',
          MetaApplicationId: 'APP',
          Requires: [`rpc:/auth.User/${String(browse.name).toLowerCase()}`],
        },
      ];
      try {
        clearMethodAndUiCaches();
        const uiAllow = await User.CheckMethodAccess(c1.Id, fullMethod);
        expect(uiAllow.allowed).toBe(true);
        expect(uiAllow.reason).toBe('method_access_ui_allow');
        expect(uiAllow.hitRuleIds).toEqual(['RES-E5-UI']);

        (RoleUiResource as any).Search = async () => [
          { MetaApplicationId: null, MetaUiResourceId: 'RES-E5-UI-DENY', Mode: 'deny' },
        ];
        (MetaUiResource as any).Search = async () => [
          {
            Id: 'RES-E5-UI-DENY',
            Name: 'res-e5-ui-deny',
            MetaApplicationId: 'APP',
            Requires: [`rpc:/auth.User/${String(browse.name).toLowerCase()}`],
          },
        ];
        clearMethodAndUiCaches();
        const uiDeny = await User.CheckMethodAccess(c1.Id, fullMethod);
        expect(uiDeny.allowed).toBe(false);
        expect(uiDeny.reason).toBe('method_access_ui_deny');
        expect(uiDeny.hitRuleIds).toEqual(['RES-E5-UI-DENY']);
      } finally {
        (RoleUiResource as any).Search = originalRoleUiSearch;
        (MetaUiResource as any).Search = originalIrUiSearch;
      }

      // Company in enabled scope but without a role binding.
      const noRoles = await User.CheckMethodAccess(c2.Id, fullMethod);
      expect(noRoles).toEqual({
        allowed: false,
        reason: 'no_roles_for_company',
        hitRuleIds: [],
      });

      // Force internal_error via authz context failure.
      const origAuthz = (User as any)._getAuthzContext;
      (User as any)._getAuthzContext = async () => {
        throw new Error('boom');
      };
      try {
        delete state[cacheKey];
        const boom = await User.CheckMethodAccess(c1.Id, fullMethod);
        expect(boom).toEqual({ allowed: false, reason: 'internal_error', hitRuleIds: [] });
      } finally {
        (User as any)._getAuthzContext = origAuthz;
      }

      return true;
    },
    { merge: false }
  );

  expect(out).toBe(true);
});

test('PR-E-5 evaluateUiDerivedMethodDecision covers empty inputs and empty requires', async () => {
  resetRequestContext();

  expect(await evaluateUiDerivedMethodDecision(['r'], '', 'browse')).toEqual({
    allowed: false,
    denied: false,
    hitRuleIds: [],
    reason: 'method_access_ui_no_match',
  });
  expect(await evaluateUiDerivedMethodDecision(['r'], 'auth.User', '')).toEqual({
    allowed: false,
    denied: false,
    hitRuleIds: [],
    reason: 'method_access_ui_no_match',
  });

  const originalRoleUiSearch = (RoleUiResource as any).Search;
  const originalIrUiSearch = (MetaUiResource as any).Search;

  (RoleUiResource as any).Search = async () => [
    { MetaApplicationId: null, MetaUiResourceId: null, Mode: 'deny' }, // global deny
  ];
  (MetaUiResource as any).Search = async () => [
    { Id: 'RES-EMPTY', Name: 'empty', Requires: [] }, // skipped
    { Id: 'RES-G', Name: 'g', Requires: ['rpc:/auth.User/browse'] },
    { Name: 'no-id', Requires: ['rpc:/auth.User/browse'] }, // missing Id; still can deny via global
  ];

  try {
    const denied = await evaluateUiDerivedMethodDecision(['ROLE-E5-EMPTY'], 'auth.User', 'browse');
    expect(denied.denied).toBe(true);
    expect(denied.reason).toBe('method_access_ui_deny');
    expect(denied.hitRuleIds).toEqual(['RES-G']);
  } finally {
    (RoleUiResource as any).Search = originalRoleUiSearch;
    (MetaUiResource as any).Search = originalIrUiSearch;
  }

  // Empty expansion resources.
  resetRequestContext();
  (RoleUiResource as any).Search = async () => [];
  (MetaUiResource as any).Search = async () => [];
  try {
    const empty = await evaluateUiDerivedMethodDecision(['ROLE-E5-NONE'], 'auth.User', 'browse');
    expect(empty).toEqual({
      allowed: false,
      denied: false,
      hitRuleIds: [],
      reason: 'method_access_ui_no_match',
    });
  } finally {
    (RoleUiResource as any).Search = originalRoleUiSearch;
    (MetaUiResource as any).Search = originalIrUiSearch;
  }
});

test('PR-E-5 evaluateUiDerivedMethodDecision covers id fallback and no-match resources', async () => {
  resetRequestContext();
  const originalRoleUiSearch = (RoleUiResource as any).Search;
  const originalIrUiSearch = (MetaUiResource as any).Search;

  // Resource keyed only by lowercase `id` (no `Id`) still participates in allow diagnostics.
  (RoleUiResource as any).Search = async () => [{ MetaApplicationId: null, MetaUiResourceId: 'res-lower-id', Mode: 'allow' }];
  (MetaUiResource as any).Search = async () => [{ id: 'res-lower-id', Name: 'lower', Requires: ['rpc:/auth.User/browse'] }];
  try {
    const allowed = await evaluateUiDerivedMethodDecision(['ROLE-E5-ID'], 'auth.User', 'browse');
    expect(allowed).toEqual({
      allowed: true,
      denied: false,
      hitRuleIds: ['res-lower-id'],
      reason: 'method_access_ui_allow',
    });
  } finally {
    (RoleUiResource as any).Search = originalRoleUiSearch;
    (MetaUiResource as any).Search = originalIrUiSearch;
  }

  // Resources exist but none match the requested method => final ui_no_match.
  resetRequestContext();
  (RoleUiResource as any).Search = async () => [{ MetaApplicationId: null, MetaUiResourceId: 'RES-OTHER', Mode: 'allow' }];
  (MetaUiResource as any).Search = async () => [
    { Id: 'RES-OTHER', Name: 'other', Requires: ['rpc:/auth.User/create'] },
  ];
  try {
    const noMatch = await evaluateUiDerivedMethodDecision(['ROLE-E5-OTHER'], 'auth.User', 'browse');
    expect(noMatch).toEqual({
      allowed: false,
      denied: false,
      hitRuleIds: [],
      reason: 'method_access_ui_no_match',
    });
  } finally {
    (RoleUiResource as any).Search = originalRoleUiSearch;
    (MetaUiResource as any).Search = originalIrUiSearch;
  }
});

test('PR-E-5 CheckMethodAccess manual deny and allow expose hitRuleIds', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();
  const c1 = { Id: uid('C1') };

  await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);
      const roleId = await createRole();
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: roleId } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );
      const browse = await resolveBrowse();

      const denyRow = await RoleMethodAccess.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaServiceId: browse.serviceId,
          MetaModelId: null,
          MetaApplicationId: null,
          Mode: 'deny',
          Source: 'manual',
        } as any,
        ['Id'] as any
      );
      disableAllowlist();
      const denied = await User.CheckMethodAccess(c1.Id, `/auth.User/${browse.name}`);
      expect(denied.allowed).toBe(false);
      expect(denied.reason).toBe('method_access_deny');
      expect(denied.hitRuleIds).toEqual([String((denyRow as any).Id)]);

      setupAllowlistForFixtures();
      await RoleMethodAccess.DeleteById(String((denyRow as any).Id));
      const allowRow = await RoleMethodAccess.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaServiceId: browse.serviceId,
          MetaModelId: null,
          MetaApplicationId: null,
          Mode: 'allow',
          Source: 'manual',
        } as any,
        ['Id'] as any
      );
      disableAllowlist();
      const allowed = await User.CheckMethodAccess(c1.Id, `/auth.User/${browse.name}`);
      expect(allowed.allowed).toBe(true);
      expect(allowed.reason).toBe('method_access_allow');
      expect(allowed.hitRuleIds).toContain(String((allowRow as any).Id));
    },
    { merge: false }
  );
});
