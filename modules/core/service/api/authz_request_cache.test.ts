// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  buildAuthzContextCacheKey,
  buildMethodAccessCacheKey,
  buildUiGrantCacheKey,
  invalidateAllAuthzCaches,
  invalidateAuthzCachesForUsers,
  invalidateAuthzRequestCaches,
} from './authz_request_cache';

function setRequest(req: any): () => void {
  const previous = (globalThis as any).$choysum;
  const current = previous || {};
  (globalThis as any).$choysum = { ...current, request: req };
  return () => {
    if (previous === undefined) {
      delete (globalThis as any).$choysum;
      return;
    }
    (globalThis as any).$choysum = previous;
  };
}

test('authz_request_cache: cache key builders trim inputs', () => {
  const authzCtx = buildAuthzContextCacheKey(' u1 ', ' c1 ');
  if (authzCtx !== 'authzContext::u1::c1') {
    throw new Error(`authzCtx = ${authzCtx}`);
  }
  const method = buildMethodAccessCacheKey(' u1 ', ' c1 ', ' /auth.User/Browse ');
  if (method !== 'methodAccess::u1::c1::/auth.User/Browse') {
    throw new Error(`method = ${method}`);
  }
  const uiGrant = buildUiGrantCacheKey(' role-sig ');
  if (uiGrant !== 'uiGrantExpansion::role-sig') {
    throw new Error(`uiGrant = ${uiGrant}`);
  }
});

test('authz_request_cache: targeted and global invalidation clear memoized entries', () => {
  const jsCtx: any = { req: { __choysumServiceState: {} as Record<string, unknown> } };
  const restore = setRequest({ context: jsCtx });
  try {
    const state = jsCtx.req.__choysumServiceState as Record<string, unknown>;
    const u1Ctx = buildAuthzContextCacheKey('u1', 'c1');
    const u2Ctx = buildAuthzContextCacheKey('u2', 'c1');
    const u1Method = buildMethodAccessCacheKey('u1', 'c1', '/auth.User/Browse');
    const uiGrant = buildUiGrantCacheKey('role-sig');

    state[u1Ctx] = { roles: ['r1'] };
    state[u2Ctx] = { roles: ['r2'] };
    state[u1Method] = true;
    state[uiGrant] = ['res1'];
    jsCtx[Symbol.for('choysum.recordrule.cache')] = { warm: true };
    jsCtx[Symbol.for('choysum.fieldrule.cache')] = { warm: true };

    invalidateAuthzCachesForUsers(['u1']);
    if (state[u1Ctx] !== undefined) throw new Error('u1Ctx not cleared');
    if (state[u1Method] !== undefined) throw new Error('u1Method not cleared');
    if (JSON.stringify(state[u2Ctx]) !== JSON.stringify({ roles: ['r2'] })) {
      throw new Error('u2Ctx should remain');
    }
    if (state[uiGrant] !== undefined) throw new Error('uiGrant should clear');
    if (jsCtx[Symbol.for('choysum.recordrule.cache')] !== undefined) {
      throw new Error('recordrule cache not cleared');
    }
    if (jsCtx[Symbol.for('choysum.fieldrule.cache')] !== undefined) {
      throw new Error('fieldrule cache not cleared');
    }

    state[u2Ctx] = { roles: ['r2'] };
    state[uiGrant] = ['res2'];
    jsCtx[Symbol.for('choysum.recordrule.cache')] = { warm: true };

    invalidateAuthzRequestCaches({ allUsers: true });
    if (state[u2Ctx] !== undefined) throw new Error('u2Ctx not cleared on allUsers');
    if (state[uiGrant] !== undefined) throw new Error('uiGrant not cleared on allUsers');
    if (jsCtx[Symbol.for('choysum.recordrule.cache')] !== undefined) {
      throw new Error('recordrule cache not cleared on allUsers');
    }

    invalidateAllAuthzCaches();
    if (Object.keys(state).length !== 0) {
      throw new Error(`expected empty state, got ${Object.keys(state).join(',')}`);
    }
  } finally {
    restore();
  }
});
