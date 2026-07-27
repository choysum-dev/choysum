// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { withContext as withModelContext } from '@/core/service/api/context';
import CompanyScopedResource from '@/auth/service/models/company_scoped_resource';
import User from '@/auth/service/models/user';
import Role from '@/auth/service/models/role';
import UserRole from '@/auth/service/models/user_role';
import RoleRecordRule from '@/auth/service/models/role_record_rule';
import { createServiceByModel } from '@/core/service/rpc';
import type IrApplicationModel from '@/meta/service/models/ir_application';
import type IrModelModel from '@/meta/service/models/ir_model';
const IrApplication = createServiceByModel<typeof IrApplicationModel>('meta.IrApplication');
const IrModel = createServiceByModel<typeof IrModelModel>('meta.IrModel');

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
  // This file focuses on RecordRule.
  // Avoid letting FieldRule fetch specs during create or update, otherwise missing userId
  // would fail before the RecordRule assertions run.
  jsCtx.req = { depth: 0, fieldRuleMode: 'skip' };
  jsCtx.identity = {};

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
  const rows = await IrModel.Search(
    {
      And: [
        ['Name', '=', modelName],
        ['Application', '=', appName],
      ],
    } as any,
    { fields: ['Id'], limit: 1 }
  );
  const id = String((rows as any)?.[0]?.Id || '').trim();
  if (!id) throw new Error(`meta model not found: ${appName}.${modelName}`);
  return id;
}

async function resolveApplicationId(appName: string): Promise<string> {
  const rows = await IrApplication.Search({ And: [['Name', '=', appName]] } as any, { fields: ['Id'], limit: 1 } as any);
  const id = String((rows as any)?.[0]?.Id || '').trim();
  if (!id) throw new Error(`meta application not found: ${appName}`);
  return id;
}

async function createUser(companyId: string): Promise<string> {
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
}

async function createRole(): Promise<string> {
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
}

async function grantRoleGlobal(userId: string, roleId: string, companyId: string): Promise<void> {
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
}

function setupAllowlistForFixtures(): void {
  // RecordRule allowlist only applies at depth=0.
  // Use it for fixture writes so record rules do not block their own support models.
  setReq({
    depth: 0,
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      // Allowlist keys must be `${model}:${op}`; model may be the full or short model name.
      // To avoid misses from internal BaseModel.Create or Browse chains in the runner,
      // allow all ops for models used by fixtures. This only affects allowlist handling
      // and does not weaken later record-rule assertions.
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

      'auth.RoleRecordRule:read',
      'auth.RoleRecordRule:write',
      'auth.RoleRecordRule:create',
      'auth.RoleRecordRule:delete',
      'RoleRecordRule:read',
      'RoleRecordRule:write',
      'RoleRecordRule:create',
      'RoleRecordRule:delete',

      'auth.CompanyScopedResource:read',
      'auth.CompanyScopedResource:write',
      'auth.CompanyScopedResource:create',
      'auth.CompanyScopedResource:delete',
      'CompanyScopedResource:read',
      'CompanyScopedResource:write',
      'CompanyScopedResource:create',
      'CompanyScopedResource:delete',
    ],
  });
}

function disableAllowlist(): void {
  setReq({ recordRuleMode: '', recordRuleAllow: [] });
}

/**
 * Capture console.warn messages while running fn; always restore console.warn.
 * The warnings array is mutable so callers can clear it between steps.
 */
async function withWarnCapture(fn: (warnings: string[]) => Promise<void>): Promise<string[]> {
  const warnings: string[] = [];
  const originalWarn = console.warn;
  console.warn = ((...args: unknown[]) => {
    warnings.push(args.map(a => String(a)).join(' '));
  }) as typeof console.warn;
  try {
    await fn(warnings);
  } finally {
    console.warn = originalWarn;
  }
  return warnings;
}

test('P2-2 record rule: missing identity.userId => read returns empty set (no throw)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  // ====== fixtures (with allowlist) ======
  setupAllowlistForFixtures();
  const userId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');

      // Allow create only (empty Condition is treated as TRUE).
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          IrModelId: locationModelId,
          Condition: null as any,
          PermCreate: true,
          PermRead: false,
          PermWrite: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      const created = await CompanyScopedResource.Create({ Name: uid('csr_rr') }, ['Id'] as any);
      return uid1;
    },
    { merge: false }
  );

  // ====== assertion (disable allowlist; drop identity) ======
  disableAllowlist();
  setIdentity(undefined);

  const names = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const rows = await CompanyScopedResource.Search([], { fields: ['Id', 'Name'] as any });
      return rows.map(r => String((r as any).Name || '')).filter(n => n.includes('csr_rr'));
    },
    { merge: false }
  );

  expect(Array.isArray(names)).toBe(true);
  expect(names.length).toBe(0);

  // Keep userId referenced to satisfy lint and confirm fixture creation succeeded.
  expect(typeof userId).toBe('string');
});

test('P2-2 record rule: no write rules => write is not additionally restricted', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();

  const { actorId, locationId } = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');

      // Allow create only; do not allow write.
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          IrModelId: locationModelId,
          Condition: null as any,
          PermCreate: true,
          PermRead: false,
          PermWrite: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      const created = await CompanyScopedResource.Create({ Name: uid('csr_wr') }, ['Id'] as any);
      return { actorId: uid1, locationId: created.Id };
    },
    { merge: false }
  );

  disableAllowlist();
  setIdentity(actorId);

  await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const newName = uid('csr_wr_new');
      // Use static Update with argument order: (condition, values, returnFields).
      await CompanyScopedResource.Update(['Id', '=', locationId] as any, { Name: newName } as any, []);
      const rows = await CompanyScopedResource.Search(['Id', '=', locationId] as any, { fields: ['Name'] as any, limit: 1 } as any);
      const got = String((rows as any)?.[0]?.Name || '');
      expect(got).toBe(newName);
    },
    { merge: false }
  );
});

test('P2-2 record rule wildcard: model scope overrides application TRUE (pick-one)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();

  const actorId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const appId = await resolveApplicationId('auth');
      const modelId = await resolveModelId('auth', 'CompanyScopedResource');

      const allowName = uid('csr_mwin_allow');
      const denyName = uid('csr_mwin_deny');

      // Application scope: TRUE (unconditional)
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          IrModelId: null,
          IrApplicationId: appId,
          Condition: null as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      // Model scope: restrictive (must win over app scope)
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          IrModelId: modelId,
          IrApplicationId: null,
          Condition: { And: [['Name', '=', allowName]] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      await CompanyScopedResource.Create({ Name: allowName } as any, ['Id'] as any);
      await CompanyScopedResource.Create({ Name: denyName } as any, ['Id'] as any);

      return uid1;
    },
    { merge: false }
  );

  disableAllowlist();
  setIdentity(actorId);

  const names = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const rows = await CompanyScopedResource.Search([], { fields: ['Name'] as any });
      return rows.map(r => String((r as any).Name || '')).filter(Boolean);
    },
    { merge: false }
  );

  expect(names.some(n => n.includes('csr_mwin_allow'))).toBe(true);
  expect(names.some(n => n.includes('csr_mwin_deny'))).toBe(false);
});

test('P2-2 record rule wildcard: application scope overrides global (pick-one)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();

  const actorId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const appId = await resolveApplicationId('auth');
      const allowName = uid('csr_awin_allow');
      const denyName = uid('csr_awin_deny');

      // Global scope: TRUE
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          IrModelId: null,
          IrApplicationId: null,
          Condition: null as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      // Application scope: restrictive (must win over global TRUE)
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          IrModelId: null,
          IrApplicationId: appId,
          Condition: { And: [['Name', '=', allowName]] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      await CompanyScopedResource.Create({ Name: allowName } as any, ['Id'] as any);
      await CompanyScopedResource.Create({ Name: denyName } as any, ['Id'] as any);

      return uid1;
    },
    { merge: false }
  );

  disableAllowlist();
  setIdentity(actorId);

  const names = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const rows = await CompanyScopedResource.Search([], { fields: ['Name'] as any });
      return rows.map(r => String((r as any).Name || '')).filter(Boolean);
    },
    { merge: false }
  );

  expect(names.some(n => n.includes('csr_awin_allow'))).toBe(true);
  expect(names.some(n => n.includes('csr_awin_deny'))).toBe(false);
});

test('P2-2 record rule wildcard: scope-local OR merge (model scope)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();

  const actorId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const modelId = await resolveModelId('auth', 'CompanyScopedResource');
      const a = uid('csr_or_a');
      const b = uid('csr_or_b');
      const c = uid('csr_or_c');

      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          IrModelId: modelId,
          IrApplicationId: null,
          Condition: { And: [['Name', '=', a]] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          IrModelId: modelId,
          IrApplicationId: null,
          Condition: { And: [['Name', '=', b]] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      await CompanyScopedResource.Create({ Name: a } as any, ['Id'] as any);
      await CompanyScopedResource.Create({ Name: b } as any, ['Id'] as any);
      await CompanyScopedResource.Create({ Name: c } as any, ['Id'] as any);

      return uid1;
    },
    { merge: false }
  );

  disableAllowlist();
  setIdentity(actorId);

  const names = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const rows = await CompanyScopedResource.Search([], { fields: ['Name'] as any });
      return rows.map(r => String((r as any).Name || '')).filter(Boolean);
    },
    { merge: false }
  );

  expect(names.some(n => n.includes('csr_or_a'))).toBe(true);
  expect(names.some(n => n.includes('csr_or_b'))).toBe(true);
  expect(names.some(n => n.includes('csr_or_c'))).toBe(false);
});

test('P2-2 record rule: unknown token in condition throws PermissionDenied', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();

  const actorId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();
      await grantRoleGlobal(uid1, roleId, c1.Id);

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');

      // The read rule contains an unknown token, which should trigger fail-closed repo token replacement.
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          IrModelId: locationModelId,
          Condition: { And: [['Id', '=', '$notARealToken']] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      // Insert one row so Search has data to evaluate.
      await CompanyScopedResource.Create({ Name: uid('csr_tok') }, ['Id'] as any);
      return uid1;
    },
    { merge: false }
  );

  disableAllowlist();
  setIdentity(actorId);

  try {
    await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        await CompanyScopedResource.Search([], { fields: ['Id'] as any });
      },
      { merge: false }
    );
    throw new Error('expected search to throw');
  } catch (err) {
    const oe = toChoysumErrorLike(err);
    expect(Boolean(oe)).toBe(true);
    expect(String(oe?.domain || '')).toBe('core.repository');
    expect(String(oe?.code || '')).toBe('record_rule_unknown_token');
  }
});

test('P3-2 company-scoped roles: record rule gated by role company scope (no cross-company leakage)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  // ====== fixtures (with allowlist) ======
  setupAllowlistForFixtures();
  const actorId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id, c2.Id] } as any,
    async () => {
      const uid1 = await createUser(c1.Id);
      setIdentity(uid1);

      const roleId = await createRole();

      // grant role scoped to c2 only
      await UserRole.Create(
        {
          UserId: { Id: uid1 } as any,
          RoleId: { Id: roleId } as any,
          CompanyId: c2.Id,
        } as any,
        ['Id'] as any
      );

      const locationModelId = await resolveModelId('auth', 'CompanyScopedResource');

      // Use a restrictive rule so we can observe whether a role scoped to c2
      // mistakenly affects queries when activeCompanyId=c1.
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          IrModelId: locationModelId,
          Condition: { And: [['CompanyId', '=', c2.Id]] } as any,
          PermRead: true,
          PermWrite: false,
          PermCreate: false,
          PermDelete: false,
        } as any,
        ['Id'] as any
      );

      await CompanyScopedResource.Create({ Name: uid('csr_c1'), CompanyId: c1.Id } as any, ['Id'] as any);
      await CompanyScopedResource.Create({ Name: uid('csr_c2'), CompanyId: c2.Id } as any, ['Id'] as any);
      await CompanyScopedResource.Create({ Name: uid('csr_shared'), CompanyId: null as any } as any, ['Id'] as any);

      return uid1;
    },
    { merge: false }
  );

  // ====== assertion (disable allowlist; enforce real record rule) ======
  disableAllowlist();
  setIdentity(actorId);

  // When activeCompanyId=c1, the user only has a role scoped to c2.
  // Correct behavior: that role's record rule must NOT apply => results should NOT be
  // unexpectedly restricted to c2.
  const namesC1 = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id, c2.Id] } as any,
    async () => {
      const rows = await CompanyScopedResource.Search([], { fields: ['Name'] as any });
      return rows.map(r => String((r as any).Name || '')).filter(Boolean);
    },
    { merge: false }
  );
  expect(namesC1.some(n => n.includes('csr_c1'))).toBe(true);
  expect(namesC1.some(n => n.includes('csr_c2'))).toBe(true);
  expect(namesC1.some(n => n.includes('csr_shared'))).toBe(true);

  // When activeCompanyId=c2, the scoped role should apply.
  const namesC2 = await withModelContext(
    { activeCompanyId: c2.Id, enabledCompanyIds: [c1.Id, c2.Id] } as any,
    async () => {
      const rows = await CompanyScopedResource.Search([], { fields: ['Name'] as any });
      return rows.map(r => String((r as any).Name || '')).filter(Boolean);
    },
    { merge: false }
  );
  expect(namesC2.some(n => n.includes('csr_c1'))).toBe(false);
  expect(namesC2.some(n => n.includes('csr_c2'))).toBe(true);
  expect(namesC2.some(n => n.includes('csr_shared'))).toBe(false);
});

test('RoleRecordRule db check: deleted rows bypass scope xor', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const roleId = await createRole();
    const modelId = await resolveModelId('auth', 'CompanyScopedResource');
    const appId = await resolveApplicationId('auth');

    const row = await RoleRecordRule.Create(
      {
        RoleId: { Id: roleId } as any,
        IrModelId: modelId,
        IrApplicationId: null,
        Condition: { And: [['CompanyId', '=', uid('C')]] } as any,
        PermRead: true,
        PermWrite: false,
        PermCreate: false,
        PermDelete: false,
      } as any,
      ['Id'] as any
    );

    const id = String((row as any)?.Id || '').trim();
    expect(id.length > 0).toBe(true);

    await RoleRecordRule.DeleteById(id);

    const repo = RoleRecordRule.getRepository().withDeleted();
    const updated = await repo.update(
      {
        IrModelId: modelId,
        IrApplicationId: appId,
      } as any,
      ['Id', '=', id] as any
    );

    expect((updated || []).length).toBe(1);

    const rows = await RoleRecordRule.Search(
      ['Id', '=', id] as any,
      {
        fields: ['Id', 'DeletedAt', 'IrModelId', 'IrApplicationId'] as any,
        withDeleted: true,
      } as any
    );

    expect(rows.length).toBe(1);
    expect(String((rows[0] as any)?.IrModelId || '').trim()).toBe(modelId);
    expect(String((rows[0] as any)?.IrApplicationId || '').trim()).toBe(appId);
    expect((rows[0] as any)?.DeletedAt != null).toBe(true);
  });
});

test('RoleRecordRule: permission-only update must not rewrite scoped fields to global', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const roleId = await createRole();
    const modelId = await resolveModelId('auth', 'CompanyScopedResource');

    const created = await RoleRecordRule.Create(
      {
        RoleId: { Id: roleId } as any,
        IrModelId: modelId,
        IrApplicationId: null,
        Condition: { And: [['CompanyId', '=', uid('C')]] } as any,
        PermRead: true,
        PermWrite: false,
        PermCreate: false,
        PermDelete: false,
      } as any,
      ['Id'] as any
    );

    const id = String((created as any)?.Id || '').trim();
    expect(id.length > 0).toBe(true);

    await RoleRecordRule.UpdateById(
      id,
      {
        PermRead: false,
        PermWrite: true,
      } as any,
      ['Id'] as any
    );

    const rows = await RoleRecordRule.Search(
      ['Id', '=', id] as any,
      { fields: ['Id', 'IrModelId', 'IrApplicationId', 'PermRead', 'PermWrite'], limit: 1 } as any
    );

    expect(rows.length).toBe(1);
    // Scope must stay scoped to the model, not become global.
    expect(String((rows[0] as any)?.IrModelId || '').trim()).toBe(modelId);
    expect(String((rows[0] as any)?.IrApplicationId || '').trim()).toBe('');
    // Permissions must reflect the update.
    expect((rows[0] as any)?.PermRead).toBe(false);
    expect((rows[0] as any)?.PermWrite).toBe(true);
  });
});

test('RoleRecordRule Kind: create defaults to grant and accepts restrict', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const roleId = await createRole();
    const modelId = await resolveModelId('auth', 'CompanyScopedResource');

    const granted = await RoleRecordRule.Create(
      {
        RoleId: { Id: roleId } as any,
        IrModelId: modelId,
        IrApplicationId: null,
        Condition: null,
        PermRead: true,
      } as any,
      ['Id', 'Kind', 'RoleId'] as any
    );

    expect(String((granted as any)?.Kind || '')).toBe('grant');
    expect(String((granted as any)?.RoleId?.Id || (granted as any)?.RoleId || '').trim()).toBe(roleId);

    const restricted = await RoleRecordRule.Create(
      {
        RoleId: { Id: roleId } as any,
        Kind: 'restrict',
        IrModelId: modelId,
        IrApplicationId: null,
        Condition: { And: [['Name', '!=', 'done']] } as any,
        PermWrite: true,
      } as any,
      ['Id', 'Kind'] as any
    );

    expect(String((restricted as any)?.Kind || '')).toBe('restrict');

    const rows = await RoleRecordRule.Search(
      ['Id', 'in', [String((granted as any).Id), String((restricted as any).Id)]] as any,
      { fields: ['Id', 'Kind'], limit: 10 } as any
    );
    const byId = new Map(rows.map((r: any) => [String(r.Id), String(r.Kind)]));
    expect(byId.get(String((granted as any).Id))).toBe('grant');
    expect(byId.get(String((restricted as any).Id))).toBe('restrict');
  });
});

test('RoleRecordRule Kind: rejects invalid values', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const roleId = await createRole();
    const modelId = await resolveModelId('auth', 'CompanyScopedResource');

    let threw = false;
    try {
      await RoleRecordRule.Create(
        {
          RoleId: { Id: roleId } as any,
          Kind: 'open',
          IrModelId: modelId,
          IrApplicationId: null,
          PermRead: true,
        } as any,
        ['Id'] as any
      );
    } catch (err: any) {
      threw = true;
      expect(String(err?.message || err)).toContain("invalid RoleRecordRule Kind: must be 'grant' or 'restrict'");
    }
    expect(threw).toBe(true);
  });
});

test('RoleRecordRule RoleId: null means everyone and warns on grant', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withWarnCapture(async warnings => {
    await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
      const modelId = await resolveModelId('auth', 'CompanyScopedResource');

      const row = await RoleRecordRule.Create(
        {
          RoleId: null,
          Kind: 'grant',
          IrModelId: modelId,
          IrApplicationId: null,
          Condition: null,
          PermRead: true,
        } as any,
        ['Id', 'Kind', 'RoleId'] as any
      );

      const id = String((row as any)?.Id || '').trim();
      expect(id.length > 0).toBe(true);
      expect(String((row as any)?.Kind || '')).toBe('grant');
      expect((row as any)?.RoleId == null || (row as any)?.RoleId === '').toBe(true);

      const rows = await RoleRecordRule.Search(
        ['Id', '=', id] as any,
        { fields: ['Id', 'Kind', 'RoleId', 'IrModelId'], limit: 1 } as any
      );
      expect(rows.length).toBe(1);
      expect(String((rows[0] as any)?.Kind || '')).toBe('grant');
      expect((rows[0] as any)?.RoleId == null || (rows[0] as any)?.RoleId === '').toBe(true);
      expect(String((rows[0] as any)?.IrModelId || '').trim()).toBe(modelId);

      expect(warnings.some(w => w.includes('Kind=grant') && w.includes('RoleId=null'))).toBe(true);
    });
  });
});

test('RoleRecordRule RoleId: whitespace string normalizes to null (everyone)', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const modelId = await resolveModelId('auth', 'CompanyScopedResource');

    const row = await RoleRecordRule.Create(
      {
        RoleId: '   ' as any,
        Kind: 'restrict',
        IrModelId: modelId,
        IrApplicationId: null,
        Condition: null,
        PermRead: true,
      } as any,
      ['Id', 'RoleId', 'Kind'] as any
    );

    expect((row as any)?.RoleId == null || (row as any)?.RoleId === '').toBe(true);

    const rows = await RoleRecordRule.Search(
      ['Id', '=', String((row as any).Id)] as any,
      { fields: ['Id', 'RoleId'], limit: 1 } as any
    );
    expect(rows.length).toBe(1);
    expect((rows[0] as any)?.RoleId == null || (rows[0] as any)?.RoleId === '').toBe(true);
  });
});

test('RoleRecordRule RoleId: restrict with null RoleId does not emit grant warn', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withWarnCapture(async warnings => {
    await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
      const modelId = await resolveModelId('auth', 'CompanyScopedResource');

      const row = await RoleRecordRule.Create(
        {
          RoleId: null,
          Kind: 'restrict',
          IrModelId: modelId,
          IrApplicationId: null,
          Condition: { And: [['Name', '!=', 'x']] } as any,
          PermWrite: true,
        } as any,
        ['Id', 'Kind', 'RoleId'] as any
      );

      expect(String((row as any)?.Kind || '')).toBe('restrict');
      expect(warnings.some(w => w.includes('wide-open grant'))).toBe(false);
    });
  });
});

test('RoleRecordRule coverage: empty RoleId string, CreateMany null, Field Kind default', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const roleId = await createRole();
    const modelId = await resolveModelId('auth', 'CompanyScopedResource');

    // Exact empty string RoleId hits `raw === ''` branch (not whitespace).
    const emptyStr = await RoleRecordRule.Create(
      {
        RoleId: '' as any,
        Kind: 'restrict',
        IrModelId: modelId,
        IrApplicationId: null,
        PermRead: true,
      } as any,
      ['Id', 'RoleId', 'Kind'] as any
    );
    // Must normalize away to null — do not accept persisted ''.
    expect((emptyStr as any)?.RoleId == null).toBe(true);

    // CreateMany(null) uses `values || []` then persists empty list.
    const none = await RoleRecordRule.CreateMany(null as any, ['Id'] as any);
    expect(Array.isArray(none)).toBe(true);
    expect(none.length).toBe(0);

    // Omit Kind entirely so Field default: () => 'grant' is applied by persistence.
    const withDefault = await RoleRecordRule.Create(
      {
        RoleId: { Id: roleId } as any,
        IrModelId: modelId,
        IrApplicationId: null,
        PermRead: true,
      } as any,
      ['Id', 'Kind'] as any
    );
    expect(String((withDefault as any)?.Kind || '')).toBe('grant');

    // Kind present but nullish hits `_normalizeKind`'s `v ?? 'grant'` branch.
    const nullKind = await RoleRecordRule.Create(
      {
        RoleId: { Id: roleId } as any,
        Kind: null as any,
        IrModelId: modelId,
        IrApplicationId: null,
        PermRead: true,
      } as any,
      ['Id', 'Kind'] as any
    );
    expect(String((nullKind as any)?.Kind || '')).toBe('grant');
  });
});

test('RoleRecordRule coverage: CreateMany, blank object RoleId, Update Kind/RoleId paths', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withWarnCapture(async warnings => {
    await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
      const roleId = await createRole();
      const modelId = await resolveModelId('auth', 'CompanyScopedResource');

      // CreateMany with concrete RoleId + grant must not warn (isEveryone=false).
      const many = await RoleRecordRule.CreateMany(
        [
          {
            RoleId: { Id: roleId } as any,
            Kind: 'grant',
            IrModelId: modelId,
            IrApplicationId: null,
            PermRead: true,
          } as any,
        ],
        ['Id', 'Kind', 'RoleId'] as any
      );
      expect(many.length).toBe(1);
      expect(String((many[0] as any)?.Kind || '')).toBe('grant');
      expect(warnings.some(w => w.includes('wide-open grant'))).toBe(false);

      // Blank object RoleId → null (everyone) + grant → warn.
      warnings.length = 0;
      const blankRole = await RoleRecordRule.Create(
        {
          RoleId: { Id: '   ' } as any,
          Kind: 'grant',
          IrModelId: modelId,
          IrApplicationId: null,
          PermRead: true,
        } as any,
        ['Id', 'RoleId'] as any
      );
      expect((blankRole as any)?.RoleId == null || (blankRole as any)?.RoleId === '').toBe(true);
      expect(warnings.some(w => w.includes('Kind=grant') && w.includes('RoleId=null'))).toBe(true);

      // Create omitting RoleId key entirely → everyone grant warn.
      warnings.length = 0;
      const omittedRole = await RoleRecordRule.Create(
        {
          Kind: 'grant',
          IrModelId: modelId,
          IrApplicationId: null,
          PermRead: true,
        } as any,
        ['Id', 'RoleId'] as any
      );
      expect((omittedRole as any)?.RoleId == null || (omittedRole as any)?.RoleId === '').toBe(true);
      expect(warnings.some(w => w.includes('wide-open grant'))).toBe(true);

      const id = String((many[0] as any).Id || '').trim();
      expect(id.length > 0).toBe(true);

      // Update Kind without RoleId (row still has concrete RoleId): early-return, no warn.
      warnings.length = 0;
      await RoleRecordRule.UpdateById(id, { Kind: 'grant' } as any, ['Id', 'Kind'] as any);
      expect(warnings.some(w => w.includes('wide-open grant'))).toBe(false);

      // Documented false-negative: RoleId already null, Kind-only flip to grant → no warn
      // (helper does not load persisted row when RoleId is omitted from the payload).
      const everyoneRestrict = await RoleRecordRule.Create(
        {
          RoleId: null,
          Kind: 'restrict',
          IrModelId: modelId,
          IrApplicationId: null,
          PermWrite: true,
        } as any,
        ['Id', 'Kind', 'RoleId'] as any
      );
      const everyoneId = String((everyoneRestrict as any)?.Id || '').trim();
      expect(everyoneId.length > 0).toBe(true);
      warnings.length = 0;
      await RoleRecordRule.UpdateById(everyoneId, { Kind: 'grant' } as any, ['Id', 'Kind'] as any);
      expect(warnings.some(w => w.includes('wide-open grant'))).toBe(false);

      // Update clearing RoleId with Kind=grant → warn.
      warnings.length = 0;
      await RoleRecordRule.Update(['Id', '=', id] as any, { Kind: 'grant', RoleId: null } as any, ['Id', 'RoleId'] as any);
      expect(warnings.some(w => w.includes('Kind=grant') && w.includes('RoleId=null'))).toBe(true);

      // Update Kind to restrict with RoleId string id (non-object branch).
      await RoleRecordRule.UpdateById(
        id,
        {
          Kind: 'restrict',
          RoleId: roleId,
        } as any,
        ['Id', 'Kind', 'RoleId'] as any
      );
      const rows = await RoleRecordRule.Search(
        ['Id', '=', id] as any,
        { fields: ['Id', 'Kind', 'RoleId'], limit: 1 } as any
      );
      expect(rows.length).toBe(1);
      expect(String((rows[0] as any)?.Kind || '')).toBe('restrict');
      expect(String((rows[0] as any)?.RoleId?.Id || (rows[0] as any)?.RoleId || '').trim()).toBe(roleId);

      // Invalid Kind on UpdateById.
      let threw = false;
      try {
        await RoleRecordRule.UpdateById(id, { Kind: 'open' } as any, ['Id'] as any);
      } catch (err: any) {
        threw = true;
        expect(String(err?.message || err)).toContain("invalid RoleRecordRule Kind: must be 'grant' or 'restrict'");
      }
      expect(threw).toBe(true);
    });
  });
});
