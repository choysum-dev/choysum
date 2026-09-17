// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model, Field, type ModelCtor, type RowOf } from '@/core/service';
import type { Insertable } from '@/core/service/api/input';
import type { FieldSelection, RowOrProjected } from '@/core/service/api/selection';
import { _lt } from '../i18n';
import User from './user/user';
import Role from './role';
import AuthzMutationModel, {
  userIdsFromUserRolePayloads,
} from '../mixins/authz_mutation_model';
import { invalidateAuthzCachesForUsers } from './_request_cache_invalidation';
import { normalizeRefId } from '@/core/service/utils/normalization';
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
   * Read a trimmed Id from a relation reference or scalar value.
   */
  static _maybeId(v: any): string {
    return normalizeRefId(v) ?? '';
  }

  /**
   * Create one UserRole row and invalidate request-scoped caches for the affected users.
   */
  static override async Create<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    value: Partial<Insertable<RowOf<C>>>,
    returnFields?: F
  ): Promise<RowOrProjected<RowOf<C>, F>> {
    // Role assignments can change effective permissions within the same request;
    // invalidate request-scoped authz/field/record caches for the affected users only.
    const out = await super.Create(value, returnFields);
    invalidateAuthzCachesForUsers(userIdsFromUserRolePayloads(value));
    return out;
  }

  /**
   * Create multiple UserRole rows and invalidate request-scoped caches for the affected users.
   */
  static override async CreateMany<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    values: Partial<Insertable<RowOf<C>>>[],
    returnFields?: F
  ): Promise<Array<RowOrProjected<RowOf<C>, F>>> {
    const out = await super.CreateMany(values, returnFields);
    invalidateAuthzCachesForUsers(userIdsFromUserRolePayloads(values));
    return out;
  }
}
