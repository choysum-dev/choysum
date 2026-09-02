// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { uniqStrings } from '@/core/service/utils/normalization';
import { getJsCtxAndReq, deleteReqStateKeysByPrefix, invalidateJsCtxSymbolCache } from './context';

const AUTHZ_CTX_PREFIX = 'authzContext::';
const METHOD_ACCESS_PREFIX = 'methodAccess::';
const UI_GRANT_PREFIX = 'uiGrantExpansion::';

type CacheGroup = {
  prefix: string;
  userScoped: boolean;
};

const AUTHZ_CACHE_GROUPS: CacheGroup[] = [
  { prefix: AUTHZ_CTX_PREFIX, userScoped: true },
  { prefix: METHOD_ACCESS_PREFIX, userScoped: true },
  { prefix: UI_GRANT_PREFIX, userScoped: false },
];

type InvalidateOpts = {
  userIds?: string[];
  allUsers?: boolean;
};

/** Build request-scoped authz context cache key. */
export function buildAuthzContextCacheKey(userId: string, companyScopeKey: string): string {
  return `${AUTHZ_CTX_PREFIX}${String(userId || '').trim()}::${String(companyScopeKey || '').trim()}`;
}

/** Build request-scoped method-access cache key. */
export function buildMethodAccessCacheKey(userId: string, companyId: string, fullMethod: string): string {
  return `${METHOD_ACCESS_PREFIX}${String(userId || '').trim()}::${String(companyId || '').trim()}::${String(fullMethod || '').trim()}`;
}

/** Build request-scoped UI-grant expansion cache key. */
export function buildUiGrantCacheKey(roleSignature: string): string {
  return `${UI_GRANT_PREFIX}${String(roleSignature || '').trim()}`;
}

/**
 * Invalidate request-scoped auth caches after permission graph changes.
 *
 * Clears memoized authz context entries on req.__choysumServiceState and
 * request-level record/field rule caches on jsCtx.
 */
export function invalidateAuthzRequestCaches(opts: InvalidateOpts = {}): void {
  const { jsCtx, req } = getJsCtxAndReq();
  const state: any = req?.__choysumServiceState;

  const ids = uniqStrings(opts.userIds || []);
  const invalidateAll = Boolean(opts.allUsers) || ids.length === 0;

  if (state && typeof state === 'object') {
    if (invalidateAll) {
      for (const group of AUTHZ_CACHE_GROUPS) {
        deleteReqStateKeysByPrefix(state as Record<string, unknown>, group.prefix);
      }
    } else {
      for (const uid of ids) {
        for (const group of AUTHZ_CACHE_GROUPS) {
          if (!group.userScoped) continue;
          deleteReqStateKeysByPrefix(state as Record<string, unknown>, `${group.prefix}${uid}::`);
        }
      }
      for (const group of AUTHZ_CACHE_GROUPS) {
        if (group.userScoped) continue;
        deleteReqStateKeysByPrefix(state as Record<string, unknown>, group.prefix);
      }
    }
  }

  invalidateJsCtxSymbolCache(jsCtx, Symbol.for('choysum.recordrule.cache'));
  invalidateJsCtxSymbolCache(jsCtx, Symbol.for('choysum.fieldrule.cache'));
}

/** Invalidate every request-scoped authz cache entry for all users. */
export function invalidateAllAuthzCaches(): void {
  invalidateAuthzRequestCaches({ allUsers: true });
}

/** Invalidate request-scoped authz cache entries for specific users. */
export function invalidateAuthzCachesForUsers(userIds: string[]): void {
  invalidateAuthzRequestCaches({ userIds });
}
