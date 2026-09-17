// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, type ModelCtor, type RowOf } from '@/core/service';
import type { Insertable, Updateable } from '@/core/service/api/input';
import type { FieldSelection, Projected, RowOrProjected } from '@/core/service/api/selection';
import type { DeleteOptions, QueryCondition, UpdateOptions } from '@/core/service/api/query';
import { normalizeRefId, uniqStrings } from '@/core/service/utils/normalization';
import { invalidateAllAuthzCaches, invalidateAuthzCachesForUsers } from '../models/_request_cache_invalidation';

/** Write op that just completed; passed to {@link AuthzMutationModel.invalidateAuthzCachesAfterWrite}. */
export type AuthzMutationOp = 'create' | 'createMany' | 'update' | 'updateById' | 'delete' | 'deleteById';

type AuthzInvalidateHost = {
  invalidateAuthzCachesAfterWrite(op: AuthzMutationOp, payload?: unknown): void;
};

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
 * Prefer when UserId is known up front (also what {@link UserRole} uses via
 * {@link AuthzMutationModel.invalidateAuthzCachesAfterWrite}).
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
 * Default Create/Update/Delete* call {@link invalidateAuthzCachesAfterWrite} after
 * a successful mutation (clears all request-scoped authz caches). Subclasses that
 * need domain prep call it before `super.*`; subclasses that need a different
 * invalidate policy override {@link invalidateAuthzCachesAfterWrite} (e.g. UserRole).
 *
 * Must be the module default export so `@Model` classes can `extends` it.
 */
export default abstract class AuthzMutationModel extends BaseModel {
  /**
   * Invalidate request-scoped authz caches after a successful write.
   *
   * Default clears every authz cache for the request. Override for targeted
   * invalidation; IMD subclasses of that override should usually call `super`
   * unless they replace the policy entirely.
   */
  static invalidateAuthzCachesAfterWrite(_op: AuthzMutationOp, _payload?: unknown): void {
    invalidateAllAuthzCaches();
  }

  /**
   * Create one row then run {@link invalidateAuthzCachesAfterWrite}.
   */
  static override async Create<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    value: Partial<Insertable<RowOf<C>>>,
    returnFields?: F
  ): Promise<RowOrProjected<RowOf<C>, F>> {
    const out = await super.Create<C, F>(value, returnFields);
    (this as unknown as AuthzInvalidateHost).invalidateAuthzCachesAfterWrite('create', value);
    return out;
  }

  /**
   * Create many rows then run {@link invalidateAuthzCachesAfterWrite}.
   */
  static override async CreateMany<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    values: Partial<Insertable<RowOf<C>>>[],
    returnFields?: F
  ): Promise<Array<RowOrProjected<RowOf<C>, F>>> {
    const out = await super.CreateMany<C, F>(values, returnFields);
    (this as unknown as AuthzInvalidateHost).invalidateAuthzCachesAfterWrite('createMany', values);
    return out;
  }

  /**
   * Update matching rows then run {@link invalidateAuthzCachesAfterWrite}.
   */
  static override async Update<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    condition: QueryCondition<RowOf<C>>,
    values: Partial<Updateable<RowOf<C>>>,
    returnFields?: F,
    options?: UpdateOptions
  ): Promise<Array<F extends FieldSelection<RowOf<C>> ? Projected<RowOf<C>, F> : Partial<RowOf<C>>>> {
    const out = await super.Update<C, F>(condition, values, returnFields, options);
    (this as unknown as AuthzInvalidateHost).invalidateAuthzCachesAfterWrite('update', { condition, values });
    return out;
  }

  /**
   * Update one row by Id then run {@link invalidateAuthzCachesAfterWrite}.
   */
  static override async UpdateById<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    id: string,
    values: Partial<Updateable<RowOf<C>>>,
    returnFields?: F,
    options?: UpdateOptions
  ): Promise<F extends FieldSelection<RowOf<C>> ? Projected<RowOf<C>, F> : Partial<RowOf<C>>> {
    const out = await super.UpdateById<C, F>(id, values, returnFields, options);
    (this as unknown as AuthzInvalidateHost).invalidateAuthzCachesAfterWrite('updateById', { id, values });
    return out;
  }

  /**
   * Delete matching rows then run {@link invalidateAuthzCachesAfterWrite}.
   */
  static override async Delete<C extends ModelCtor>(
    this: C,
    condition: QueryCondition<RowOf<C>>,
    options?: DeleteOptions
  ): Promise<number> {
    const out = await super.Delete<C>(condition, options);
    (this as unknown as AuthzInvalidateHost).invalidateAuthzCachesAfterWrite('delete', condition);
    return out;
  }

  /**
   * Delete one row by Id then run {@link invalidateAuthzCachesAfterWrite}.
   */
  static override async DeleteById<C extends ModelCtor>(this: C, id: string, options?: DeleteOptions): Promise<number> {
    const out = await super.DeleteById<C>(id, options);
    (this as unknown as AuthzInvalidateHost).invalidateAuthzCachesAfterWrite('deleteById', id);
    return out;
  }
}
