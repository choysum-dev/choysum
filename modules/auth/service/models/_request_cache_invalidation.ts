// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { uniqStrings } from '@/core/service/utils/normalization';
import { getJsCtxAndReq } from './_user_runtime_context';

export const AUTHZ_CTX_PREFIX = 'authzContext::';
export const METHOD_ACCESS_PREFIX = 'methodAccess::';
export const UI_GRANT_PREFIX = 'uiGrantExpansion::';
export const UI_REQUIRES_PREFIX = 'uiRequiresMatch::';

type InvalidateOpts = {
  userIds?: string[];
  allUsers?: boolean;
};

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
      for (const k of Object.keys(state)) {
        if (k.startsWith(AUTHZ_CTX_PREFIX)) delete state[k];
        if (k.startsWith(METHOD_ACCESS_PREFIX)) delete state[k];
        if (k.startsWith(UI_GRANT_PREFIX)) delete state[k];
        if (k.startsWith(UI_REQUIRES_PREFIX)) delete state[k];
      }
    } else {
      for (const uid of ids) {
        const prefix = `${AUTHZ_CTX_PREFIX}${uid}::`;
        for (const k of Object.keys(state)) {
          if (k.startsWith(prefix)) delete state[k];
          if (k.startsWith(`${METHOD_ACCESS_PREFIX}${uid}::`)) delete state[k];
        }
      }
      // uiGrantExpansion/uiRequiresMatch are role-set scoped and do not embed userId.
      // For safety, clear them when any targeted invalidation happens.
      for (const k of Object.keys(state)) {
        if (k.startsWith(UI_GRANT_PREFIX)) delete state[k];
        if (k.startsWith(UI_REQUIRES_PREFIX)) delete state[k];
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
