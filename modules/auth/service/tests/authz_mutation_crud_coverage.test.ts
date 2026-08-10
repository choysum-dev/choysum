// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import User from '@/auth/service/models/user';
import Role from '@/auth/service/models/role';
import UserRole from '@/auth/service/models/user_role';
import RoleInheritance from '@/auth/service/models/role_inheritance';
import RoleMethodAccess from '@/auth/service/models/role_method_access';
import RoleRecordRule from '@/auth/service/models/role_record_rule';
import RoleFieldRule from '@/auth/service/models/role_field_rule';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaServiceModel from '@/meta/service/models/service';
import type MetaFieldModel from '@/meta/service/models/field';
import { metaModelId } from './_meta_ids';
import { ensureRequestContext, resetRequestContext, uid } from '@/auth/service/tests/_request_context_fixtures';

const MetaService = createServiceByModel<typeof MetaServiceModel>('meta.MetaService');
const MetaField = createServiceByModel<typeof MetaFieldModel>('meta.MetaField');

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

function setupAllowlistForFixtures(): void {
  setReq({
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
      'auth.RoleInheritance:read',
      'auth.RoleInheritance:write',
      'auth.RoleInheritance:create',
      'auth.RoleInheritance:delete',
      'RoleInheritance:read',
      'RoleInheritance:write',
      'RoleInheritance:create',
      'RoleInheritance:delete',
      'auth.RoleMethodAccess:read',
      'auth.RoleMethodAccess:write',
      'auth.RoleMethodAccess:create',
      'auth.RoleMethodAccess:delete',
      'RoleMethodAccess:read',
      'RoleMethodAccess:write',
      'RoleMethodAccess:create',
      'RoleMethodAccess:delete',
      'auth.RoleRecordRule:read',
      'auth.RoleRecordRule:write',
      'auth.RoleRecordRule:create',
      'auth.RoleRecordRule:delete',
      'RoleRecordRule:read',
      'RoleRecordRule:write',
      'RoleRecordRule:create',
      'RoleRecordRule:delete',
      'auth.RoleFieldRule:read',
      'auth.RoleFieldRule:write',
      'auth.RoleFieldRule:create',
      'auth.RoleFieldRule:delete',
      'RoleFieldRule:read',
      'RoleFieldRule:write',
      'RoleFieldRule:create',
      'RoleFieldRule:delete',
      'meta.MetaModel:read',
      'meta.MetaService:read',
      'meta.MetaField:read',
      'MetaModel:read',
      'MetaService:read',
      'MetaField:read',
    ],
  });
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

async function createRole(codePrefix: string): Promise<{ id: string }> {
  const created = await Role.Create(
    {
      Name: uid('role'),
      Code: uid(codePrefix),
      Description: 'test',
      IsActive: true,
      IsSystem: false,
    } as any,
    ['Id'] as any
  );
  return { id: created.Id };
}

async function resolveModelId(app: string, name: string): Promise<string> {
  const id = await metaModelId(app, name);
  if (!id) throw new Error(`meta model not found: ${app}.${name}`);
  return id;
}

async function resolveService(modelId: string, serviceName: string): Promise<{ id: string }> {
  const rows = await MetaService.Search({ And: [['ModelId', '=', modelId]] } as any, { fields: ['Id', 'Name'], limit: 5000 } as any);
  const target = String(serviceName || '')
    .trim()
    .toLowerCase();
  const hit = (rows || []).find(
    (r: any) =>
      String((r as any).Name || '')
        .trim()
        .toLowerCase() === target
  ) as any;
  const id = String(hit?.Id || '').trim();
  if (!id) throw new Error(`meta service not found: ${modelId}.${serviceName}`);
  return { id };
}

async function resolveFieldId(modelId: string, fieldName: string): Promise<string> {
  const hit = (
    await MetaField.Search(
      {
        And: [
          ['ModelId', '=', modelId],
          ['Name', '=', fieldName],
        ],
      } as any,
      { fields: ['Id'], limit: 1 } as any
    )
  )?.[0] as any;
  const id = String(hit?.Id || '').trim();
  if (!id) throw new Error(`meta field not found: ${modelId}.${fieldName}`);
  return id;
}

test('authz mutation CRUD coverage: RoleInheritance CreateMany/Update/Delete paths', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const parent = await createRole('INH_P');
    const child = await createRole('INH_C');
    const child2 = await createRole('INH_C2');

    const many = await RoleInheritance.CreateMany(
      [
        {
          ParentRoleId: { Id: parent.id } as any,
          ChildRoleId: { Id: child.id } as any,
        } as any,
      ],
      ['Id'] as any
    );
    expect(many.length).toBe(1);
    const id = String((many[0] as any)?.Id || '').trim();
    expect(id.length > 0).toBe(true);

    await RoleInheritance.UpdateById(id, { ChildRoleId: { Id: child2.id } as any } as any, ['Id'] as any);
    await RoleInheritance.Update(['Id', '=', id] as any, { ChildRoleId: { Id: child.id } as any } as any, ['Id'] as any);

    const deletedById = await RoleInheritance.DeleteById(id);
    expect(deletedById).toBe(1);

    const again = await RoleInheritance.Create(
      {
        ParentRoleId: { Id: parent.id } as any,
        ChildRoleId: { Id: child.id } as any,
      } as any,
      ['Id'] as any
    );
    const againId = String((again as any)?.Id || '').trim();
    const deleted = await RoleInheritance.Delete(['Id', '=', againId] as any);
    expect(deleted).toBe(1);
  });
});

test('authz mutation CRUD coverage: UserRole Update/Delete paths', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  const c1 = { Id: uid('C1') };
  await withModelContext({ activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any, async () => {
    const userId = await createUser(c1.Id);
    setIdentity(userId);
    const role = await createRole('UR_COV');
    const role2 = await createRole('UR_COV2');

    const created = await UserRole.Create(
      {
        UserId: { Id: userId } as any,
        RoleId: { Id: role.id } as any,
        CompanyId: c1.Id,
      } as any,
      ['Id'] as any
    );
    const id = String((created as any)?.Id || '').trim();
    expect(id.length > 0).toBe(true);

    await UserRole.UpdateById(id, { RoleId: { Id: role2.id } as any } as any, ['Id'] as any);
    await UserRole.Update(['Id', '=', id] as any, { RoleId: { Id: role.id } as any } as any, ['Id'] as any);

    const deletedById = await UserRole.DeleteById(id);
    expect(deletedById).toBe(1);

    const again = await UserRole.Create(
      {
        UserId: { Id: userId } as any,
        RoleId: { Id: role.id } as any,
        CompanyId: c1.Id,
      } as any,
      ['Id'] as any
    );
    const againId = String((again as any)?.Id || '').trim();
    const deleted = await UserRole.Delete(['Id', '=', againId] as any);
    expect(deleted).toBe(1);
  });
});

test('authz mutation CRUD coverage: rule models condition Delete paths', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  const c1 = { Id: uid('C1') };
  await withModelContext({ activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any, async () => {
    const role = await createRole('RULE_DEL');
    const userModelId = await resolveModelId('auth', 'User');
    const browse = await resolveService(userModelId, 'browse');
    const fieldId = await resolveFieldId(userModelId, 'Username');

    const method = await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaServiceId: browse.id,
        MetaModelId: null,
        MetaApplicationId: null,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );
    const methodId = String((method as any)?.Id || '').trim();
    expect(await RoleMethodAccess.Delete(['Id', '=', methodId] as any)).toBe(1);

    const record = await RoleRecordRule.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaModelId: userModelId,
        MetaApplicationId: null,
        Kind: 'grant',
        PermRead: true,
      } as any,
      ['Id'] as any
    );
    const recordId = String((record as any)?.Id || '').trim();
    expect(await RoleRecordRule.Delete(['Id', '=', recordId] as any)).toBe(1);

    const field = await RoleFieldRule.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaModelId: userModelId,
        MetaFieldId: fieldId,
        MetaApplicationId: null,
        PermRead: 'allow',
        PermWrite: 'deny',
      } as any,
      ['Id'] as any
    );
    const fieldRuleId = String((field as any)?.Id || '').trim();
    expect(await RoleFieldRule.Delete(['Id', '=', fieldRuleId] as any)).toBe(1);
  });
});
