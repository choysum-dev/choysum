// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import AttachmentOwnerMixin from '@/core/service/mixins/attachment_owner_model';
import AuthzMutationModel, {
  mutateThenInvalidateAllAuthzCaches,
  mutateThenInvalidateAuthzCachesForUsers,
  userIdsFromUserRolePayloads,
} from '../mixins/authz_mutation_model';
import RoleInheritance from '../models/role_inheritance';
import Role from '../models/role';
import User from '../models/user/user';
import UserRole from '../models/user_role';
import { getServiceFactory, registerServiceFactory, unregisterServiceFactory } from '@/core/service/rpc';

/**
 * Harness consumer for AuthzMutationModel extend contract (not a persisted domain model).
 * Mirrors other apps: `@Model('X') class X extends AuthzMutationModel`.
 */
class AuthzMutationHarness extends AuthzMutationModel {}

function withServiceFactory<T>(modelName: string, factory: () => unknown, fn: () => Promise<T> | T): Promise<T> | T {
  const previous = getServiceFactory(modelName);
  registerServiceFactory(modelName, factory as any);
  const restore = () => {
    unregisterServiceFactory(modelName);
    if (previous) registerServiceFactory(modelName, previous);
  };
  try {
    const out = fn();
    if (out && typeof (out as Promise<T>).then === 'function') {
      return (out as Promise<T>).finally(restore);
    }
    restore();
    return out;
  } catch (err) {
    restore();
    throw err;
  }
}

test('AuthzMutationModel: RoleInheritance, UserRole, and Role extend the mixin', () => {
  expect(Object.prototype.isPrototypeOf.call(AuthzMutationModel, RoleInheritance)).toBe(true);
  expect(Object.prototype.isPrototypeOf.call(AuthzMutationModel, UserRole)).toBe(true);
  expect(Object.prototype.isPrototypeOf.call(AuthzMutationModel, Role)).toBe(true);
  expect(Object.prototype.isPrototypeOf.call(AuthzMutationModel, AuthzMutationHarness)).toBe(true);
  expect(Role.prototype instanceof AuthzMutationModel).toBe(true);
});

test('AuthzMutationModel helpers: userIdsFromUserRolePayloads extracts UserId refs', () => {
  expect(userIdsFromUserRolePayloads({ UserId: 'u1' })).toEqual(['u1']);
  expect(userIdsFromUserRolePayloads([{ UserId: { Id: 'u2' } }, { UserId: 'u2' }, { UserId: 'u3' }])).toEqual(['u2', 'u3']);
  expect(userIdsFromUserRolePayloads(null)).toEqual([]);
});

test('AuthzMutationModel helpers: mutateThenInvalidate wrappers return mutate result', async () => {
  const all = await mutateThenInvalidateAllAuthzCaches(async () => 42);
  expect(all).toBe(42);
  const users = await mutateThenInvalidateAuthzCachesForUsers(['u1'], async () => 'ok');
  expect(users).toBe('ok');
});

test('AuthzMutationModel: harness Create is defined on the mixin prototype chain', () => {
  expect(typeof AuthzMutationHarness.Create).toBe('function');
  expect(typeof AuthzMutationHarness.DeleteById).toBe('function');
  expect(typeof BaseModel.Create).toBe('function');
});

test('User: extends AttachmentOwnerMixin and exposes bind/unbind entry points', () => {
  expect(User.prototype instanceof AttachmentOwnerMixin).toBe(true);
  expect(typeof User.AttachmentBind).toBe('function');
  expect(typeof User.AttachmentUnbind).toBe('function');
});

test('User: AttachmentBind / AttachmentUnbind dial document.AttachmentBinding', async () => {
  let bindReq: unknown;
  let unbindReq: unknown;
  await withServiceFactory(
    'document.AttachmentBinding',
    () => ({
      Bind: async (req: any) => {
        bindReq = req;
        return { attachmentBindingId: 'b1', status: 'active' };
      },
      Unbind: async (req: any) => {
        unbindReq = req;
        return { attachmentBindingId: req.attachmentBindingId, status: 'unbound' };
      },
    }),
    async () => {
      const bound = await User.AttachmentBind({
        attachmentObjectId: 'c1',
        ownerModel: 'auth.User',
        ownerRecordId: 'u1',
        fieldName: 'Avatar',
        mutationId: 'mut1',
      });
      expect(bound).toEqual({ attachmentBindingId: 'b1', status: 'active' });
      expect(bindReq).toMatchObject({ fieldName: 'Avatar', ownerModel: 'auth.User' });

      const unbound = await User.AttachmentUnbind({
        attachmentBindingId: 'b1',
        mutationId: 'mut2',
        reason: 'other',
      });
      expect(unbound.status).toBe('unbound');
      expect(unbindReq).toMatchObject({ attachmentBindingId: 'b1', reason: 'other' });
    }
  );
});
