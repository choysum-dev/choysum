// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {  BaseModel, type ModelCtor, type RowOf } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection, Projected, RowOrProjected } from '@/core/service/api/selection';
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
  static override async Create<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    value: Partial<Insertable<RowOf<C>>>,
    returnFields?: F
  ): Promise<RowOrProjected<RowOf<C>, F>> {
    const out = await super.Create(value, returnFields);
    invalidateAllAuthzCaches();
    return out;
  }

  /**
   * Create many rows and invalidate every request-scoped authz cache.
   */
  static override async CreateMany<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    values: Partial<Insertable<RowOf<C>>>[],
    returnFields?: F
  ): Promise<Array<RowOrProjected<RowOf<C>, F>>> {
    const out = await super.CreateMany(values, returnFields);
    invalidateAllAuthzCaches();
    return out;
  }

  /**
   * Update matching rows and invalidate every request-scoped authz cache.
   */
  static override async Update<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    condition: QueryCondition<RowOf<C>>,
    values: Partial<Updateable<RowOf<C>>>,
    returnFields?: F,
    options?: UpdateOptions
  ): Promise<Array<F extends FieldSelection<RowOf<C>> ? Projected<RowOf<C>, F> : Partial<RowOf<C>>>> {
    const out = await super.Update(condition, values, returnFields, options);
    invalidateAllAuthzCaches();
    return out;
  }

  /**
   * Update one row by Id and invalidate every request-scoped authz cache.
   */
  static override async UpdateById<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    id: string,
    values: Partial<Updateable<RowOf<C>>>,
    returnFields?: F,
    options?: UpdateOptions
  ): Promise<F extends FieldSelection<RowOf<C>> ? Projected<RowOf<C>, F> : Partial<RowOf<C>>> {
    const out = await super.UpdateById(id, values, returnFields, options);
    invalidateAllAuthzCaches();
    return out;
  }

  /**
   * Delete matching rows and invalidate every request-scoped authz cache.
   */
  static override async Delete<C extends ModelCtor>(
    this: C,
    condition: QueryCondition<RowOf<C>>,
    options?: DeleteOptions
  ): Promise<number> {
    const out = await super.Delete(condition, options);
    invalidateAllAuthzCaches();
    return out;
  }

  /**
   * Delete one row by Id and invalidate every request-scoped authz cache.
   */
  static override async DeleteById<C extends ModelCtor>(this: C, id: string, options?: DeleteOptions): Promise<number> {
    const out = await super.DeleteById(id, options);
    invalidateAllAuthzCaches();
    return out;
  }
}
