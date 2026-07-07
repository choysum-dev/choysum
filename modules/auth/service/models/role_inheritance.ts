// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import Role from './role';
import { invalidateAllAuthzCaches } from './_authz_mutation_helpers';

/**
 * RoleInheritance links a parent role to a child role so effective permissions
 * can be expanded through inheritance.
 */
@Model('RoleInheritance')
export default class RoleInheritance extends BaseModel {
  /**
   * Parent role that contributes permissions to the child role.
   */
  @Field({ type: 'ManyToOne', relation: { targetModel: () => Role } })
  ParentRoleId: Role;

  /**
   * Child role that inherits permissions from ParentRoleId.
   */
  @Field({ type: 'ManyToOne', relation: { targetModel: () => Role } })
  ChildRoleId: Role;

  /**
   * Create one RoleInheritance row and invalidate request-scoped auth caches.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    const out = await super.Create(value as any, returnFields as any);
    invalidateAllAuthzCaches();
    return out as unknown as T;
  }

  /**
   * Create multiple RoleInheritance rows and invalidate request-scoped auth caches.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const out = await super.CreateMany(values as any, returnFields as any);
    invalidateAllAuthzCaches();
    return out as unknown as T[];
  }

  /**
   * Update RoleInheritance rows and invalidate request-scoped auth caches.
   */
  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: any
  ): Promise<Partial<T>[]> {
    const out = await super.Update(condition as any, values as any, returnFields as any, options as any);
    invalidateAllAuthzCaches();
    return out as unknown as Partial<T>[];
  }

  /**
   * Update one RoleInheritance row by Id and invalidate request-scoped auth caches.
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
   * Delete matching RoleInheritance rows and invalidate request-scoped auth caches.
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
   * Delete one RoleInheritance row by Id and invalidate request-scoped auth caches.
   */
  static override async DeleteById<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, id: string, options?: any): Promise<number> {
    const out = await super.DeleteById(id as any, options as any);
    invalidateAllAuthzCaches();
    return out;
  }
}
