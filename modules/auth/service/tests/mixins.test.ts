// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import AuthzMutationModel, {
  mutateThenInvalidateAllAuthzCaches,
  mutateThenInvalidateAuthzCachesForUsers,
  userIdsFromUserRolePayloads,
} from '../mixins/authz_mutation_model';
import RoleInheritance from '../models/role_inheritance';
import UserRole from '../models/user_role';

/**
 * Harness consumer for AuthzMutationModel extend contract (not a persisted domain model).
 * Mirrors other apps: `@Model('X') class X extends AuthzMutationModel`.
 */
class AuthzMutationHarness extends AuthzMutationModel {}

test('AuthzMutationModel: RoleInheritance and UserRole extend the mixin', () => {
  expect(Object.prototype.isPrototypeOf.call(AuthzMutationModel, RoleInheritance)).toBe(true);
  expect(Object.prototype.isPrototypeOf.call(AuthzMutationModel, UserRole)).toBe(true);
  expect(Object.prototype.isPrototypeOf.call(AuthzMutationModel, AuthzMutationHarness)).toBe(true);
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
