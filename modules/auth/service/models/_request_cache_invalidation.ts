// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { uniqStrings } from '@/core/service/utils/normalization';
import { getJsCtxAndReq } from '@/core/service/api/context';

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

/**
 * Build request-scoped authz context cache key.
 */
export function buildAuthzContextCacheKey(userId: string, companyScopeKey: string): string {
  return `${AUTHZ_CTX_PREFIX}${String(userId || '').trim()}::${String(companyScopeKey || '').trim()}`;
}

/**
 * Build request-scoped method-access cache key.
 */
export function buildMethodAccessCacheKey(userId: string, companyId: string, fullMethod: string): string {
  return `${METHOD_ACCESS_PREFIX}${String(userId || '').trim()}::${String(companyId || '').trim()}::${String(fullMethod || '').trim()}`;
}

/**
 * Build request-scoped UI-grant expansion cache key.
 */
export function buildUiGrantCacheKey(roleSignature: string): string {
  return `${UI_GRANT_PREFIX}${String(roleSignature || '').trim()}`;
}

function deleteCacheByPrefix(state: Record<string, unknown>, prefix: string): void {
  for (const key of Object.keys(state)) {
    if (key.startsWith(prefix)) delete state[key];
  }
}

/**
 * Invalidate request-scoped auth caches after permission graph changes.
 *
 * Clears:
 * - memoized authz context entries stored in req.__choysumServiceState (User._getAuthzContext)
 * - request-level record/field rule caches on jsCtx (Symbol.for('choysum.recordrule.cache') / Symbol.for('choysum.fieldrule.cache'))
 */
export function invalidateAuthzRequestCaches(opts: InvalidateOpts = {}): void {
  const { jsCtx, req } = getJsCtxAndReq();
  const state: any = req?.__choysumServiceState;

  const ids = uniqStrings(opts.userIds || []);
  const invalidateAll = Boolean(opts.allUsers) || ids.length === 0;

  // 1) Invalidate authz context memoization
  if (state && typeof state === 'object') {
    if (invalidateAll) {
      for (const group of AUTHZ_CACHE_GROUPS) {
        deleteCacheByPrefix(state as Record<string, unknown>, group.prefix);
      }
    } else {
      for (const uid of ids) {
        for (const group of AUTHZ_CACHE_GROUPS) {
          if (!group.userScoped) continue;
          deleteCacheByPrefix(state as Record<string, unknown>, `${group.prefix}${uid}::`);
        }
      }
      // uiGrantExpansion keys are role-set scoped and do not embed userId.
      // For safety, clear them when any targeted invalidation happens.
      for (const group of AUTHZ_CACHE_GROUPS) {
        if (group.userScoped) continue;
        deleteCacheByPrefix(state as Record<string, unknown>, group.prefix);
      }
    }
  }

  // 2) Invalidate request-level record/field rule caches
  try {
    delete (jsCtx as any)[Symbol.for('choysum.recordrule.cache')];
    delete (jsCtx as any)[Symbol.for('choysum.fieldrule.cache')];
  } catch {
    // ignore
  }
}

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
