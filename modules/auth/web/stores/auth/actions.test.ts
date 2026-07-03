// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';

let mockState: any;
let mockHelpers: any;
let mockAuthStorage: { clearAuthStorage: ReturnType<typeof vi.fn> };

// authStorage is imported by actions.ts; mock it so we can assert clearAuthStorage calls.
vi.mock('./storage', () => {
  mockAuthStorage = { clearAuthStorage: vi.fn() };
  return {
    authStorage: mockAuthStorage,
  };
});

// Mock @vueuse/core to avoid isClient checks interfering with clearAuth.
vi.mock('@vueuse/core', () => {
  return {
    isClient: true,
    useTimeoutFn: vi.fn(() => ({ stop: vi.fn() })),
  };
});

// Mock the error module.
vi.mock('../../error', () => {
  const actual = vi.importActual('../../error');
  return actual;
});

// Mock the utils module.
vi.mock('./utils', () => {
  return {
    hashPasswordClient: vi.fn(async (password: string) => password),
    getCsrfTokenFromCookie: vi.fn(() => null),
    getDeviceInfo: vi.fn(() => 'test-device'),
    withLoading: (fn: Function) => fn,
  };
});

function buildMockState() {
  return {
    tokens: { value: null },
    currentUser: { value: null },
    rememberMe: { value: false },
    loading: { value: false },
    initialized: { value: false },
    identity: { value: null },
    permissionState: { value: null },
    isAuthenticated: { value: false },
    isAccessTokenValid: { value: false },
    isRefreshTokenValid: { value: false },
    shouldRefreshToken: { value: false },
    refreshState: {
      timerId: null as number | null,
      refreshing: false,
      lastRefreshTime: 0,
    },
    authOptions: {
      autoRefresh: false,
      attachDeviceInfo: false,
      refreshThreshold: 30000,
      refreshInterval: 30000,
      defaultRedirect: '/',
    },
    userStore: {
      Login: vi.fn(),
      Logout: vi.fn(),
      Register: vi.fn(),
      RefreshTokens: vi.fn(),
      Browse: vi.fn(),
      GetPermissionState: vi.fn(),
      SwitchCompanyScope: vi.fn(),
    },
  };
}

function buildMockHelpers() {
  return {
    resetAuthState: vi.fn(),
    updateTokenIdentity: vi.fn(),
    setupRefreshTimer: vi.fn(),
    clearRefreshTimer: vi.fn(),
  };
}

describe('loginImpl', () => {
  beforeEach(() => {
    mockState = buildMockState();
    mockHelpers = buildMockHelpers();
    vi.clearAllMocks();
  });

  it('clears auth state before calling the Login RPC', async () => {
    // Arrange: a successful login response.
    const loginResponse = {
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      expiresAt: Date.now() + 3600_000,
    };
    mockState.userStore.Login.mockResolvedValue(loginResponse);

    const { defineAuthActions } = await import('./actions');
    const actions = defineAuthActions(mockState as any, mockHelpers as any);

    // Act: call login.
    await actions.login('admin', 'secret');

    // Assert: resetAuthState was called before the Login RPC.
    // clearAuth() calls helpers.resetAuthState() and authStorage.clearAuthStorage().
    expect(mockHelpers.resetAuthState).toHaveBeenCalled();
    expect(mockAuthStorage.clearAuthStorage).toHaveBeenCalled();

    // The Login RPC should have been called after clearAuth.
    expect(mockState.userStore.Login).toHaveBeenCalled();
    const resetCallOrder = mockHelpers.resetAuthState.mock.invocationCallOrder[0];
    const loginCallOrder = mockState.userStore.Login.mock.invocationCallOrder[0];
    expect(resetCallOrder).toBeLessThan(loginCallOrder);
  });

  it('clears auth state even when the access token is still unexpired', async () => {
    // Arrange: simulate an unexpired but invalid token (e.g. after key rotation).
    mockState.tokens.value = {
      accessToken: 'stale-access',
      refreshToken: 'stale-refresh',
      expiresAt: Date.now() + 3600_000, // still valid by time
    };
    mockState.isAccessTokenValid.value = true;

    const loginResponse = {
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      expiresAt: Date.now() + 3600_000,
    };
    mockState.userStore.Login.mockResolvedValue(loginResponse);

    const { defineAuthActions } = await import('./actions');
    const actions = defineAuthActions(mockState as any, mockHelpers as any);

    // Act: call login.
    await actions.login('admin', 'secret');

    // Assert: clearAuth was called unconditionally, despite isAccessTokenValid=true.
    expect(mockHelpers.resetAuthState).toHaveBeenCalled();
    expect(mockAuthStorage.clearAuthStorage).toHaveBeenCalled();
  });

  it('still logs in successfully when stale tokens exist', async () => {
    // Arrange: expired tokens from a previous session.
    mockState.tokens.value = {
      accessToken: 'stale-access',
      refreshToken: 'stale-refresh',
      expiresAt: Date.now() - 1000, // expired
    };
    mockState.isAccessTokenValid.value = false;

    const loginResponse = {
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      expiresAt: Date.now() + 3600_000,
    };
    mockState.userStore.Login.mockResolvedValue(loginResponse);

    const { defineAuthActions } = await import('./actions');
    const actions = defineAuthActions(mockState as any, mockHelpers as any);

    // Act: call login.
    await actions.login('admin', 'secret');

    // Assert: Login RPC was called despite stale tokens.
    expect(mockState.userStore.Login).toHaveBeenCalled();
    // clearAuth was called unconditionally before login.
    expect(mockHelpers.resetAuthState).toHaveBeenCalled();
  });

  it('awaits active initInFlight before clearing auth and logging in', async () => {
    // Arrange: expired tokens so initAuth will attempt a refresh.
    mockState.tokens.value = {
      accessToken: 'stale-access',
      refreshToken: 'stale-refresh',
      expiresAt: Date.now() - 1000,
    };
    mockState.isAccessTokenValid.value = false;
    mockState.shouldRefreshToken.value = true;

    // Deferred promise so we can control when initAuth completes.
    let resolveRefresh: (value: unknown) => void;
    const refreshDeferred = new Promise((resolve) => {
      resolveRefresh = resolve;
    });
    mockState.userStore.RefreshTokens.mockReturnValue(refreshDeferred);

    const loginResponse = {
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      expiresAt: Date.now() + 3600_000,
    };
    mockState.userStore.Login.mockResolvedValue(loginResponse);

    const { defineAuthActions } = await import('./actions');
    const actions = defineAuthActions(mockState as any, mockHelpers as any);

    // Start initAuth without awaiting — this sets initInFlight.
    const initPromise = actions.ensureAuthReady();
    // Let initAuth reach the pending RefreshTokens call.
    await Promise.resolve();

    // Act: call login while initInFlight is still active.
    const loginPromise = actions.login('admin', 'secret');

    // Let login reach the `await initInFlight` point.
    await Promise.resolve();

    // Resolve initAuth's RefreshTokens so both initAuth and login can proceed.
    resolveRefresh!({
      accessToken: 'refreshed-access',
      refreshToken: 'refreshed-refresh',
      expiresAt: Date.now() + 3600_000,
    });

    await initPromise;
    await loginPromise;

    // Assert: login completed, clearAuth was called, Login RPC was called.
    expect(mockHelpers.resetAuthState).toHaveBeenCalled();
    expect(mockState.userStore.Login).toHaveBeenCalled();
  });

  it('catches initInFlight failure and still proceeds to login', async () => {
    // Arrange: expired tokens so initAuth refreshes, but RefreshTokens fails.
    mockState.tokens.value = {
      accessToken: 'stale-access',
      refreshToken: 'stale-refresh',
      expiresAt: Date.now() - 1000,
    };
    mockState.isAccessTokenValid.value = false;
    mockState.shouldRefreshToken.value = true;

    // Deferred promise that will reject.
    let rejectRefresh: (reason: unknown) => void;
    const refreshDeferred = new Promise((_resolve, reject) => {
      rejectRefresh = reject;
    });
    mockState.userStore.RefreshTokens.mockReturnValue(refreshDeferred);

    const loginResponse = {
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      expiresAt: Date.now() + 3600_000,
    };
    mockState.userStore.Login.mockResolvedValue(loginResponse);

    const { defineAuthActions } = await import('./actions');
    const actions = defineAuthActions(mockState as any, mockHelpers as any);

    // Start initAuth without awaiting.
    const initPromise = actions.ensureAuthReady();
    await Promise.resolve();

    // Act: call login while initInFlight is active.
    const loginPromise = actions.login('admin', 'secret');
    await Promise.resolve();

    // initAuth fails — the catch block in loginImpl should swallow it.
    rejectRefresh!(new Error('token signature is invalid'));

    // initAuth should settle (with error), login should still succeed.
    await expect(initPromise).rejects.toThrow();
    await loginPromise;

    // Assert: login still completed — Login RPC was called.
    expect(mockState.userStore.Login).toHaveBeenCalled();
    // clearAuth was called (both from initAuth's catch and loginImpl).
    expect(mockHelpers.resetAuthState).toHaveBeenCalled();
  });
});
