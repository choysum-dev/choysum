// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Role from '@/auth/service/models/role';
import { resolveEffectiveModelId } from '@/auth/service/models/_resolve_effective_model';
import RoleFieldRule from '@/auth/service/models/role_field_rule';
import RoleRecordRule from '@/auth/service/models/role_record_rule';
import User from '@/auth/service/models/user';
import UserRole from '@/auth/service/models/user_role';
import { withPermissionGraphBypass } from '@/auth/service/models/_user_authz_shared';
import { invalidateAuthzCachesForUsers } from '@/auth/service/models/_request_cache_invalidation';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

const FR_FIXTURE_ROLE_CODE = 'document.fixture.fr_allow';
const KNOWN_DOCUMENT_TEST_USER_IDS = ['usr_document_test', 'usr_scope_a', 'usr_scope_b'];

let authUserOwnerGrantsSeeded = false;
let createdOwnerGrantRuleId = '';
let createdOwnerFrRuleId = '';
let createdOwnerFrRoleId = '';
/** Resolution memo (may point at reused pre-existing rows that teardown must not delete). */
let resolvedOwnerFrRoleId = '';
let ownerFrRuleEnsured = false;
const createdOwnerFrUserRoleIds: string[] = [];
const createdOwnerFrUserIds: string[] = [];
let previousRecordRuleEnabled: unknown = undefined;
let capturedRecordRuleEnv = false;
let previousFieldRuleEnabled: unknown = undefined;
let capturedFieldRuleEnv = false;

function isEmptyCondition(cond: unknown): boolean {
  if (cond == null) return true;
  if (typeof cond !== 'object' || Array.isArray(cond)) return false;
  const keys = Object.keys(cond as Record<string, unknown>);
  if (keys.length === 0) return true;
  if (keys.length === 1) {
    const key = keys[0];
    const val = (cond as Record<string, unknown>)[key];
    if ((key === 'And' || key === 'Or') && Array.isArray(val) && val.length === 0) return true;
  }
  return false;
}

function isNotFoundDeleteError(err: unknown): boolean {
  const code = String((err as any)?.code || '').toLowerCase();
  if (code === 'not_found' || code === 'notfound' || code === '5') return true;
  const msg = String((err as any)?.message || err || '').toLowerCase();
  return msg.includes('not found') || msg.includes('not_found');
}

function clearRequestAuthzCaches(): void {
  const root: any = (globalThis as any).$choysum ?? {};
  const jsCtx = root?.request?.context;
  if (jsCtx) {
    delete jsCtx[RR_CACHE_KEY];
    delete jsCtx[FR_CACHE_KEY];
  }
}

function currentIdentityUserId(): string {
  const root: any = (globalThis as any).$choysum ?? {};
  return String(root?.request?.context?.identity?.userId || '').trim();
}

/**
 * Document unit tests historically relied on repository RecordRule allow-by-default
 * for UploadSession / Binding / Content CRUD. Under deny-default those writes need
 * either grant packs or a repository-layer disable. Owner authorization still calls
 * GetRecordRuleCondition directly (unaffected by this flag).
 */
export function disableRepositoryRecordRuleForDocumentTests(): void {
  const root = globalThis as any;
  const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? { ...root.__CHOYSUM_RUNTIME_ENV__ } : {};
  if (!capturedRecordRuleEnv) {
    previousRecordRuleEnabled = prev.CHOYSUM_GRPC_RECORD_RULE_ENABLED;
    capturedRecordRuleEnv = true;
  }
  root.__CHOYSUM_RUNTIME_ENV__ = { ...prev, CHOYSUM_GRPC_RECORD_RULE_ENABLED: false };
}

/**
 * Mirror RecordRule disable for repository FieldRule: nested Create/Update at depth>0
 * ignores top-level fieldRuleMode=skip, so deny-default would block UploadSession rows.
 * Owner authorization still calls GetFieldRuleSpec directly (unaffected by this flag).
 */
export function disableRepositoryFieldRuleForDocumentTests(): void {
  const root = globalThis as any;
  const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? { ...root.__CHOYSUM_RUNTIME_ENV__ } : {};
  if (!capturedFieldRuleEnv) {
    previousFieldRuleEnabled = prev.CHOYSUM_GRPC_FIELD_RULE_ENABLED;
    capturedFieldRuleEnv = true;
  }
  root.__CHOYSUM_RUNTIME_ENV__ = { ...prev, CHOYSUM_GRPC_FIELD_RULE_ENABLED: false };
}

/**
 * Seed an everyone grant on auth.User (read+write only) so document owner-authorization
 * happy paths work under RecordRule deny-default. Deny cases that use unknown.Model
 * remain unaffected. Tracks the created id for optional teardown.
 */
export async function ensureAuthUserOwnerRecordRuleGrants(): Promise<void> {
  if (authUserOwnerGrantsSeeded) return;

  await withPermissionGraphBypass(async () => {
    const modelId = await resolveEffectiveModelId('auth', 'User');
    if (!modelId) {
      throw new Error('meta model auth.User not found for document owner RR fixture');
    }

    // Reuse only an exact fixture-shaped grant (R/W only, empty Condition).
    const existing = await RoleRecordRule.Search(
      {
        And: [
          ['RoleId', 'is', null],
          ['Kind', '=', 'grant'],
          ['MetaModelId', '=', modelId],
          ['MetaApplicationId', 'is', null],
          ['PermRead', '=', true],
          ['PermWrite', '=', true],
          ['PermCreate', '=', false],
          ['PermDelete', '=', false],
        ],
      } as any,
      { fields: ['Id', 'Condition'], limit: 8 } as any
    );
    const reusable = (existing || []).find(r => isEmptyCondition((r as any)?.Condition));
    if (reusable) {
      // Pre-existing exact-shape rule; do not delete it on teardown.
      return;
    }

    const created = await RoleRecordRule.Create(
      {
        RoleId: null as any,
        Kind: 'grant',
        MetaModelId: modelId,
        MetaApplicationId: null,
        Condition: { And: [] } as any,
        PermRead: true,
        PermWrite: true,
        PermCreate: false,
        PermDelete: false,
      } as any,
      ['Id'] as any
    );
    createdOwnerGrantRuleId = String((created as any)?.Id || '').trim();
  });

  authUserOwnerGrantsSeeded = true;
  clearRequestAuthzCaches();
}

/**
 * Seed a fixture role with global FieldRule allow and attach it to document test
 * identities so GetFieldRuleSpec happy paths work under FieldRule deny-default.
 * Explicit field-deny cases (extra RoleFieldRule on admin) still win via specificity.
 *
 * Safe to call repeatedly: role/rule creation is idempotent; UserRole grants are
 * ensured for the current identity on every call.
 */
export async function ensureAuthUserOwnerFieldRuleGrants(): Promise<void> {
  const userIds = new Set<string>(KNOWN_DOCUMENT_TEST_USER_IDS);
  const current = currentIdentityUserId();
  if (current) userIds.add(current);

  await withPermissionGraphBypass(async () => {
    const roleId = await ensureFixtureFrRole();
    await ensureFixtureFrRule(roleId);

    for (const userId of userIds) {
      await ensureFixtureUser(userId);
      await ensureFixtureUserRole(userId, roleId);
    }

    await invalidateAuthzCachesForUsers(Array.from(userIds));
  });

  clearRequestAuthzCaches();
}

async function ensureFixtureFrRole(): Promise<string> {
  if (resolvedOwnerFrRoleId) return resolvedOwnerFrRoleId;

  // Include soft-deleted rows: Code is UNIQUE across deleted rows, so a prior
  // suite teardown that soft-deleted this fixture would otherwise make Create fail.
  const existingRoles = await Role.Search(['Code', '=', FR_FIXTURE_ROLE_CODE] as any, {
    fields: ['Id', 'DeletedAt'],
    limit: 1,
    withDeleted: true,
  } as any);
  const existing = (existingRoles as any)?.[0];
  const existingId = String(existing?.Id || '').trim();
  if (existingId) {
    if (existing?.DeletedAt != null) {
      const repo = Role.getRepository().withDeleted();
      await repo.update({ DeletedAt: null, IsActive: true } as any, ['Id', '=', existingId] as any);
    }
    resolvedOwnerFrRoleId = existingId;
    return existingId;
  }

  const createdRole = await Role.Create(
    {
      Name: 'Document Fixture FR Allow',
      Code: FR_FIXTURE_ROLE_CODE,
      Description: 'test fixture: global field-rule allow for document owner auth',
      IsActive: true,
      IsSystem: false,
    } as any,
    ['Id'] as any
  );
  createdOwnerFrRoleId = String((createdRole as any)?.Id || '').trim();
  if (!createdOwnerFrRoleId) throw new Error('failed to create document FR fixture role');
  resolvedOwnerFrRoleId = createdOwnerFrRoleId;
  return createdOwnerFrRoleId;
}

async function ensureFixtureFrRule(roleId: string): Promise<void> {
  if (ownerFrRuleEnsured) return;

  const existingFr = await RoleFieldRule.Search(
    {
      And: [
        ['RoleId', '=', roleId],
        ['MetaApplicationId', 'is', null],
        ['MetaModelId', 'is', null],
        ['MetaFieldId', 'is', null],
        ['PermRead', '=', 'allow'],
        ['PermWrite', '=', 'allow'],
      ],
    } as any,
    { fields: ['Id', 'DeletedAt'], limit: 1, withDeleted: true } as any
  );
  const existing = (existingFr as any)?.[0];
  const existingId = String(existing?.Id || '').trim();
  if (existingId) {
    if (existing?.DeletedAt != null) {
      const repo = RoleFieldRule.getRepository().withDeleted();
      await repo.update({ DeletedAt: null } as any, ['Id', '=', existingId] as any);
    }
    // Pre-existing / revived rule; leave teardown alone.
    ownerFrRuleEnsured = true;
    return;
  }

  const createdFr = await RoleFieldRule.Create(
    {
      RoleId: { Id: roleId } as any,
      MetaApplicationId: null,
      MetaModelId: null,
      MetaFieldId: null,
      PermRead: 'allow',
      PermWrite: 'allow',
    } as any,
    ['Id'] as any
  );
  createdOwnerFrRuleId = String((createdFr as any)?.Id || '').trim();
  if (!createdOwnerFrRuleId) throw new Error('failed to create document FR fixture rule');
  ownerFrRuleEnsured = true;
}

async function ensureFixtureUser(userId: string): Promise<void> {
  const existing = await User.Search(['Id', '=', userId] as any, {
    fields: ['Id', 'DeletedAt'],
    limit: 1,
    withDeleted: true,
  } as any);
  const row = (existing as any)?.[0];
  if (row) {
    if (row.DeletedAt != null) {
      const repo = User.getRepository().withDeleted();
      await repo.update({ DeletedAt: null, IsActive: true } as any, ['Id', '=', userId] as any);
    }
    return;
  }

  await User.Create(
    {
      Id: userId,
      Username: `doc_fr_${userId}`.slice(0, 64),
      PasswordHash: 'test',
      FirstName: 'Doc',
      LastName: 'Fixture',
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  createdOwnerFrUserIds.push(userId);
}

async function ensureFixtureUserRole(userId: string, roleId: string): Promise<void> {
  const existing = await UserRole.Search(
    {
      And: [
        ['UserId', '=', userId],
        ['RoleId', '=', roleId],
        ['CompanyId', 'is', null],
      ],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  if ((existing || []).length > 0) return;

  const created = await UserRole.Create(
    {
      UserId: { Id: userId } as any,
      RoleId: { Id: roleId } as any,
      CompanyId: null as any,
    } as any,
    ['Id'] as any
  );
  const id = String((created as any)?.Id || '').trim();
  if (id) createdOwnerFrUserRoleIds.push(id);
}

/**
 * Restore process env mutated by document fixtures and delete suite-owned
 * RR/FR fixtures (never deletes pre-existing reused rules).
 */
export async function restoreDocumentOwnerAuthFixtures(): Promise<void> {
  if (capturedRecordRuleEnv) {
    const root = globalThis as any;
    const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? { ...root.__CHOYSUM_RUNTIME_ENV__ } : {};
    if (previousRecordRuleEnabled === undefined) {
      delete prev.CHOYSUM_GRPC_RECORD_RULE_ENABLED;
    } else {
      prev.CHOYSUM_GRPC_RECORD_RULE_ENABLED = previousRecordRuleEnabled;
    }
    root.__CHOYSUM_RUNTIME_ENV__ = prev;
    capturedRecordRuleEnv = false;
    previousRecordRuleEnabled = undefined;
  }

  if (capturedFieldRuleEnv) {
    const root = globalThis as any;
    const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? { ...root.__CHOYSUM_RUNTIME_ENV__ } : {};
    if (previousFieldRuleEnabled === undefined) {
      delete prev.CHOYSUM_GRPC_FIELD_RULE_ENABLED;
    } else {
      prev.CHOYSUM_GRPC_FIELD_RULE_ENABLED = previousFieldRuleEnabled;
    }
    root.__CHOYSUM_RUNTIME_ENV__ = prev;
    capturedFieldRuleEnv = false;
    previousFieldRuleEnabled = undefined;
  }

  // Always allow a later ensure* to re-seed / refresh caches.
  authUserOwnerGrantsSeeded = false;
  resolvedOwnerFrRoleId = '';
  ownerFrRuleEnsured = false;

  await withPermissionGraphBypass(async () => {
    if (createdOwnerGrantRuleId) {
      await safeDeleteById(async () => RoleRecordRule.DeleteById(createdOwnerGrantRuleId), createdOwnerGrantRuleId, async id => {
        const stillThere = await RoleRecordRule.Search([['Id', '=', id]] as any, { fields: ['Id'], limit: 1 } as any);
        return (stillThere || []).length > 0;
      });
      createdOwnerGrantRuleId = '';
    }

    for (const userRoleId of createdOwnerFrUserRoleIds.splice(0)) {
      await safeDeleteById(async () => UserRole.DeleteById(userRoleId), userRoleId, async id => {
        const stillThere = await UserRole.Search([['Id', '=', id]] as any, { fields: ['Id'], limit: 1 } as any);
        return (stillThere || []).length > 0;
      });
    }

    // Keep FR fixture Role/RoleFieldRule/User for the process lifetime.
    // Soft-deleting them leaves UNIQUE indexes occupied and breaks the next suite.
    createdOwnerFrRuleId = '';
    createdOwnerFrRoleId = '';
    createdOwnerFrUserIds.length = 0;
  });

  clearRequestAuthzCaches();
}

async function safeDeleteById(
  deleteFn: () => Promise<unknown>,
  id: string,
  stillExists: (id: string) => Promise<boolean>
): Promise<void> {
  try {
    await deleteFn();
  } catch (err) {
    if (isNotFoundDeleteError(err)) return;
    if (await stillExists(id)) throw err;
  }
}
