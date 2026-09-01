// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  mutateThenInvalidateAllAuthzCaches,
  mutateThenInvalidateAuthzCachesForUsers,
  userIdsFromUserRolePayloads,
} from '@/auth/service/mixins/authz_mutation_model';
import {
  buildAuthzContextCacheKey,
  buildMethodAccessCacheKey,
  buildUiGrantCacheKey,
} from '@/auth/service/models/_request_cache_invalidation';
import { ensureRequestContext, resetRequestContext } from '@/auth/service/tests/_request_context_fixtures';

function ensureServiceState(): Record<string, unknown> {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.req.__choysumServiceState || typeof jsCtx.req.__choysumServiceState !== 'object') {
    jsCtx.req.__choysumServiceState = {};
  }
  return jsCtx.req.__choysumServiceState as Record<string, unknown>;
}

test('authz mutation helpers: userIdsFromUserRolePayloads normalizes refs', () => {
  expect(userIdsFromUserRolePayloads(null)).toEqual([]);
  expect(userIdsFromUserRolePayloads({ UserId: '  u1  ' })).toEqual(['u1']);
  expect(userIdsFromUserRolePayloads([{ UserId: { Id: 'u1' } }, { UserId: 'u1' }, { UserId: 'u2' }])).toEqual(['u1', 'u2']);
});

test('authz mutation helpers: forUsers invalidation is targeted; all clears everyone', async () => {
  resetRequestContext();
  const state = ensureServiceState();
  const jsCtx = ensureRequestContext();

  const u1Ctx = buildAuthzContextCacheKey('u1', 'c1');
  const u2Ctx = buildAuthzContextCacheKey('u2', 'c1');
  const u1Method = buildMethodAccessCacheKey('u1', 'c1', '/auth.User/Browse');
  const u2Method = buildMethodAccessCacheKey('u2', 'c1', '/auth.User/Browse');
  const uiGrant = buildUiGrantCacheKey('role-sig');

  state[u1Ctx] = { roles: ['r1'] };
  state[u2Ctx] = { roles: ['r2'] };
  state[u1Method] = true;
  state[u2Method] = false;
  state[uiGrant] = ['res1'];
  (jsCtx as any)[Symbol.for('choysum.recordrule.cache')] = { warm: true };
  (jsCtx as any)[Symbol.for('choysum.fieldrule.cache')] = { warm: true };

  const result = await mutateThenInvalidateAuthzCachesForUsers(['u1'], async () => 'ok-for-users');
  expect(result).toBe('ok-for-users');

  // Targeted: only u1 user-scoped keys drop; uiGrant still cleared for safety.
  expect(state[u1Ctx]).toBe(undefined);
  expect(state[u1Method]).toBe(undefined);
  expect(state[u2Ctx]).toEqual({ roles: ['r2'] });
  expect(state[u2Method]).toBe(false);
  expect(state[uiGrant]).toBe(undefined);
  expect((jsCtx as any)[Symbol.for('choysum.recordrule.cache')]).toBe(undefined);
  expect((jsCtx as any)[Symbol.for('choysum.fieldrule.cache')]).toBe(undefined);

  // Re-warm leftover u2 + a fresh ui grant, then invalidate all.
  state[u2Ctx] = { roles: ['r2'] };
  state[u2Method] = false;
  state[uiGrant] = ['res2'];
  (jsCtx as any)[Symbol.for('choysum.recordrule.cache')] = { warm: true };

  const allResult = await mutateThenInvalidateAllAuthzCaches(async () => 'ok-all');
  expect(allResult).toBe('ok-all');
  expect(state[u2Ctx]).toBe(undefined);
  expect(state[u2Method]).toBe(undefined);
  expect(state[uiGrant]).toBe(undefined);
  expect((jsCtx as any)[Symbol.for('choysum.recordrule.cache')]).toBe(undefined);
});

test('authz mutation helpers: failed mutate does not invalidate', async () => {
  resetRequestContext();
  const state = ensureServiceState();
  const key = buildAuthzContextCacheKey('u9', 'c9');
  state[key] = { keep: true };

  let threw = false;
  try {
    await mutateThenInvalidateAllAuthzCaches(async () => {
      throw new Error('mutate boom');
    });
  } catch (e: any) {
    threw = String(e?.message || e).includes('mutate boom');
  }
  expect(threw).toBe(true);
  expect(state[key]).toEqual({ keep: true });
});

test('authz mutation helpers: failed mutate does not invalidate (forUsers)', async () => {
  resetRequestContext();
  const state = ensureServiceState();
  const key = buildAuthzContextCacheKey('u9', 'c9');
  state[key] = { keep: true };

  let threw = false;
  try {
    await mutateThenInvalidateAuthzCachesForUsers(['u9'], async () => {
      throw new Error('mutate boom');
    });
  } catch (e: any) {
    threw = String(e?.message || e).includes('mutate boom');
  }
  expect(threw).toBe(true);
  expect(state[key]).toEqual({ keep: true });
});
