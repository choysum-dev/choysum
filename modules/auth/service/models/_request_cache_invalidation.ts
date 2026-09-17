// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  invalidateAuthzCachesForUsers as invalidateAuthzCachesForUsersCore,
} from '@/core/service/api/authz_request_cache';

export {
  buildAuthzContextCacheKey,
  buildMethodAccessCacheKey,
  buildUiGrantCacheKey,
  invalidateAllAuthzCaches,
  invalidateAuthzRequestCaches,
} from '@/core/service/api/authz_request_cache';

type InvalidateForUsers = (userIds: string[]) => void;

let invalidateAuthzCachesForUsersImpl: InvalidateForUsers = invalidateAuthzCachesForUsersCore;

/** Invalidate request-scoped authz cache entries for specific users. */
export function invalidateAuthzCachesForUsers(userIds: string[]): void {
  invalidateAuthzCachesForUsersImpl(userIds);
}

/** Test-only: replace or restore the per-user invalidate implementation. */
export function __setInvalidateAuthzCachesForUsersForTest(fn: InvalidateForUsers | null): void {
  invalidateAuthzCachesForUsersImpl = fn ?? invalidateAuthzCachesForUsersCore;
}
