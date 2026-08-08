// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { withContext as withModelContext } from '@/core/service/api/context';
import { buildRelationAliasCandidates } from '@/core/service/testing';
import CompanyScopedResource from '@/auth/service/models/company_scoped_resource';
import User from '@/auth/service/models/user';
import Role from '@/auth/service/models/role';
import UserRole from '@/auth/service/models/user_role';
import RoleFieldRule from '@/auth/service/models/role_field_rule';
import { evaluateFieldRules } from '@/auth/service/models/_user_field_rule_eval';
import { resolveEffectiveApplicationId, resolveEffectiveModelId } from '../models/_resolve_effective_model';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaFieldModel from '@/meta/service/models/field';
const MetaField = createServiceByModel<typeof MetaFieldModel>('meta.MetaField');

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

const relAliasCandidates = buildRelationAliasCandidates;

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
  // This file focuses on FieldRule.
  // Use a top-level allowlist so RecordRule does not interfere with fixtures or assertions.
  jsCtx.req = {
    depth: 0,
    recordRuleMode: 'allowlist',
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

      // FieldRule and RecordRule evaluation both read the role inheritance graph (RoleInheritance).
      // Without it, depth=0 allowlist flows can hit record_rule_entry_allowlist_miss.
      'auth.RoleInheritance:read',
      'auth.RoleInheritance:write',
      'auth.RoleInheritance:create',
      'auth.RoleInheritance:delete',
      'RoleInheritance:read',
      'RoleInheritance:write',
      'RoleInheritance:create',
      'RoleInheritance:delete',

      'auth.RoleFieldRule:read',
      'auth.RoleFieldRule:write',
      'auth.RoleFieldRule:create',
      'auth.RoleFieldRule:delete',
      'RoleFieldRule:read',
      'RoleFieldRule:write',
      'RoleFieldRule:create',
      'RoleFieldRule:delete',

      'auth.CompanyScopedResource:read',
      'auth.CompanyScopedResource:write',
      'auth.CompanyScopedResource:create',
      'auth.CompanyScopedResource:delete',
      'CompanyScopedResource:read',
      'CompanyScopedResource:write',
      'CompanyScopedResource:create',
      'CompanyScopedResource:delete',
    ],
    fieldRuleMode: '',
  };
  // FieldRule evaluation requires identity.userId.
  // The first fixture User.Create runs before actorId is available, so seed a placeholder userId first.
  // Later cases switch to the real actor with setIdentity(actorId).
  jsCtx.identity = { userId: uid('bootstrap_user') };

  // Clear runtime/context caches and overrides to avoid cross-test leakage.
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];

  // Clear request-scoped record and field rule caches.
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

function setIdentity(userId?: string): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.identity) jsCtx.identity = {};
  if (userId) jsCtx.identity.userId = userId;
  else delete jsCtx.identity.userId;
}

function setReq(patch: Record<string, any>): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.req) jsCtx.req = {};
  Object.assign(jsCtx.req, patch);
}

function toChoysumErrorLike(err: any): { domain?: string; code?: string; message?: string } | null {
  if (!err) return null;

  const visited = new Set<any>();
  const queue: any[] = [err];

  while (queue.length) {
    const cur = queue.shift();
    if (!cur || visited.has(cur)) continue;
    visited.add(cur);

    if (cur instanceof ChoysumError) return cur as any;

    if (typeof cur === 'object') {
      const domain = (cur as any).domain;
      const code = (cur as any).code;
      const message = (cur as any).message;
      if (typeof domain === 'string' || typeof code === 'string') {
        return {
          domain: typeof domain === 'string' ? domain : undefined,
          code: typeof code === 'string' ? code : undefined,
          message: typeof message === 'string' ? message : undefined,
        };
      }

      // Common wrappers
      if ((cur as any).error) queue.push((cur as any).error);
      if ((cur as any).cause) queue.push((cur as any).cause);
      if (Array.isArray((cur as any).details)) {
        for (const d of (cur as any).details) queue.push(d);
      }
      if ((cur as any).info) queue.push((cur as any).info);
      if (typeof (cur as any).rawMessage === 'string') queue.push({ message: (cur as any).rawMessage });
    }

    // Fallback: parse from message string.
    const msg = typeof cur === 'string' ? cur : typeof (cur as any)?.message === 'string' ? (cur as any).message : '';
    if (msg) {
      // JSON payload
      if (msg.trim().startsWith('{')) {
        try {
          const obj = JSON.parse(msg);
          if (obj && (typeof obj.domain === 'string' || typeof obj.code === 'string')) {
            return {
              domain: typeof obj.domain === 'string' ? obj.domain : undefined,
              code: typeof obj.code === 'string' ? obj.code : undefined,
              message: typeof obj.message === 'string' ? obj.message : msg,
            };
          }
        } catch {
          // ignore
        }
      }

      // Matches: "ChoysumError: [core.repository] record_rule_denied: ..."
      // or: "[core.repository] record_rule_denied: ..."
      const m = msg.match(/\[(?<domain>[^\]]+)\]\s+(?<code>[^:\s]+):\s+(?<message>[\s\S]*)$/);
      if (m?.groups?.domain || m?.groups?.code) {
        return {
          domain: m.groups?.domain,
          code: m.groups?.code,
          message: m.groups?.message,
        };
      }
    }
  }

  return null;
}

async function resolveModelId(appName: string, modelName: string): Promise<string> {
  const id = await resolveEffectiveModelId(appName, modelName);
  if (!id) throw new Error(`meta model not found: app=${appName} model=${modelName}`);
  return id;
}

async function resolveApplicationId(appName: string): Promise<string> {
  const id = await resolveEffectiveApplicationId(appName);
  if (!id) throw new Error(`meta application not found: name=${appName}`);
  return id;
}

async function resolveFieldId(modelId: string, fieldName: string): Promise<string> {
  const rows = await MetaField.Search(
    {
      And: [
        ['ModelId', '=', modelId],
        ['Name', '=', fieldName],
      ],
    } as any,
    { fields: ['Id', 'UpdatedAt'], orderBy: { field: 'UpdatedAt', order: 'desc' }, limit: 1 }
  );
  const id = String((rows as any)?.[0]?.Id || '').trim();
  if (!id) throw new Error(`meta field not found: modelId=${modelId} field=${fieldName}`);
  return id;
}

async function withFieldRuleSkip<T>(fn: () => Promise<T>): Promise<T> {
  const jsCtx = ensureRequestContext();
  const prev = jsCtx.req?.fieldRuleMode;
  setReq({ fieldRuleMode: 'skip' });
  try {
    return await fn();
  } finally {
    setReq({ fieldRuleMode: prev ?? '' });
  }
}

async function createUser(companyId: string): Promise<string> {
  // Fixtures must bypass FieldRule: deny-default would block User.Create with no FR grants (PR-C-1).
  return await withFieldRuleSkip(async () => {
    const created = await User.Create(
      {
        Username: uid('u'),
        PasswordHash: 'test',
        FirstName: 'T',
        LastName: 'U',
        CompanyId: companyId,
        CompanyIds: [companyId],
        IsActive: true,
      } as any,
      ['Id'] as any
    );
    return created.Id;
  });
}

async function createRole(): Promise<string> {
  return await withFieldRuleSkip(async () => {
    const created = await Role.Create(
      {
        Name: uid('role'),
        Code: uid('ROLE'),
        Description: 'test',
        IsActive: true,
        IsSystem: false,
      } as any,
      ['Id'] as any
    );
    return created.Id;
  });
}

async function grantRoleGlobal(userId: string, roleId: string, companyId: string): Promise<void> {
  await withFieldRuleSkip(async () => {
    await withModelContext(
      { activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any,
      async () => {
        await UserRole.Create(
          {
            UserId: { Id: userId } as any,
            RoleId: { Id: roleId } as any,
            CompanyId: null as any, // global role
          } as any,
          ['Id'] as any
        );
      },
      { merge: false }
    );
  });
}

test('P4 field rule: denyWriteFields blocks update when payload includes denied field', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const { actorId, locationId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');
      const nameFieldId = await resolveFieldId(locationModelId, 'Name');

      // Make the Name field readable but not writable.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'allow',
          PermWrite: 'deny',
        } as any,
        ['Id'] as any
      );

      // Create a fixture row in top-level skip mode so create is not blocked by denyWrite.
      setReq({ depth: 0, fieldRuleMode: 'skip' });
      const created = await CompanyScopedResource.Create({ Name: uid('csr_fr') }, ['Id'] as any);

      return { actorId: uid1, locationId: created.Id };
    },
    { merge: false }
  );

  // Ensure the previous skip mode does not pollute cache state, then restore normal mode.
  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setReq({ depth: 0, fieldRuleMode: '' });
  setIdentity(actorId);

  try {
    await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        // Use static Update directly to bypass BaseModel.update() change tracking.
        // Signature: Update(condition, values, returnFields).
        await CompanyScopedResource.Update(['Id', '=', locationId] as any, { Name: uid('csr_fr_new') } as any, []);
      },
      { merge: false }
    );
    throw new Error('expected update to throw');
  } catch (err) {
    const oe = toChoysumErrorLike(err);
    expect(Boolean(oe)).toBe(true);
    expect(String(oe?.domain || '')).toBe('core.repository');
    expect(String(oe?.code || '')).toBe('field_rule_readonly_violation');
  }
});

test('P4 field rule: same request RoleFieldRule write invalidates request cache', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const actorId = await createUser(c1.Id);
      setIdentity(actorId);

      const roleId = await createRole();

      // grant global role
      await withFieldRuleSkip(async () => {
        await UserRole.Create(
          {
            UserId: { Id: actorId } as any,
            RoleId: { Id: roleId } as any,
            CompanyId: null as any,
          } as any,
          ['Id'] as any
        );
      });

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');
      const nameFieldId = await resolveFieldId(locationModelId, 'Name');

      // fixtures: create a record without being blocked by future denyWrite
      setReq({ depth: 0, fieldRuleMode: 'skip' });
      const created = await CompanyScopedResource.Create({ Name: uid('csr_midreq') }, ['Id'] as any);

      // Under deny-default, seed an allow so the warm Update can succeed and populate the request cache.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'allow',
          PermWrite: 'allow',
        } as any,
        ['Id'] as any
      );

      setReq({ depth: 0, fieldRuleMode: '' });
      await CompanyScopedResource.Update(['Id', '=', created.Id] as any, { Name: uid('csr_pre') } as any, []);

      // mutate permission graph mid-request: deny writing Name (same-scope deny-wins)
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'allow',
          PermWrite: 'deny',
        } as any,
        ['Id'] as any
      );

      try {
        await CompanyScopedResource.Update(['Id', '=', created.Id] as any, { Name: uid('csr_post') } as any, []);
        throw new Error('expected update to throw');
      } catch (err) {
        const oe = toChoysumErrorLike(err);
        expect(Boolean(oe)).toBe(true);
        expect(String(oe?.domain || '')).toBe('core.repository');
        expect(String(oe?.code || '')).toBe('field_rule_readonly_violation');
      }
    },
    { merge: false }
  );
});

test('P4 field rule: top-level fieldRuleMode=skip bypasses denyWriteFields', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const { actorId, locationId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');
      const nameFieldId = await resolveFieldId(locationModelId, 'Name');

      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'allow',
          PermWrite: 'deny',
        } as any,
        ['Id'] as any
      );

      // Create the initial row in skip mode.
      setReq({ depth: 0, fieldRuleMode: 'skip' });
      const created = await CompanyScopedResource.Create({ Name: uid('csr_fr2') }, ['Id'] as any);
      return { actorId: uid1, locationId: created.Id };
    },
    { merge: false }
  );

  setIdentity(actorId);

  // Top-level skip should allow updating denied fields.
  const updatedName = uid('csr_fr2_new');
  await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, fieldRuleMode: 'skip' });
      // Use static Update with argument order: (condition, values, returnFields).
      await CompanyScopedResource.Update(['Id', '=', locationId] as any, { Name: updatedName } as any, []);
    },
    { merge: false }
  );

  // Reload in skip mode to confirm the updated value without denyReadFields interference.
  const reloadedSkip = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, fieldRuleMode: 'skip' });
      return await CompanyScopedResource.Browse(locationId, ['Id', 'Name'] as any);
    },
    { merge: false }
  );
  expect(String((reloadedSkip as any).Name)).toBe(updatedName);

  // Reload again to confirm the update succeeded.
  const reloaded = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, fieldRuleMode: '' });
      return await CompanyScopedResource.Browse(locationId, ['Id', 'Name'] as any);
    },
    { merge: false }
  );
  expect(String((reloaded as any).Name)).toBe(updatedName);
});

test('P4 field rule: top-level facade return strips denyReadFields for BaseModel', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  // Hand-written facade: the inner Browse(depth>0) returns a proxy model and does not take the
  // BaseModel.Browse top-level stripping branch. The top-level service runtime should still strip
  // denyReadFields from BaseModel returns.
  class LocationFacade {
    static async GetLocationForUi(id: string): Promise<any> {
      return await CompanyScopedResource.Browse(id, ['Id', 'Name'] as any);
    }
  }

  const { actorId, locationId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');
      const nameFieldId = await resolveFieldId(locationModelId, 'Name');

      // Make the Name field externally unreadable (PermRead=false) and unwritable (PermWrite=false).
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'deny',
          PermWrite: 'deny',
        } as any,
        ['Id'] as any
      );

      // Create a fixture row in top-level skip mode.
      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: 'skip' });
      const created = await CompanyScopedResource.Create({ Name: uid('csr_fr_read') }, ['Id'] as any);
      return { actorId: uid1, locationId: created.Id };
    },
    { merge: false }
  );

  // Ensure the previous skip mode does not pollute cache state, then restore normal mode.
  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setIdentity(actorId);

  const result = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: '' });
      return await LocationFacade.GetLocationForUi(locationId);
    },
    { merge: false }
  );

  expect(typeof (result as any)?.toPlainObject).toBe('undefined');
  expect((result as any)?.__choysum_plain).toBe(true);
  expect(String((result as any)?.Id || '')).toBe(locationId);
  expect(Object.prototype.hasOwnProperty.call(result as any, 'Name')).toBe(false);
});

test('P4 field rule: top-level Browse strips denyReadFields (via service runtime finalize)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const { actorId, locationId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');
      const nameFieldId = await resolveFieldId(locationModelId, 'Name');

      // Make the Name field externally unreadable (PermRead=false).
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'deny',
          PermWrite: 'deny',
        } as any,
        ['Id'] as any
      );

      // Create a fixture row in top-level skip mode.
      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: 'skip' });
      const created = await CompanyScopedResource.Create({ Name: uid('csr_fr_browse') }, ['Id'] as any);
      return { actorId: uid1, locationId: created.Id };
    },
    { merge: false }
  );

  // Ensure the previous skip mode does not pollute cache state, then restore normal mode.
  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setIdentity(actorId);

  const result = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: '' });
      return await CompanyScopedResource.Browse(locationId, ['Id', 'Name'] as any);
    },
    { merge: false }
  );

  expect(String((result as any).Id)).toBe(String(locationId));
  expect(Object.prototype.hasOwnProperty.call(result as any, 'Name')).toBe(false);
});

test('P4 field rule: top-level response recursively strips denyReadFields for relation payload ($rel$_*)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const { actorId, roleId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const rid = await createRole();
      await grantRoleGlobal(uid1, rid, c1.Id);

      const userModelId = await resolveModelId('auth', 'User');
      const usernameFieldId = await resolveFieldId(userModelId, 'Username');
      const urModelId = await resolveModelId('auth', 'UserRole');
      const userIdFieldId = await resolveFieldId(urModelId, 'UserId');

      // Deny-default: allow reading the UserRole.UserId relation and User model fields,
      // then deny Username specifically so nested strip can be asserted.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: rid } as any,
          MetaModelId: urModelId,
          MetaFieldId: userIdFieldId,
          PermRead: 'allow',
          PermWrite: 'allow',
        } as any,
        ['Id'] as any
      );
      await RoleFieldRule.Create(
        {
          RoleId: { Id: rid } as any,
          MetaModelId: userModelId,
          MetaFieldId: null,
          PermRead: 'allow',
          PermWrite: 'allow',
        } as any,
        ['Id'] as any
      );
      await RoleFieldRule.Create(
        {
          RoleId: { Id: rid } as any,
          MetaModelId: userModelId,
          MetaFieldId: usernameFieldId,
          PermRead: 'deny',
          PermWrite: 'allow',
        } as any,
        ['Id'] as any
      );

      return { actorId: uid1, roleId: rid };
    },
    { merge: false }
  );

  // Ensure cache isolation
  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setIdentity(actorId);

  class UserRoleFacade {
    static async SearchForUi(roleId: string): Promise<any> {
      return await UserRole.Search(
        ['RoleId', '=', roleId] as any,
        {
          fields: ['Id', { UserId: ['Id', 'Username', 'FirstName'] as any } as any] as any,
          limit: 10,
        } as any
      );
    }
  }

  const rows = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: '' });
      return await UserRoleFacade.SearchForUi(roleId);
    },
    { merge: false }
  );

  expect(Array.isArray(rows)).toBe(true);
  expect((rows as any[]).length > 0).toBe(true);
  const r0: any = (rows as any[])[0];

  if ((r0 as any)?.__choysum_plain !== true) {
    try {
      console.log(
        '[DEBUG][field_rule#14] plain markers: rows.__choysum_plain=',
        (rows as any)?.__choysum_plain,
        'r0.__choysum_plain=',
        (r0 as any)?.__choysum_plain,
        'r0Keys=',
        Object.keys(r0 || {})
      );
    } catch {}
  }
  expect((r0 as any)?.__choysum_plain).toBe(true);
  const relKey = relAliasCandidates('UserId').find(k => Object.prototype.hasOwnProperty.call(r0, k));
  const userRelRaw = (r0 as any)?.UserId && typeof (r0 as any).UserId === 'object' ? (r0 as any).UserId : relKey ? (r0 as any)[relKey as any] : undefined;
  if (!userRelRaw) {
    try {
      console.log('[DEBUG][field_rule#14] user relation payload not found; relCandidates=', relAliasCandidates('UserId'), 'r0Keys=', Object.keys(r0 || {}));
    } catch {}
  }
  expect(Boolean(userRelRaw)).toBe(true);

  const userRel =
    typeof userRelRaw === 'string'
      ? (() => {
          try {
            return JSON.parse(userRelRaw);
          } catch {
            return undefined;
          }
        })()
      : userRelRaw;
  if (!userRel || typeof userRel !== 'object') {
    try {
      console.log('[DEBUG][field_rule#14] relKey=', relKey, 'typeof=', typeof userRel, 'val=', userRel);
    } catch {}
  }
  expect(Boolean(userRel && typeof userRel === 'object')).toBe(true);
  const hasId = Object.prototype.hasOwnProperty.call(userRel, 'Id') || Object.prototype.hasOwnProperty.call(userRel, 'id');
  if (!hasId) {
    try {
      console.log('[DEBUG][field_rule#14] userRel keys=', Object.keys(userRel || {}));
    } catch {}
  }
  expect(hasId).toBe(true);
  const hasUsername = Object.prototype.hasOwnProperty.call(userRel, 'Username') || Object.prototype.hasOwnProperty.call(userRel, 'username');
  if (hasUsername) {
    try {
      console.log('[DEBUG][field_rule#14] username still present, keys=', Object.keys(userRel || {}));
    } catch {}
  }
  expect(hasUsername).toBe(false);
});

test('P4 field rule: denyReadFields can deny relation field key (drops both UserId and $rel$_UserId)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const { actorId, roleId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const rid = await createRole();
      await grantRoleGlobal(uid1, rid, c1.Id);

      const urModelId = await resolveModelId('auth', 'UserRole');
      const userIdFieldId = await resolveFieldId(urModelId, 'UserId');

      // Deny read on the relation field itself.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: rid } as any,
          MetaModelId: urModelId,
          MetaFieldId: userIdFieldId,
          PermRead: 'deny',
          PermWrite: 'allow',
        } as any,
        ['Id'] as any
      );

      return { actorId: uid1, roleId: rid };
    },
    { merge: false }
  );

  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setReq({ depth: 0, fieldRuleMode: '' });
  setIdentity(actorId);

  class UserRoleFacade {
    static async SearchForUi(roleId: string): Promise<any> {
      return await UserRole.Search(
        ['RoleId', '=', roleId] as any,
        {
          fields: ['Id', 'UserId', { UserId: ['Id', 'Username'] as any } as any] as any,
          limit: 10,
        } as any
      );
    }
  }

  const rows = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: '' });
      return await UserRoleFacade.SearchForUi(roleId);
    },
    { merge: false }
  );

  expect(Array.isArray(rows)).toBe(true);
  expect((rows as any[]).length > 0).toBe(true);
  const r0: any = (rows as any[])[0];

  expect((r0 as any)?.__choysum_plain).toBe(true);
  expect(Object.prototype.hasOwnProperty.call(r0, 'UserId')).toBe(false);
  for (const k of relAliasCandidates('UserId')) {
    expect(Object.prototype.hasOwnProperty.call(r0, k)).toBe(false);
  }
});

test('P4 field rule wildcard: model deny-write + field allow-write overrides for specific field', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const { actorId, locationId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');
      const nameFieldId = await resolveFieldId(locationModelId, 'Name');

      // Model wildcard: deny writing all fields in this model.
      // Deny-default + read-deny⇒write-deny: also seed model allow-read so field write allow can take effect.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaApplicationId: null,
          MetaModelId: locationModelId,
          MetaFieldId: null,
          PermRead: 'allow',
          PermWrite: 'deny',
        } as any,
        ['Id'] as any
      );

      // Field exact override: allow writing Name.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaApplicationId: null,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'allow',
          PermWrite: 'allow',
        } as any,
        ['Id'] as any
      );

      // fixtures: create a record without being blocked by denyWrite
      setReq({ depth: 0, fieldRuleMode: 'skip' });
      const created = await CompanyScopedResource.Create({ Name: uid('csr_wild') }, ['Id'] as any);
      return { actorId: uid1, locationId: created.Id };
    },
    { merge: false }
  );

  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setReq({ depth: 0, fieldRuleMode: '' });
  setIdentity(actorId);

  // Name should be writable due to field override.
  await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, fieldRuleMode: '' });
      await CompanyScopedResource.Update(['Id', '=', locationId] as any, { Name: uid('csr_wild_ok') } as any, []);
    },
    { merge: false }
  );

  // But CompanyId should remain write-denied by model wildcard.
  try {
    await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        setReq({ depth: 0, fieldRuleMode: '' });
        await CompanyScopedResource.Update(['Id', '=', locationId] as any, { CompanyId: uid('C2') } as any, []);
      },
      { merge: false }
    );
    throw new Error('expected update to throw');
  } catch (err) {
    const oe = toChoysumErrorLike(err);
    expect(Boolean(oe)).toBe(true);
    expect(String(oe?.domain || '')).toBe('core.repository');
    expect(String(oe?.code || '')).toBe('field_rule_readonly_violation');
  }
});

test('P4 field rule wildcard: app deny-read can be overridden by model allow-read', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const { actorId, locationId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const authAppId = await resolveApplicationId('auth');
      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');

      // Application wildcard: deny-read everything under auth.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaApplicationId: authAppId,
          MetaModelId: null,
          MetaFieldId: null,
          PermRead: 'deny',
        } as any,
        ['Id'] as any
      );

      // Model wildcard override: allow-read CompanyScopedResource.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaApplicationId: null,
          MetaModelId: locationModelId,
          MetaFieldId: null,
          PermRead: 'allow',
        } as any,
        ['Id'] as any
      );

      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: 'skip' });
      const created = await CompanyScopedResource.Create({ Name: uid('csr_app_override') }, ['Id'] as any);
      return { actorId: uid1, locationId: created.Id };
    },
    { merge: false }
  );

  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setIdentity(actorId);

  const row = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: '' });
      return await CompanyScopedResource.Browse(locationId, ['Id', 'Name'] as any);
    },
    { merge: false }
  );

  expect(Object.prototype.hasOwnProperty.call(row as any, 'Name')).toBe(true);
  expect(String((row as any).Id)).toBe(String(locationId));
});

test('P4 field rule wildcard: same-scope deny wins over allow (field exact)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const { actorId, locationId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');
      const nameFieldId = await resolveFieldId(locationModelId, 'Name');

      // Same scope (field exact), conflicting decisions -> deny wins.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'allow',
        } as any,
        ['Id'] as any
      );
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'deny',
        } as any,
        ['Id'] as any
      );

      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: 'skip' });
      const created = await CompanyScopedResource.Create({ Name: uid('csr_same_scope') }, ['Id'] as any);
      return { actorId: uid1, locationId: created.Id };
    },
    { merge: false }
  );

  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setIdentity(actorId);

  const row = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: '' });
      return await CompanyScopedResource.Browse(locationId, ['Id', 'Name'] as any);
    },
    { merge: false }
  );

  expect(Object.prototype.hasOwnProperty.call(row as any, 'Name')).toBe(false);
});

test('P4 field rule wildcard: write implies read (read=deny forces write=deny)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const { actorId, locationId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');
      const nameFieldId = await resolveFieldId(locationModelId, 'Name');

      // Attempt to allow write but deny read on the same field.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'deny',
          PermWrite: 'allow',
        } as any,
        ['Id'] as any
      );

      setReq({ depth: 0, fieldRuleMode: 'skip' });
      const created = await CompanyScopedResource.Create({ Name: uid('csr_wir') }, ['Id'] as any);
      return { actorId: uid1, locationId: created.Id };
    },
    { merge: false }
  );

  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setReq({ depth: 0, fieldRuleMode: '' });
  setIdentity(actorId);

  try {
    await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        await CompanyScopedResource.Update(['Id', '=', locationId] as any, { Name: uid('csr_wir_new') } as any, []);
      },
      { merge: false }
    );
    throw new Error('expected update to throw');
  } catch (err) {
    const oe = toChoysumErrorLike(err);
    expect(Boolean(oe)).toBe(true);
    expect(String(oe?.domain || '')).toBe('core.repository');
    expect(String(oe?.code || '')).toBe('field_rule_readonly_violation');
  }
});

test('RoleFieldRule db check: deleted rows bypass scope xor', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const modelId = await resolveModelId('auth', 'CompanyScopedResource');
      const fieldId = await resolveFieldId(modelId, 'Name');
      const appId = await resolveApplicationId('auth');

      const row = await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: modelId,
          MetaFieldId: fieldId,
          MetaApplicationId: null,
          PermRead: 'allow',
          PermWrite: null,
        } as any,
        ['Id'] as any
      );

      const id = String((row as any)?.Id || '').trim();
      expect(id.length > 0).toBe(true);

      await RoleFieldRule.DeleteById(id);

      const repo = RoleFieldRule.getRepository().withDeleted();
      const updated = await repo.update(
        {
          MetaFieldId: fieldId,
          MetaModelId: modelId,
          MetaApplicationId: appId,
        } as any,
        ['Id', '=', id] as any
      );

      expect((updated || []).length).toBe(1);

      const rows = await RoleFieldRule.Search(
        ['Id', '=', id] as any,
        {
          fields: ['Id', 'DeletedAt', 'MetaFieldId', 'MetaModelId', 'MetaApplicationId', 'PermRead', 'PermWrite'] as any,
          withDeleted: true,
        } as any
      );

      expect(rows.length).toBe(1);
      expect(String((rows[0] as any)?.MetaFieldId || '').trim()).toBe(fieldId);
      expect(String((rows[0] as any)?.MetaModelId || '').trim()).toBe(modelId);
      expect(String((rows[0] as any)?.MetaApplicationId || '').trim()).toBe(appId);
      expect(String((rows[0] as any)?.PermRead || '').trim()).toBe('allow');
      expect(String((rows[0] as any)?.PermWrite || '').trim()).toBe('');
      expect((rows[0] as any)?.DeletedAt != null).toBe(true);
    },
    { merge: false }
  );
});

test('RoleFieldRule: permission-only update must not rewrite scoped fields to global', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const modelId = await resolveModelId('auth', 'CompanyScopedResource');
      const fieldId = await resolveFieldId(modelId, 'Name');

      const created = await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: modelId,
          MetaFieldId: fieldId,
          MetaApplicationId: null,
          PermRead: 'allow',
          PermWrite: 'deny',
        } as any,
        ['Id', 'MetaFieldId', 'MetaModelId', 'MetaApplicationId', 'PermRead', 'PermWrite'] as any
      );

      const id = String((created as any)?.Id || '').trim();
      expect(id.length > 0).toBe(true);

      await RoleFieldRule.UpdateById(
        id,
        {
          PermRead: 'deny',
        } as any,
        ['Id'] as any
      );

      const rows = await RoleFieldRule.Search(
        ['Id', '=', id] as any,
        { fields: ['Id', 'MetaFieldId', 'MetaModelId', 'MetaApplicationId', 'PermRead', 'PermWrite'], limit: 1 } as any
      );

      expect(rows.length).toBe(1);
      expect(String((rows[0] as any)?.MetaFieldId || '').trim()).toBe(fieldId);
      expect(String((rows[0] as any)?.MetaModelId || '').trim()).toBe(modelId);
      expect(String((rows[0] as any)?.MetaApplicationId || '').trim()).toBe('');
      expect(String((rows[0] as any)?.PermRead || '').trim()).toBe('deny');
      // Untouched PermWrite must survive a partial permission update.
      expect(String((rows[0] as any)?.PermWrite || '').trim()).toBe('deny');

      await RoleFieldRule.Update(
        ['Id', '=', id] as any,
        { PermWrite: 'allow' } as any,
        ['Id'] as any
      );

      const afterWriteOnly = await RoleFieldRule.Search(
        ['Id', '=', id] as any,
        { fields: ['Id', 'PermRead', 'PermWrite'], limit: 1 } as any
      );
      expect(afterWriteOnly.length).toBe(1);
      expect(String((afterWriteOnly[0] as any)?.PermRead || '').trim()).toBe('deny');
      expect(String((afterWriteOnly[0] as any)?.PermWrite || '').trim()).toBe('allow');
    },
    { merge: false }
  );
});

// ---------------------------------------------------------------------------
// PR-C-1: FieldRule deny-by-default
// ---------------------------------------------------------------------------

test('P4 field rule deny-default: no roles denies all non-system fields', async () => {
  resetRequestContext();
  setReq({ depth: 0, fieldRuleMode: '' });

  const out = await withModelContext({} as any, async () => {
    return await evaluateFieldRules({
      appName: 'auth',
      modelName: 'CompanyScopedResource',
      modelFullName: 'auth.CompanyScopedResource',
      roleIds: [],
    });
  }, { merge: false });

  expect(out.reason).toBe('no_roles_deny_by_default');
  expect(out.denyReadFields).toEqual(out.denyWriteFields);
  expect(out.denyReadFields.includes('Name')).toBe(true);
  expect(out.denyReadFields.includes('Id')).toBe(false);
  expect(out.denyReadFields.includes('DisplayName')).toBe(false);
});

test('P4 field rule deny-default: only system fields yields no_fields_deny_by_default', async () => {
  resetRequestContext();
  setReq({ depth: 0, fieldRuleMode: '' });

  const origMetaFieldSearch = (MetaField as any).Search;
  (MetaField as any).Search = async () => [
    { Id: 'sys_id', Name: 'Id' },
    { Id: 'sys_dn', Name: 'DisplayName' },
    { Id: 'sys_ca', Name: 'CreatedAt' },
  ];

  try {
    const out = await withModelContext({} as any, async () => {
      return await evaluateFieldRules({
        appName: 'auth',
        modelName: 'CompanyScopedResource',
        modelFullName: 'auth.CompanyScopedResource',
        roleIds: ['role_unused'],
      });
    }, { merge: false });

    expect(out.reason).toBe('no_fields_deny_by_default');
    expect(out.denyReadFields).toEqual([]);
    expect(out.denyWriteFields).toEqual([]);
  } finally {
    (MetaField as any).Search = origMetaFieldSearch;
  }
});

test('P4 field rule deny-default: roles without FR rows deny all non-system fields', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };
  const { roleId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);
      const rid = await createRole();
      await grantRoleGlobal(uid1, rid, c1.Id);
      return { roleId: rid };
    },
    { merge: false }
  );

  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setReq({ depth: 0, fieldRuleMode: '' });

  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      return await evaluateFieldRules({
        appName: 'auth',
        modelName: 'CompanyScopedResource',
        modelFullName: 'auth.CompanyScopedResource',
        roleIds: [roleId],
      });
    },
    { merge: false }
  );

  expect(out.reason).toBe('no_field_rules_deny_by_default');
  expect(out.denyReadFields).toEqual(out.denyWriteFields);
  expect(out.denyReadFields.includes('Name')).toBe(true);
});

test('P4 field rule deny-default: uncovered field is denied when other FR rows exist', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };
  const { actorId, roleId, locationId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);
      const rid = await createRole();
      await grantRoleGlobal(uid1, rid, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');
      const nameFieldId = await resolveFieldId(locationModelId, 'Name');

      // Only Name is allowed; CompanyId and other fields remain uncovered ⇒ denied.
      await RoleFieldRule.Create(
        {
          RoleId: { Id: rid } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'allow',
          PermWrite: 'allow',
        } as any,
        ['Id'] as any
      );

      setReq({ depth: 0, fieldRuleMode: 'skip' });
      const created = await CompanyScopedResource.Create({ Name: uid('csr_dd') }, ['Id'] as any);
      return { actorId: uid1, roleId: rid, locationId: created.Id };
    },
    { merge: false }
  );

  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setReq({ depth: 0, fieldRuleMode: '' });
  setIdentity(actorId);

  const spec = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      return await evaluateFieldRules({
        appName: 'auth',
        modelName: 'CompanyScopedResource',
        modelFullName: 'auth.CompanyScopedResource',
        roleIds: [roleId],
      });
    },
    { merge: false }
  );

  expect(spec.reason).toBe('ok');
  expect(spec.denyReadFields.includes('Name')).toBe(false);
  expect(spec.denyWriteFields.includes('Name')).toBe(false);
  expect(spec.denyReadFields.includes('CompanyId')).toBe(true);
  expect(spec.denyWriteFields.includes('CompanyId')).toBe(true);

  const row = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setReq({ depth: 0, kind: 'grpc', fieldRuleMode: '' });
      return await CompanyScopedResource.Browse(locationId, ['Id', 'Name', 'CompanyId'] as any);
    },
    { merge: false }
  );
  expect(Object.prototype.hasOwnProperty.call(row as any, 'Name')).toBe(true);
  expect(Object.prototype.hasOwnProperty.call(row as any, 'CompanyId')).toBe(false);
});

test('PR-D-2: wide model allow + field-scope deny clamps sensitive column', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };
  const { roleId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);
      const rid = await createRole();
      await grantRoleGlobal(uid1, rid, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');
      const nameFieldId = await resolveFieldId(locationModelId, 'Name');

      await RoleFieldRule.Create(
        {
          RoleId: { Id: rid } as any,
          MetaModelId: locationModelId,
          MetaFieldId: null,
          PermRead: 'allow',
          PermWrite: 'allow',
        } as any,
        ['Id'] as any
      );
      await RoleFieldRule.Create(
        {
          RoleId: { Id: rid } as any,
          MetaModelId: locationModelId,
          MetaFieldId: nameFieldId,
          PermRead: 'deny',
          PermWrite: 'deny',
        } as any,
        ['Id'] as any
      );

      return { roleId: rid };
    },
    { merge: false }
  );

  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setReq({ depth: 0, fieldRuleMode: '' });

  const spec = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      return await evaluateFieldRules({
        appName: 'auth',
        modelName: 'CompanyScopedResource',
        modelFullName: 'auth.CompanyScopedResource',
        roleIds: [roleId],
      });
    },
    { merge: false }
  );

  expect(spec.reason).toBe('ok');
  expect(spec.denyReadFields.includes('Name')).toBe(true);
  expect(spec.denyWriteFields.includes('Name')).toBe(true);
  expect(spec.denyReadFields.includes('CompanyId')).toBe(false);
  expect(spec.denyWriteFields.includes('CompanyId')).toBe(false);
});

test('P4 field rule deny-default: null FR Search result denies all non-system fields', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };
  const { roleId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);
      const rid = await createRole();
      await grantRoleGlobal(uid1, rid, c1.Id);
      return { roleId: rid };
    },
    { merge: false }
  );

  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setReq({ depth: 0, fieldRuleMode: '' });

  const origSearch = (RoleFieldRule as any).Search;
  (RoleFieldRule as any).Search = async () => null;

  try {
    const out = await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        return await evaluateFieldRules({
          appName: 'auth',
          modelName: 'CompanyScopedResource',
          modelFullName: 'auth.CompanyScopedResource',
          roleIds: [roleId],
        });
      },
      { merge: false }
    );

    expect(out.reason).toBe('no_field_rules_deny_by_default');
    expect(out.denyReadFields.includes('Name')).toBe(true);
  } finally {
    (RoleFieldRule as any).Search = origSearch;
  }
});

test('P4 field rule observability: hitRuleIds collect present Ids and tolerate missing Id rows', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };
  const { roleId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);
      const rid = await createRole();
      await grantRoleGlobal(uid1, rid, c1.Id);
      return { roleId: rid };
    },
    { merge: false }
  );

  const modelId = await resolveModelId('auth', 'CompanyScopedResource');
  const nameFieldId = await resolveFieldId(modelId, 'Name');

  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[FR_CACHE_KEY];
  setReq({ depth: 0, fieldRuleMode: '' });

  const origSearch = (RoleFieldRule as any).Search;
  (RoleFieldRule as any).Search = async () => [
    {
      Id: 'fr_hit_1',
      MetaApplicationId: null,
      MetaModelId: modelId,
      MetaFieldId: nameFieldId,
      PermRead: 'allow',
      PermWrite: 'allow',
    },
    {
      Id: '',
      MetaApplicationId: null,
      MetaModelId: modelId,
      MetaFieldId: null,
      PermRead: 'allow',
      PermWrite: 'allow',
    },
  ];

  try {
    const out = await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        return await evaluateFieldRules({
          appName: 'auth',
          modelName: 'CompanyScopedResource',
          modelFullName: 'auth.CompanyScopedResource',
          roleIds: [roleId],
        });
      },
      { merge: false }
    );

    expect(out.reason).toBe('ok');
    expect(out.hitRuleIds).toEqual(['fr_hit_1']);
    expect(out.denyReadFields.includes('Name')).toBe(false);
  } finally {
    (RoleFieldRule as any).Search = origSearch;
  }
});

// ---------------------------------------------------------------------------
// OnchangeMetaModelId coverage
// ---------------------------------------------------------------------------

test('RoleFieldRule OnchangeMetaModelId clears MetaFieldId and narrows picker when model is selected', async () => {
  const result = await RoleFieldRule.Onchange(
    {
      Id: 'onchange-rule-1',
      MetaModelId: 'model-123',
      MetaFieldId: 'field-456',
    },
    ['MetaModelId']
  );

  expect(result.condition).toEqual([{ field: 'MetaFieldId', condition: ['ModelId', '=', 'model-123'] }]);
  expect(result.value).toEqual({ MetaFieldId: null });
});

test('RoleFieldRule OnchangeMetaModelId clears MetaFieldId and blocks picker when model is cleared', async () => {
  const result = await RoleFieldRule.Onchange(
    {
      Id: 'onchange-rule-2',
      MetaModelId: undefined,
      MetaFieldId: 'field-456',
    },
    ['MetaModelId']
  );

  expect(result.condition).toEqual([{ field: 'MetaFieldId', condition: ['Id', '=', '0'] }]);
  expect(result.value).toEqual({ MetaFieldId: null });
});

test('RoleFieldRule OnchangeLogicalModelName clears Meta scopes', async () => {
  const result = await RoleFieldRule.Onchange(
    {
      Id: 'onchange-rule-logical',
      LogicalModelName: 'TranslationTerm',
      MetaApplicationId: 'app-1',
      MetaModelId: 'model-1',
      MetaFieldId: 'field-1',
    },
    ['LogicalModelName']
  );

  expect(result.value).toEqual({
    MetaApplicationId: null,
    MetaModelId: null,
    MetaFieldId: null,
  });
});

test('RoleMethodAccess OnchangeLogicalModelName clears Meta scopes', async () => {
  const RoleMethodAccess = (await import('@/auth/service/models/role_method_access')).default;
  const result = await RoleMethodAccess.Onchange(
    {
      Id: 'onchange-ma-logical',
      LogicalModelName: 'FieldDefault',
      MetaApplicationId: 'app-1',
      MetaModelId: 'model-1',
      MetaServiceId: 'svc-1',
      LogicalMethods: ['Get'],
    },
    ['LogicalModelName']
  );

  expect(result.value).toEqual({
    MetaApplicationId: null,
    MetaModelId: null,
    MetaServiceId: null,
  });
});

test('RoleFieldRule coverage: CreateMany, perm validation branches, and Update paths', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const modelId = await resolveModelId('auth', 'CompanyScopedResource');
      const fieldId = await resolveFieldId(modelId, 'Name');

      const none = await RoleFieldRule.CreateMany(null as any, ['Id'] as any);
      expect(Array.isArray(none)).toBe(true);
      expect(none.length).toBe(0);

      const many = await RoleFieldRule.CreateMany(
        [
          {
            RoleId: { Id: roleId } as any,
            MetaModelId: modelId,
            MetaFieldId: fieldId,
            MetaApplicationId: null,
            PermRead: 'allow',
            PermWrite: 'deny',
          } as any,
        ],
        ['Id', 'PermRead', 'PermWrite'] as any
      );
      expect(many.length).toBe(1);
      const id = String((many[0] as any)?.Id || '').trim();
      expect(id.length > 0).toBe(true);

      // Update with no perm keys: _validatePerms early-returns without rewriting perms.
      await RoleFieldRule.Update(['Id', '=', id] as any, {} as any, ['Id'] as any);
      const afterEmpty = await RoleFieldRule.Search(
        ['Id', '=', id] as any,
        { fields: ['Id', 'PermRead', 'PermWrite'], limit: 1 } as any
      );
      expect(String((afterEmpty[0] as any)?.PermRead || '').trim()).toBe('allow');
      expect(String((afterEmpty[0] as any)?.PermWrite || '').trim()).toBe('deny');

      // Create without any perm rejects.
      let missingPerm = '';
      try {
        await RoleFieldRule.Create(
          {
            RoleId: { Id: roleId } as any,
            MetaModelId: modelId,
            MetaFieldId: fieldId,
            MetaApplicationId: null,
          } as any,
          ['Id'] as any
        );
      } catch (e: any) {
        missingPerm = String(e?.message || e);
      }
      expect(missingPerm.includes('must provide at least one of PermRead/PermWrite')).toBe(true);

      // Update that explicitly clears both perms rejects.
      let clearedBoth = '';
      try {
        await RoleFieldRule.UpdateById(id, { PermRead: null, PermWrite: null } as any, ['Id'] as any);
      } catch (e: any) {
        clearedBoth = String(e?.message || e);
      }
      expect(clearedBoth.includes('must provide at least one of PermRead/PermWrite')).toBe(true);

      // Invalid perm token rejects.
      let badPerm = '';
      try {
        await RoleFieldRule.Create(
          {
            RoleId: { Id: roleId } as any,
            MetaModelId: modelId,
            MetaFieldId: fieldId,
            MetaApplicationId: null,
            PermRead: 'maybe',
          } as any,
          ['Id'] as any
        );
      } catch (e: any) {
        badPerm = String(e?.message || e);
      }
      expect(badPerm.includes('invalid RoleFieldRule perm value')).toBe(true);

      // Create with only PermWrite (no PermRead key) still materializes both columns.
      const writeOnly = await RoleFieldRule.Create(
        {
          RoleId: { Id: roleId } as any,
          MetaModelId: modelId,
          MetaFieldId: null,
          MetaApplicationId: null,
          PermWrite: 'allow',
        } as any,
        ['Id', 'PermRead', 'PermWrite'] as any
      );
      expect(String((writeOnly as any)?.PermWrite || '').trim()).toBe('allow');
      expect((writeOnly as any)?.PermRead == null || String((writeOnly as any)?.PermRead || '').trim() === '').toBe(true);

      // Update with both keys present and non-null (dual-key success path).
      await RoleFieldRule.UpdateById(
        id,
        { PermRead: 'deny', PermWrite: 'allow' } as any,
        ['Id'] as any
      );
      const afterBoth = await RoleFieldRule.Search(
        ['Id', '=', id] as any,
        { fields: ['Id', 'PermRead', 'PermWrite'], limit: 1 } as any
      );
      expect(String((afterBoth[0] as any)?.PermRead || '').trim()).toBe('deny');
      expect(String((afterBoth[0] as any)?.PermWrite || '').trim()).toBe('allow');

      // Blank-string perm normalizes to null on a single-key update (sibling preserved).
      await RoleFieldRule.UpdateById(id, { PermRead: '   ' } as any, ['Id'] as any);
      const afterBlank = await RoleFieldRule.Search(
        ['Id', '=', id] as any,
        { fields: ['Id', 'PermRead', 'PermWrite'], limit: 1 } as any
      );
      expect(String((afterBlank[0] as any)?.PermRead || '').trim()).toBe('');
      expect(String((afterBlank[0] as any)?.PermWrite || '').trim()).toBe('allow');
    },
    { merge: false }
  );
});
