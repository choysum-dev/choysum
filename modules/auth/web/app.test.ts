// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';

let mockAuthStore: any;
let capturedTokenProvider: any = null;

// Mock useAuthStore.
vi.mock('./stores/auth', () => {
  return {
    useAuthStore: () => mockAuthStore,
  };
});

// Mock @/core/web/rpc to capture the token provider.
vi.mock('@/core/web/rpc', () => {
  return {
    setTokenProvider: (provider: any) => {
      capturedTokenProvider = typeof provider === 'function' ? provider() : provider;
    },
    setCSRFProvider: vi.fn(),
  };
});

// Mock Vue reactivity APIs used by app.ts.
vi.mock('vue', () => {
  return {
    watch: vi.fn(),
    ref: (val: any) => ({ value: val }),
    computed: (fn: Function) => ({ value: fn() }),
  };
});

// Mock @/web/web (the app instance).
vi.mock('@/web/web', () => {
  return {
    default: {
      setup: (fn: Function) => fn({} as any),
      menu: { getMenus: () => [] },
      unmount: vi.fn(),
    },
  };
});

// Mock @/web/web/directives/action.
vi.mock('@/web/web/directives/action', () => {
  return { setGlobalActionChecker: vi.fn() };
});

// Mock the route module.
vi.mock('./route', () => {
  return { setupRouter: vi.fn() };
});

// Mock the menu module.
vi.mock('./menu', () => {
  return { setupAppMenu: vi.fn() };
});

// Mock menu/applyPermissionToMenus.
vi.mock('./menu/applyPermissionToMenus', () => {
  return { applyPermissionToMenus: vi.fn() };
});

// Mock permission module.
vi.mock('@/auth/web/permission', () => {
  return { hasAction: () => true };
});

function buildMockAuthStore(overrides: Record<string, unknown> = {}) {
  return {
    tokens: null,
    shouldRefreshToken: false,
    refreshToken: vi.fn(),
    logout: vi.fn(),
    isAuthenticated: false,
    identity: null,
    permissionState: null,
    loadPermissionState: vi.fn(),
    clearAuth: vi.fn(),
    getCsrfToken: vi.fn(() => null),
    ...overrides,
  };
}

describe('setupTokenProvider refreshToken', () => {
  beforeEach(() => {
    capturedTokenProvider = null;
    vi.clearAllMocks();
    // Reset module cache so app.ts re-evaluates with fresh mocks.
    vi.resetModules();
  });

  it('uses console.warn (not console.error) when token refresh fails', async () => {
    // Arrange: auth store has a refresh token, but refresh fails.
    mockAuthStore = buildMockAuthStore({
      tokens: { accessToken: 'old-access', refreshToken: 'old-refresh', expiresAt: Date.now() + 1000 },
      refreshToken: vi.fn(async () => {
        throw new Error('token signature is invalid');
      }),
      shouldRefreshToken: false,
    });

    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    // Act: import app.ts to trigger setupApp -> setupTokenProvider.
    await import('./app');

    // The token provider should have been captured.
    expect(capturedTokenProvider).not.toBeNull();

    // Call the refreshToken callback.
    const result = await capturedTokenProvider.refreshToken();

    // Assert: refresh failed (returns false).
    expect(result).toBe(false);

    // console.warn should have been called (not console.error).
    expect(warnSpy).toHaveBeenCalledWith('[Auth] Token refresh failed:', expect.any(Error));
    // No console.error for the refresh failure path.
    const refreshErrorCalls = errorSpy.mock.calls.filter(args => typeof args[0] === 'string' && args[0].includes('Token refresh failed'));
    expect(refreshErrorCalls).toHaveLength(0);

    warnSpy.mockRestore();
    errorSpy.mockRestore();
  });

  it('returns false when token refresh fails', async () => {
    // Arrange: auth store has a refresh token; refresh fails and clears tokens.
    mockAuthStore = buildMockAuthStore({
      tokens: { accessToken: 'old-access', refreshToken: 'old-refresh', expiresAt: Date.now() + 1000 },
      refreshToken: vi.fn(async () => {
        // Simulate the store action clearing tokens on failure.
        mockAuthStore.tokens = null;
        throw new Error('token signature is invalid');
      }),
      shouldRefreshToken: false,
    });

    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    // Act: import app.ts to trigger setupApp -> setupTokenProvider.
    await import('./app');

    // Call the refreshToken callback.
    const result = await capturedTokenProvider.refreshToken();

    // Assert: returns false on failure.
    expect(result).toBe(false);
    expect(warnSpy).toHaveBeenCalled();

    warnSpy.mockRestore();
  });
});
