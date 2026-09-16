// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { invalidateAllAuthzCaches, invalidateAuthzCachesForUsers } from '../models/_request_cache_invalidation';
import { uniqStrings } from '@/core/service/utils/normalization';

export { userIdsFromUserRolePayloads } from '@/core/service/orm/model/model_write_policy';

/**
 * Run a permission-graph mutation, then invalidate every request-scoped authz cache.
 *
 * Prefer `@Model({ afterMutation: { invalidateAuthz: 'all' } })` for persisted models;
 * keep this helper for non-model call sites and unit tests.
 */
export async function mutateThenInvalidateAllAuthzCaches<T>(mutate: () => Promise<T>): Promise<T> {
  const out = await mutate();
  invalidateAllAuthzCaches();
  return out;
}

/**
 * Run a mutation, then invalidate request-scoped authz caches for specific users.
 *
 * Prefer `@Model({ afterMutation: { invalidateAuthz: 'usersFromPayload' } })` for UserRole;
 * keep this helper for non-model call sites and unit tests.
 */
export async function mutateThenInvalidateAuthzCachesForUsers<T>(
  userIds: Array<string | null | undefined>,
  mutate: () => Promise<T>
): Promise<T> {
  const out = await mutate();
  invalidateAuthzCachesForUsers(uniqStrings(userIds));
  return out;
}
