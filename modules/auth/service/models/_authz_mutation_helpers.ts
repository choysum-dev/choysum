// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeRefId, uniqStrings } from '@/core/service/utils/normalization';
import { invalidateAllAuthzCaches, invalidateAuthzCachesForUsers } from './_request_cache_invalidation';

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
