// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { _lt } from '../i18n';
import Role from './role';

/**
 * RoleInheritance links a parent role to a child role so effective permissions
 * can be expanded through inheritance.
 */
@Model('RoleInheritance', { afterMutation: { invalidateAuthz: 'all' } })
export default class RoleInheritance extends BaseModel {
  /**
   * Parent role that contributes permissions to the child role.
   */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Role },
    condition: ['IsActive', '=', true],
    string: _lt('Parent Role', { scope: 'auth.model.RoleInheritance.fields' }),
  })
  ParentRoleId: Role;

  /**
   * Child role that inherits permissions from ParentRoleId.
   */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Role },
    condition: ['IsActive', '=', true],
    string: _lt('Child Role', { scope: 'auth.model.RoleInheritance.fields' }),
  })
  ChildRoleId: Role;
}
