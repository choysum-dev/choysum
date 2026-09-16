// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { _lt } from '../i18n';
import User from './user/user';
import Role from './role';
import { normalizeRefId } from '@/core/service/utils/normalization';
import type Company from '@/base/service/models/company';

/**
 * UserRole assigns a role to a user, optionally within one company scope.
 */
@Model('UserRole', {
  companyField: 'CompanyId',
  afterMutation: { invalidateAuthz: 'usersFromPayload' },
})
export default class UserRole extends BaseModel {
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
}
