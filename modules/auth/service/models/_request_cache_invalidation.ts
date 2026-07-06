// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { invalidateRequestCaches } from '../utils/auth_cache_manager';

type InvalidateOpts = {
  userIds?: string[];
  allUsers?: boolean;
};

/**
 * Invalidate request-scoped auth caches after permission graph changes.
 *
 * Delegates to AuthCacheManager which provides structured cache invalidation
 * instead of directly manipulating req.__choysumServiceState internals.
 *
 * Clears:
 * - memoized authz context entries stored in req.__choysumServiceState (User._getAuthzContext)
 * - request-level record/field rule caches on jsCtx (Symbol.for('choysum.recordrule.cache') / Symbol.for('choysum.fieldrule.cache'))
 */
export function invalidateAuthzRequestCaches(opts: InvalidateOpts = {}): void {
  invalidateRequestCaches(opts);
}
