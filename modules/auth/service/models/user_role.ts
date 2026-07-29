// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import { _lt } from '../i18n';
import User from './user';
import Role from './role';
import {
  mutateThenInvalidateAllAuthzCaches,
  mutateThenInvalidateAuthzCachesForUsers,
  userIdsFromUserRolePayloads,
} from './_authz_mutation_helpers';
import { normalizeRefId } from '@/core/service/utils/normalization';

/**
 * UserRole assigns a role to a user, optionally within one company scope.
 */
@Model('UserRole', { companyField: 'CompanyId' })
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
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Company', { scope: 'auth.model.UserRole.fields' }),
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
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    // Role assignments can change effective permissions within the same request;
    // invalidate request-scoped authz/field/record caches for the affected users only.
    return mutateThenInvalidateAuthzCachesForUsers(userIdsFromUserRolePayloads(value), async () => {
      const created = await super.Create(value as any, returnFields as any);
      return created as unknown as T;
    });
  }

  /**
   * Create multiple UserRole rows and invalidate request-scoped caches for the affected users.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    return mutateThenInvalidateAuthzCachesForUsers(userIdsFromUserRolePayloads(values), async () => {
      const created = await super.CreateMany(values as any, returnFields as any);
      return created as unknown as T[];
    });
  }

  /**
   * Update UserRole rows and invalidate request-scoped auth caches conservatively.
   */
  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>[]> {
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.Update(condition as any, values as any, returnFields as any, options as any);
      return out as unknown as Partial<T>[];
    });
  }

  /**
   * Update one UserRole row by Id and invalidate request-scoped auth caches.
   */
  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>> {
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.UpdateById(id as any, values as any, returnFields as any, options as any);
      return out as unknown as Partial<T>;
    });
  }

  /**
   * Delete matching UserRole rows and invalidate request-scoped auth caches.
   */
  static override async Delete<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    options?: any
  ): Promise<number> {
    return mutateThenInvalidateAllAuthzCaches(() => super.Delete(condition as any, options as any));
  }

  /**
   * Delete one UserRole row by Id and invalidate request-scoped auth caches.
   */
  static override async DeleteById<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, id: string, options?: any): Promise<number> {
    return mutateThenInvalidateAllAuthzCaches(() => super.DeleteById(id as any, options as any));
  }
}
