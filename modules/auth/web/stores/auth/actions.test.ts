// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineAuthActions } from './actions';

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
    resolve(value: unknown) {
      impl = async () => value;
    },
    returnValue(value: unknown) {
      impl = () => value;
    },
  };
}

function buildMockState() {
  const Login = makeFn();
  const Logout = makeFn();
  const Register = makeFn();
  const RefreshTokens = makeFn();
  const Browse = makeFn();
  const GetPermissionState = makeFn();
  const SwitchCompanyScope = makeFn();
  return {
    tokens: { value: null as any },
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
      Login: Login.fn,
      Logout: Logout.fn,
      Register: Register.fn,
      RefreshTokens: RefreshTokens.fn,
      Browse: Browse.fn,
      GetPermissionState: GetPermissionState.fn,
      SwitchCompanyScope: SwitchCompanyScope.fn,
    },
    _recorders: { Login, Logout, Register, RefreshTokens, Browse, GetPermissionState, SwitchCompanyScope },
  };
}

function buildMockHelpers() {
  const resetAuthState = makeFn();
  const updateTokenIdentity = makeFn();
  const setupRefreshTimer = makeFn();
  const clearRefreshTimer = makeFn();
  return {
    resetAuthState: resetAuthState.fn,
    updateTokenIdentity: updateTokenIdentity.fn,
    setupRefreshTimer: setupRefreshTimer.fn,
    clearRefreshTimer: clearRefreshTimer.fn,
    _recorders: { resetAuthState, updateTokenIdentity, setupRefreshTimer, clearRefreshTimer },
  };
}

test('loginImpl: clears auth state before calling the Login RPC', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  const clearAuthStorage = makeFn();
  mockState._recorders.Login.resolve({
    accessToken: 'new-access',
    refreshToken: 'new-refresh',
    expiresAt: Date.now() + 3600_000,
  });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    clearAuthStorage: clearAuthStorage.fn,
  });

  await actions.login('admin', 'secret');

  expect(mockHelpers._recorders.resetAuthState.calls.length).toBeGreaterThan(0);
  expect(clearAuthStorage.calls.length).toBeGreaterThan(0);
  expect(mockState._recorders.Login.calls.length).toBe(1);
});

test('loginImpl: clears auth state even when the access token is still unexpired', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  const clearAuthStorage = makeFn();
  mockState.tokens.value = {
    accessToken: 'stale-access',
    refreshToken: 'stale-refresh',
    expiresAt: Date.now() + 3600_000,
  };
  mockState.isAccessTokenValid.value = true;
  mockState._recorders.Login.resolve({
    accessToken: 'new-access',
    refreshToken: 'new-refresh',
    expiresAt: Date.now() + 3600_000,
  });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    clearAuthStorage: clearAuthStorage.fn,
  });

  await actions.login('admin', 'secret');

  expect(mockHelpers._recorders.resetAuthState.calls.length).toBeGreaterThan(0);
  expect(clearAuthStorage.calls.length).toBeGreaterThan(0);
});

test('loginImpl: still logs in successfully when stale tokens exist', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'stale-access',
    refreshToken: 'stale-refresh',
    expiresAt: Date.now() - 1000,
  };
  mockState.isAccessTokenValid.value = false;
  mockState._recorders.Login.resolve({
    accessToken: 'new-access',
    refreshToken: 'new-refresh',
    expiresAt: Date.now() + 3600_000,
  });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    clearAuthStorage: () => {},
  });

  await actions.login('admin', 'secret');

  expect(mockState._recorders.Login.calls.length).toBe(1);
  expect(mockHelpers._recorders.resetAuthState.calls.length).toBeGreaterThan(0);
});

test('loginImpl: awaits active initInFlight before clearing auth and logging in', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'stale-access',
    refreshToken: 'stale-refresh',
    expiresAt: Date.now() - 1000,
  };
  mockState.isAccessTokenValid.value = false;
  mockState.shouldRefreshToken.value = true;

  let resolveRefresh: (value: unknown) => void = () => {};
  const refreshDeferred = new Promise(resolve => {
    resolveRefresh = resolve;
  });
  mockState._recorders.RefreshTokens.returnValue(refreshDeferred);
  mockState._recorders.Login.resolve({
    accessToken: 'new-access',
    refreshToken: 'new-refresh',
    expiresAt: Date.now() + 3600_000,
  });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    clearAuthStorage: () => {},
  });

  const initPromise = actions.ensureAuthReady();
  await Promise.resolve();
  const loginPromise = actions.login('admin', 'secret');
  await Promise.resolve();

  resolveRefresh({
    accessToken: 'refreshed-access',
    refreshToken: 'refreshed-refresh',
    expiresAt: Date.now() + 3600_000,
  });

  await initPromise;
  await loginPromise;

  expect(mockHelpers._recorders.resetAuthState.calls.length).toBeGreaterThan(0);
  expect(mockState._recorders.Login.calls.length).toBe(1);
});

test('loginImpl: catches initInFlight failure and still proceeds to login', async () => {
  const prevWarn = console.warn;
  console.warn = () => {};
  try {
    const mockState = buildMockState();
    const mockHelpers = buildMockHelpers();
    mockState.tokens.value = {
      accessToken: 'stale-access',
      refreshToken: 'stale-refresh',
      expiresAt: Date.now() - 1000,
    };
    mockState.isAccessTokenValid.value = false;
    mockState.shouldRefreshToken.value = true;

    let rejectRefresh: (reason: unknown) => void = () => {};
    const refreshDeferred = new Promise((_resolve, reject) => {
      rejectRefresh = reject;
    });
    mockState._recorders.RefreshTokens.returnValue(refreshDeferred);
    mockState._recorders.Login.resolve({
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      expiresAt: Date.now() + 3600_000,
    });

    const actions = defineAuthActions(mockState as any, mockHelpers as any, {
      isClient: true,
      clearAuthStorage: () => {},
    });

    const initPromise = actions.ensureAuthReady();
    await Promise.resolve();
    const loginPromise = actions.login('admin', 'secret');
    await Promise.resolve();

    rejectRefresh(new Error('token signature is invalid'));

    await initPromise;
    await loginPromise;

    expect(mockState._recorders.Login.calls.length).toBe(1);
    expect(mockHelpers._recorders.resetAuthState.calls.length).toBeGreaterThan(0);
    expect(mockState.initialized.value).toBe(true);
  } finally {
    console.warn = prevWarn;
  }
});
