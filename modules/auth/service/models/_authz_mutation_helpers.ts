// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { invalidateAuthzRequestCaches } from './_request_cache_invalidation';

/**
 * Invalidate every request-scoped authz cache entry for all users.
 *
 * Call this after any permission-graph mutation whose affected user set
 * cannot be precisely determined (e.g. role definition changes, inheritance
 * edge changes, rule-scope updates).
 */
export function invalidateAllAuthzCaches(): void {
  invalidateAuthzRequestCaches({ allUsers: true });
}

/**
 * Invalidate request-scoped authz cache entries for specific users.
 *
 * Prefer this over {@link invalidateAllAuthzCaches} when the mutation target
 * is a known, narrow set of users (e.g. a role assignment).
 */
export function invalidateAuthzCachesForUsers(userIds: string[]): void {
  invalidateAuthzRequestCaches({ userIds });
}
