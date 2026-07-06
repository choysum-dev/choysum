// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * AuthCacheManager provides structured invalidation of request-scoped
 * authorization caches.  Models should call these methods instead of
 * directly deleting keys from req.__choysumServiceState.
 */

type InvalidateOpts = {
  userIds?: string[];
  allUsers?: boolean;
};

/**
 * Resolve the current request context from the Go-injected jsCtx map.
 * Prefers the proper `$choysum.request.context` path set by choysumrpc.js
 * at request dispatch time.
 */
function resolveRequestState(): { jsCtx: Record<string, unknown>; req: Record<string, unknown> } {
  const root: any = (globalThis as any)?.$choysum;

  let jsCtx: any;
  const getRequestContext = root?.getRequestContext;
  if (typeof getRequestContext === 'function') {
    try {
      jsCtx = getRequestContext();
    } catch {
      jsCtx = undefined;
    }
  }

  if (!jsCtx) {
    const getActiveRequest = root?.getActiveRequest;
    if (typeof getActiveRequest === 'function') {
      try {
        jsCtx = getActiveRequest()?.context;
      } catch {
        jsCtx = undefined;
      }
    }
  }

  jsCtx = (jsCtx ?? root?.request?.context ?? root?.context ?? root) as any;
  const req: any = (jsCtx?.req ?? {}) as any;
  return { jsCtx, req };
}

const AUTHZ_PREFIXES = ['authzContext::', 'methodAccess::', 'uiGrantExpansion::', 'uiRequiresMatch::'] as const;

/**
 * Invalidate all authz-context, method-access, and ui-rule memoization
 * entries stored in the per-request service state.
 */
function invalidateStateCache(state: Record<string, unknown>, opts: InvalidateOpts): void {
  if (!state || typeof state !== 'object') return;

  const invalidateAll = Boolean(opts.allUsers) || !Array.isArray(opts.userIds) || opts.userIds.length === 0;

  if (invalidateAll) {
    for (const k of Object.keys(state)) {
      for (const prefix of AUTHZ_PREFIXES) {
        if (k.startsWith(prefix)) {
          delete state[k];
          break;
        }
      }
    }
    return;
  }

  for (const uid of opts.userIds!) {
    const id = String(uid ?? '').trim();
    if (!id) continue;
    for (const k of Object.keys(state)) {
      if (k.startsWith(`authzContext::${id}::`) || k.startsWith(`methodAccess::${id}::`)) {
        delete state[k];
      }
    }
  }

  // uiGrantExpansion / uiRequiresMatch are role-set scoped and do not embed
  // userId.  Clear them whenever targeted invalidation occurs.
  for (const k of Object.keys(state)) {
    if (k.startsWith('uiGrantExpansion::') || k.startsWith('uiRequiresMatch::')) {
      delete state[k];
    }
  }
}

/**
 * Invalidate request-level record-rule and field-rule caches stored as
 * well-known Symbol-keyed properties on the JS context object.
 */
function invalidateRuleCaches(jsCtx: Record<string, unknown>): void {
  try {
    delete (jsCtx as any)[Symbol.for('choysum.recordrule.cache')];
    delete (jsCtx as any)[Symbol.for('choysum.fieldrule.cache')];
  } catch {
    // ignore frozen / non-configurable objects
  }
}

/**
 * Invalidate request-scoped authorization caches after permission-graph
 * changes (e.g. role assignment, rule update, module install/uninstall).
 *
 * Clears:
 * - memoized authz-context entries in req.__choysumServiceState
 * - request-level record/field rule caches on the JS context
 */
export function invalidateRequestCaches(opts: InvalidateOpts = {}): void {
  const { jsCtx, req } = resolveRequestState();
  const state: any = req?.__choysumServiceState;

  invalidateStateCache(state, opts);
  invalidateRuleCaches(jsCtx);
}
