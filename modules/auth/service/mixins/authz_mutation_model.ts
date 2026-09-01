// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition } from '@/core/service/api/query';
import { normalizeRefId, uniqStrings } from '@/core/service/utils/normalization';
import { invalidateAllAuthzCaches, invalidateAuthzCachesForUsers } from '../models/_request_cache_invalidation';

/**
 * Run a permission-graph mutation, then invalidate every request-scoped authz cache.
 *
 * Use for role / rule / inheritance mutations whose affected user set cannot be
 * precisely determined from the write payload.
 */
export async function mutateThenInvalidateAllAuthzCaches<T>(mutate: () => Promise<T>): Promise<T> {
  const out = await mutate();
  invalidateAllAuthzCaches();
  return out;
}

/**
 * Run a mutation, then invalidate request-scoped authz caches for specific users.
 *
 * Prefer for {@link UserRole} Create / CreateMany where UserId is known up front.
 */
export async function mutateThenInvalidateAuthzCachesForUsers<T>(
  userIds: Array<string | null | undefined>,
  mutate: () => Promise<T>
): Promise<T> {
  const out = await mutate();
  invalidateAuthzCachesForUsers(uniqStrings(userIds));
  return out;
}

/**
 * Collect UserId refs from a UserRole create payload (single row or many).
 */
export function userIdsFromUserRolePayloads(values: unknown): string[] {
  const rows = Array.isArray(values) ? values : values != null ? [values] : [];
  return uniqStrings(rows.map((v: any) => normalizeRefId(v?.UserId)));
}

/**
 * Base for auth models whose writes change the permission graph.
 *
 * Default Create/Update/Delete* invalidate all request-scoped authz caches after
 * the mutation. Subclasses that need domain prep call it before `super.*`;
 * UserRole overrides Create/CreateMany for targeted per-user invalidation.
 *
 * Must be the module default export so `@Model` classes can `extends` it.
 */
export default abstract class AuthzMutationModel extends BaseModel {
  /**
   * Create one row and invalidate every request-scoped authz cache.
   */
  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.Create(value as any, returnFields as any);
      return out as unknown as T;
    });
  }

  /**
   * Create many rows and invalidate every request-scoped authz cache.
   */
  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    return mutateThenInvalidateAllAuthzCaches(async () => {
      const out = await super.CreateMany(values as any, returnFields as any);
      return out as unknown as T[];
    });
  }

  /**
   * Update matching rows and invalidate every request-scoped authz cache.
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
   * Update one row by Id and invalidate every request-scoped authz cache.
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
   * Delete matching rows and invalidate every request-scoped authz cache.
   */
  static override async Delete<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    options?: any
  ): Promise<number> {
    return mutateThenInvalidateAllAuthzCaches(() => super.Delete(condition as any, options as any));
  }

  /**
   * Delete one row by Id and invalidate every request-scoped authz cache.
   */
  static override async DeleteById<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, id: string, options?: any): Promise<number> {
    return mutateThenInvalidateAllAuthzCaches(() => super.DeleteById(id as any, options as any));
  }
}
