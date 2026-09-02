// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { uniqStrings } from '@/core/service/utils/normalization';
import { getJsCtxAndReq, deleteReqStateKeysByPrefix, invalidateJsCtxSymbolCache } from './context';

const AUTHZ_CTX_PREFIX = 'authzContext::';
const METHOD_ACCESS_PREFIX = 'methodAccess::';
const UI_GRANT_PREFIX = 'uiGrantExpansion::';

type InvalidateOpts = {
  userIds?: string[];
  allUsers?: boolean;
};

function asTrimmedString(value: unknown): string {
  if (value == null) return '';
  return String(value).trim();
}

/** Build request-scoped authz context cache key. */
export function buildAuthzContextCacheKey(userId: string, companyScopeKey: string): string {
  return `${AUTHZ_CTX_PREFIX}${asTrimmedString(userId)}::${asTrimmedString(companyScopeKey)}`;
}

/** Build request-scoped method-access cache key. */
export function buildMethodAccessCacheKey(userId: string, companyId: string, fullMethod: string): string {
  return `${METHOD_ACCESS_PREFIX}${asTrimmedString(userId)}::${asTrimmedString(companyId)}::${asTrimmedString(fullMethod)}`;
}

/** Build request-scoped UI-grant expansion cache key. */
export function buildUiGrantCacheKey(roleSignature: string): string {
  return `${UI_GRANT_PREFIX}${asTrimmedString(roleSignature)}`;
}

function resolveServiceState(req: unknown): Record<string, unknown> | undefined {
  if (req == null) return undefined;
  const state = (req as { __choysumServiceState?: unknown }).__choysumServiceState;
  if (state == null) return undefined;
  if (typeof state !== 'object') return undefined;
  return state as Record<string, unknown>;
}

/**
 * Invalidate request-scoped auth caches after permission graph changes.
 *
 * Clears memoized authz context entries on req.__choysumServiceState and
 * request-level record/field rule caches on jsCtx.
 */
export function invalidateAuthzRequestCaches(opts: InvalidateOpts = {}): void {
  const { jsCtx, req } = getJsCtxAndReq();
  const ids = uniqStrings(opts.userIds == null ? [] : opts.userIds);
  const invalidateAll = opts.allUsers === true ? true : ids.length === 0;
  const record = resolveServiceState(req);

  if (record !== undefined) {
    if (invalidateAll) {
      deleteReqStateKeysByPrefix(record, AUTHZ_CTX_PREFIX);
      deleteReqStateKeysByPrefix(record, METHOD_ACCESS_PREFIX);
      deleteReqStateKeysByPrefix(record, UI_GRANT_PREFIX);
    } else {
      for (const uid of ids) {
        deleteReqStateKeysByPrefix(record, `${AUTHZ_CTX_PREFIX}${uid}::`);
        deleteReqStateKeysByPrefix(record, `${METHOD_ACCESS_PREFIX}${uid}::`);
      }
      deleteReqStateKeysByPrefix(record, UI_GRANT_PREFIX);
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
