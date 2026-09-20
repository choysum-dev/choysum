// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model, Field } from '@/core/service';
import { _lt } from '../i18n';
import User from './user/user';
import Role from './role';
import AuthzMutationModel, {
  userIdsFromUserRolePayloads,
  type AuthzMutationOp,
} from '../mixins/authz_mutation_model';
import { invalidateAuthzCachesForUsers } from './_request_cache_invalidation';
import type Company from '@/base/service/models/company';

/**
 * UserRole assigns a role to a user, optionally within one company scope.
 */
@Model('UserRole', { companyField: 'CompanyId' })
export default class UserRole extends AuthzMutationModel {
  /**
   * User that receives the role assignment.
   */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => User },
    condition: ['IsActive', '=', true],
    string: _lt('User', { scope: 'auth.model.UserRole.fields' }),
  })
  UserId: User;

  /**
   * Role granted to the user.
   */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Role },
    condition: ['IsActive', '=', true],
    string: _lt('Role', { scope: 'auth.model.UserRole.fields' }),
  })
  RoleId: Role;

  /**
   * Company scope for the assignment when the grant is company-specific.
   */
  @Field<Company>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Company', { scope: 'auth.model.UserRole.fields' }),
    help: _lt('Leave empty to grant the role in every company the user can access.', {
      scope: 'auth.model.UserRole.fields',
    }),
  })
  CompanyId?: string;

  /**
   * Create/CreateMany: clear authz caches only for UserIds in the write payload.
   * If no UserId can be resolved, fall back to invalidating all authz caches.
   * Other ops keep the mixin default (invalidate all).
   */
  static override invalidateAuthzCachesAfterWrite(op: AuthzMutationOp, payload?: unknown): void {
    if (op === 'create' || op === 'createMany') {
      const userIds = userIdsFromUserRolePayloads(payload);
      if (userIds.length > 0) {
        invalidateAuthzCachesForUsers(userIds);
        return;
      }
    }
    super.invalidateAuthzCachesAfterWrite(op, payload);
  }
}