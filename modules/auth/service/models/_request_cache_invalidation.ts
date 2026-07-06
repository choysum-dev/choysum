// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeStringArray } from '@/core/service/utils/strings';

type InvalidateOpts = {
  userIds?: string[];
  allUsers?: boolean;
};

/**
 * Resolve the current request context and backing request object.
 */
function getJsCtxAndReq(): { jsCtx: any; req: any } {
  const root: any = (globalThis as any)?.$choysum;
  const jsCtx: any = (root?.request?.context ?? root?.context ?? root) as any;
  const req: any = (jsCtx?.req ?? jsCtx?.request?.context?.req ?? jsCtx?.context?.req) as any;
  return { jsCtx, req };
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

  const ids = normalizeStringArray(opts.userIds || []);
  const invalidateAll = Boolean(opts.allUsers) || ids.length === 0;

  // 1) Invalidate authz context memoization
  if (state && typeof state === 'object') {
    if (invalidateAll) {
      for (const k of Object.keys(state)) {
        if (k.startsWith('authzContext::')) delete state[k];
        if (k.startsWith('methodAccess::')) delete state[k];
        if (k.startsWith('uiGrantExpansion::')) delete state[k];
        if (k.startsWith('uiRequiresMatch::')) delete state[k];
      }
    } else {
      for (const uid of ids) {
        const prefix = `authzContext::${uid}::`;
        for (const k of Object.keys(state)) {
          if (k.startsWith(prefix)) delete state[k];
          if (k.startsWith(`methodAccess::${uid}::`)) delete state[k];
        }
      }
      // uiGrantExpansion/uiRequiresMatch are role-set scoped and do not embed userId.
      // For safety, clear them when any targeted invalidation happens.
      for (const k of Object.keys(state)) {
        if (k.startsWith('uiGrantExpansion::')) delete state[k];
        if (k.startsWith('uiRequiresMatch::')) delete state[k];
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
