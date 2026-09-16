// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, type ModelCtor } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { DeleteOptions, QueryCondition, UpdateOptions } from '@/core/service/api/query';
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

type UserRoleUserIdPayload = {
  UserId?: string | { Id: string } | null;
};

/**
 * Collect UserId refs from a UserRole create payload (single row or many).
 */
export function userIdsFromUserRolePayloads(
  values: Partial<UserRoleUserIdPayload> | Array<Partial<UserRoleUserIdPayload>> | null | undefined
): string[] {
  const rows = Array.isArray(values) ? values : values != null ? [values] : [];
  return uniqStrings(rows.map(v => normalizeRefId(v?.UserId)));
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
    this: ModelCtor<T>,
    value: Partial<Insertable<T>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    return mutateThenInvalidateAllAuthzCaches(() => super.Create<T>(value, returnFields));
  }

  /**
   * Create many rows and invalidate every request-scoped authz cache.
   */
  static override async CreateMany<T extends BaseModel>(
    this: ModelCtor<T>,
    values: Partial<Insertable<T>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    return mutateThenInvalidateAllAuthzCaches(() => super.CreateMany<T>(values, returnFields));
  }

  /**
   * Update matching rows and invalidate every request-scoped authz cache.
   */
  static override async Update<T extends BaseModel>(
    this: ModelCtor<T>,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>[]> {
    return mutateThenInvalidateAllAuthzCaches(() => super.Update<T>(condition, values, returnFields, options));
  }

  /**
   * Update one row by Id and invalidate every request-scoped authz cache.
   */
  static override async UpdateById<T extends BaseModel>(
    this: ModelCtor<T>,
    id: string,
    values: Partial<Updateable<T>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>> {
    return mutateThenInvalidateAllAuthzCaches(() => super.UpdateById<T>(id, values, returnFields, options));
  }

  /**
   * Delete matching rows and invalidate every request-scoped authz cache.
   */
  static override async Delete<T extends BaseModel>(
    this: ModelCtor<T>,
    condition: QueryCondition<T>,
    options?: DeleteOptions
  ): Promise<number> {
    return mutateThenInvalidateAllAuthzCaches(() => super.Delete<T>(condition, options));
  }

  /**
   * Delete one row by Id and invalidate every request-scoped authz cache.
   */
  static override async DeleteById<T extends BaseModel>(this: ModelCtor<T>, id: string, options?: DeleteOptions): Promise<number> {
    return mutateThenInvalidateAllAuthzCaches(() => super.DeleteById<T>(id, options));
  }
}
