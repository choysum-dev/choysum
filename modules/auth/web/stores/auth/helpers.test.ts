// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, reactive, computed } from 'vue';
import {
  clearCompanyScopeView,
  publishCompanyScopeView,
  useCompanyScopeViewKey,
} from '@/web/web/stores/companyScopeView';
import { defineAuthHelpers } from './helpers';

/**
 * Build a minimal unsigned JWT for identity extraction in unit tests.
 */
function makeJwt(payload: Record<string, unknown>): string {
  const enc = (obj: unknown) => {
    const json = JSON.stringify(obj);
    const b64 = btoa(json).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
    return b64;
  };
  return `${enc({ alg: 'none', typ: 'JWT' })}.${enc(payload)}.sig`;
}

/**
 * Minimal auth state surface for helper unit tests.
 */
function buildState(overrides: Record<string, unknown> = {}) {
  const tokens = ref<any>(null);
  const identity = ref<any>(null);
  const currentUser = ref<any>(null);
  const rememberMe = ref(false);
  const refreshState = reactive({
    timerId: null as number | null,
    refreshing: false,
    lastRefreshTime: 0,
  });
  return {
    tokens,
    identity,
    currentUser,
    rememberMe,
    refreshState,
    authOptions: {
      refreshThreshold: 60_000,
      refreshInterval: 30_000,
      autoRefresh: false,
    },
    shouldRefreshToken: computed(() => false),
    ...overrides,
  } as any;
}

test('updateTokenIdentity: publishes company scope from token metadata', () => {
  clearCompanyScopeView();
  const state = buildState();
  state.tokens.value = {
    accessToken: makeJwt({
      sub: 'usr_1',
      jti: 'tok_1',
      meta: { activeCompanyId: 'cmp_a', enabledCompanyIds: ['cmp_a', 'cmp_b'] },
    }),
    refreshToken: 'r1',
    expiresAt: Date.now() + 60_000,
  };
  const helpers = defineAuthHelpers(state);
  expect(helpers.updateTokenIdentity()).toBe(true);
  expect(state.identity.value?.userId).toBe('usr_1');
  expect(useCompanyScopeViewKey().value).toBe('cmp_a|cmp_a,cmp_b');
});

test('updateTokenIdentity: clears company scope when access token is missing', () => {
  publishCompanyScopeView('cmp_x', ['cmp_x']);
  const state = buildState();
  state.tokens.value = { accessToken: '', refreshToken: 'r1', expiresAt: Date.now() + 60_000 };
  const helpers = defineAuthHelpers(state);
  expect(helpers.updateTokenIdentity()).toBe(false);
  expect(state.identity.value).toBe(null);
  expect(useCompanyScopeViewKey().value).toBe('|');
});

test('updateTokenIdentity: clears company scope when identity extraction fails', () => {
  publishCompanyScopeView('cmp_x', ['cmp_x']);
  const state = buildState();
  state.tokens.value = {
    accessToken: 'not-a-jwt',
    refreshToken: 'r1',
    expiresAt: Date.now() + 60_000,
  };
  const helpers = defineAuthHelpers(state);
  let caught: unknown;
  try {
    helpers.updateTokenIdentity();
  } catch (err) {
    caught = err;
  }
  expect(Boolean(caught)).toBe(true);
  expect(state.identity.value).toBe(null);
  expect(useCompanyScopeViewKey().value).toBe('|');
});

test('resetAuthState: clears published company scope', () => {
  publishCompanyScopeView('cmp_a', ['cmp_a']);
  const state = buildState();
  state.tokens.value = { accessToken: 'a', refreshToken: 'r', expiresAt: Date.now() + 60_000 };
  state.identity.value = { userId: 'usr_1' };
  state.currentUser.value = { Id: 'usr_1' };
  state.rememberMe.value = true;
  const helpers = defineAuthHelpers(state);
  helpers.resetAuthState();
  expect(state.tokens.value).toBe(null);
  expect(state.identity.value).toBe(null);
  expect(useCompanyScopeViewKey().value).toBe('|');
});
