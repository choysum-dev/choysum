// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import User from './user';
import Role from './role';
import { invalidateAllAuthzCaches, invalidateAuthzCachesForUsers } from './_authz_mutation_helpers';

/**
 * UserRole assigns a role to a user, optionally within one company scope.
 */
@Model('UserRole', { companyScoped: true })
export default class UserRole extends BaseModel {
  /**
   * User that receives the role assignment.
   */
  @Field({ type: 'ManyToOne', relation: { targetModel: () => User } })
  UserId: User;

  /**
   * Role granted to the user.
   */
  @Field({ type: 'ManyToOne', relation: { targetModel: () => Role } })
  RoleId: Role;

  /**
   * Company scope for the assignment when the grant is company-specific.
   */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Company', column: { notNull: false, size: 20, index: true } })
  CompanyId?: string;

  /**
   * Read a trimmed Id from a relation reference or scalar value.
   */
  private static _maybeId(v: any): string {
    if (!v) return '';
    if (typeof v === 'string') return v.trim();
    const id = (v as any).Id;
    return typeof id === 'string' ? id.trim() : '';
  }

  /**
   * Create one UserRole row and invalidate request-scoped caches for the affected users.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    const created = await super.Create(value as any, returnFields as any);

    // Role assignments can change effective permissions within the same request;
    // invalidate request-scoped authz/field/record caches for correctness.
    const rows: any[] = Array.isArray(value as any) ? (value as any) : [value];
    const userIds = rows.map((v: any) => (this as any)._maybeId(v?.UserId)).filter(Boolean);
    invalidateAuthzCachesForUsers(userIds);

    return created as unknown as T;
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
    const out = await super.Update(condition as any, values as any, returnFields as any, options as any);
    // Conservative: any mutation to the permission graph can make request-scoped authz decisions stale.
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>[];
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
    const out = await super.UpdateById(id as any, values as any, returnFields as any, options as any);
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>;
  }

  /**
   * Delete matching UserRole rows and invalidate request-scoped auth caches.
   */
  static override async Delete<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    options?: any
  ): Promise<number> {
    const out = await super.Delete(condition as any, options as any);
    invalidateAllAuthzCaches();
    return out;
  }

  /**
   * Delete one UserRole row by Id and invalidate request-scoped auth caches.
   */
  static override async DeleteById<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, id: string, options?: any): Promise<number> {
    const out = await super.DeleteById(id as any, options as any);
    invalidateAllAuthzCaches();
    return out;
  }
}
