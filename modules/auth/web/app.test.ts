// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getTokenProvider, setTokenProvider } from '@/core/web/rpc/providers';
import { setupTokenProvider } from './setup_token_provider';

type Call = { args: unknown[] };

function makeFn() {
  const calls: Call[] = [];
  let impl: ((...args: any[]) => any) | undefined;
  const fn = (...args: any[]) => {
    calls.push({ args });
    return impl ? impl(...args) : undefined;
  };
  return {
    fn,
    calls,
    reject(err: unknown) {
      impl = async () => {
        throw err;
      };
    },
    impl(next: (...args: any[]) => any) {
      impl = next;
    },
  };
}

function buildMockAuthStore(overrides: Record<string, unknown> = {}) {
  return {
    tokens: null as null | { accessToken: string; refreshToken: string; expiresAt: number },
    shouldRefreshToken: false,
    refreshToken: makeFn().fn,
    logout: makeFn().fn,
    isAuthenticated: false,
    identity: null,
    permissionState: null,
    loadPermissionState: makeFn().fn,
    clearAuth: makeFn().fn,
    getCsrfToken: () => null,
    ...overrides,
  };
}

test('setupTokenProvider: uses console.warn (not console.error) when token refresh fails', async () => {
  setTokenProvider(null);
  const refreshToken = makeFn();
  refreshToken.reject(new Error('token signature is invalid'));
  const mockAuthStore = buildMockAuthStore({
    tokens: { accessToken: 'old-access', refreshToken: 'old-refresh', expiresAt: Date.now() + 1000 },
    refreshToken: refreshToken.fn,
  });

  const prevWarn = console.warn;
  const prevError = console.error;
  const warnCalls: unknown[][] = [];
  const errorCalls: unknown[][] = [];
  console.warn = (...args: unknown[]) => {
    warnCalls.push(args);
  };
  console.error = (...args: unknown[]) => {
    errorCalls.push(args);
  };

  try {
    setupTokenProvider(mockAuthStore as any);
    const provider = getTokenProvider();
    expect(provider).toBeTruthy();
    const result = await provider!.refreshToken();
    expect(result).toBe(false);
    expect(warnCalls.length).toBe(1);
    expect(String(warnCalls[0]?.[0])).toContain('[Auth] Token refresh failed:');
    const refreshErrorCalls = errorCalls.filter(args => typeof args[0] === 'string' && String(args[0]).includes('Token refresh failed'));
    expect(refreshErrorCalls.length).toBe(0);
  } finally {
    console.warn = prevWarn;
    console.error = prevError;
    setTokenProvider(null);
  }
});

test('setupTokenProvider: returns false when token refresh fails', async () => {
  setTokenProvider(null);
  const refreshToken = makeFn();
  const mockAuthStore = buildMockAuthStore({
    tokens: { accessToken: 'old-access', refreshToken: 'old-refresh', expiresAt: Date.now() + 1000 },
    refreshToken: refreshToken.fn,
  });
  refreshToken.impl(async () => {
    mockAuthStore.tokens = null;
    throw new Error('token signature is invalid');
  });

  const prevWarn = console.warn;
  console.warn = () => {};
  try {
    setupTokenProvider(mockAuthStore as any);
    const provider = getTokenProvider();
    const result = await provider!.refreshToken();
    expect(result).toBe(false);
  } finally {
    console.warn = prevWarn;
    setTokenProvider(null);
  }
});
