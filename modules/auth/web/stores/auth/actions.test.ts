// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineAuthActions } from './actions';
import { hashPasswordClient } from './utils';
import { ChoysumError } from '../../error';

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
    reject(error: unknown) {
      impl = async () => {
        throw error;
      };
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
  const CompleteInitialSetup = makeFn();
  const GetInitialSetupState = makeFn();
  const RefreshTokens = makeFn();
  const Browse = makeFn();
  const UpdateById = makeFn();
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
      CompleteInitialSetup: CompleteInitialSetup.fn,
      GetInitialSetupState: GetInitialSetupState.fn,
      RefreshTokens: RefreshTokens.fn,
      Browse: Browse.fn,
      UpdateById: UpdateById.fn,
      GetPermissionState: GetPermissionState.fn,
      SwitchCompanyScope: SwitchCompanyScope.fn,
    },
    _recorders: {
      Login,
      Logout,
      Register,
      CompleteInitialSetup,
      GetInitialSetupState,
      RefreshTokens,
      Browse,
      UpdateById,
      GetPermissionState,
      SwitchCompanyScope,
    },
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

test('loginImpl: sends the trimmed identifier in the login envelope', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.Login.resolve({
    accessToken: 'new-access',
    refreshToken: 'new-refresh',
    expiresAt: Date.now() + 3600_000,
  });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    clearAuthStorage: () => {},
  });

  await actions.login('  admin  ', 'secret');

  const payload = mockState._recorders.Login.calls[0].args[0] as { UsernameOrEmail: string; Password: string };
  expect(payload.UsernameOrEmail).toBe('admin');
  expect(payload.Password).toBe(await hashPasswordClient('secret', 'admin'));
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

test('register: trims identity, keeps it authoritative, and maps fullName', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.Register.resolve({ UserId: 'usr_reg' });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  const result = await actions.register('  ada  ', '  ada@example.com  ', 'secret', {
    Username: 'spoof',
    Email: 'spoof@example.com',
    fullName: '  Ada Lovelace  ',
    CompanyId: 'cmp_spoof',
  });

  expect(result).toEqual({ UserId: 'usr_reg' });
  expect(mockState._recorders.Register.calls.length).toBe(1);
  const payload = mockState._recorders.Register.calls[0].args[0] as {
    User: Record<string, unknown>;
    Password: string;
  };
  expect(payload.User.Username).toBe('ada');
  expect(payload.User.Email).toBe('ada@example.com');
  expect(payload.User.FirstName).toBe('Ada Lovelace');
  expect(payload.User.fullName).toBe(undefined);
  expect(payload.User.CompanyId).toBe('cmp_spoof');
  expect(payload.Password).toBe(await hashPasswordClient('secret', 'ada'));
});

test('register: does not overwrite an explicit FirstName from fullName', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.Register.resolve({ UserId: 'usr_reg' });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  await actions.register('ada', 'ada@example.com', 'secret', {
    FirstName: 'Ann',
    fullName: 'Ada Lovelace',
  });

  const payload = mockState._recorders.Register.calls[0].args[0] as { User: Record<string, unknown> };
  expect(payload.User.FirstName).toBe('Ann');
  expect(payload.User.fullName).toBe(undefined);
});

test('register: rejects a response without a UserId string', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.Register.resolve({ UserId: '' });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  let caught: any;
  try {
    await actions.register('ada', 'ada@example.com', 'secret');
  } catch (err) {
    caught = err;
  }
  expect(String(caught?.code || '')).toBe('REGISTRATION_FAILED');
});

function setupInput() {
  return {
    Admin: { Username: '  admin  ', Password: 'secret1', LanguageCode: 'zh_CN', Email: 'owner@acme.io' },
    Organization: {
      Name: 'Acme',
      CountryCode: 'CN',
      CurrencyCode: 'CNY',
      Timezone: 'Asia/Shanghai',
    },
  };
}

test('completeInitialSetup: hashes password, sends sections, and marks setup complete', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.CompleteInitialSetup.resolve({ UserId: 'usr_admin' });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  const result = await actions.completeInitialSetup(setupInput());

  expect(result).toEqual({ UserId: 'usr_admin' });
  const payload = mockState._recorders.CompleteInitialSetup.calls[0].args[0] as any;
  expect(payload.Admin.Username).toBe('admin');
  expect(payload.Admin.Password).toBe(await hashPasswordClient('secret1', 'admin'));
  expect(payload.Admin.LanguageCode).toBe('zh_CN');
  expect(payload.Admin.Email).toBe('owner@acme.io');
  expect(payload.Organization).toEqual(setupInput().Organization);
  expect(await actions.ensureSetupStateReady()).toBe(true);
  expect(mockState._recorders.GetInitialSetupState.calls.length).toBe(0);
});

test('completeInitialSetup: rethrows server errors with Field metadata unchanged', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  const serverErr = new ChoysumError({ domain: 'auth', code: 'VALIDATION_FAILED', message: 'Username is already taken' }).withMetadata({
    Field: 'Admin.Username',
  });
  mockState._recorders.CompleteInitialSetup.reject(serverErr);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  let caught: any = null;
  try {
    await actions.completeInitialSetup(setupInput());
  } catch (err) {
    caught = err;
  }
  expect(caught).toBe(serverErr);
  expect(caught.metadata.Field).toBe('Admin.Username');
});

test('completeInitialSetup: wraps non-Choysum failures', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.CompleteInitialSetup.reject(new Error('rpc down'));

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  let caught: any = null;
  try {
    await actions.completeInitialSetup(setupInput());
  } catch (err) {
    caught = err;
  }
  expect(caught instanceof ChoysumError).toBe(true);
  expect(String(caught?.code || '')).toBe('UNKNOWN');
});

test('getInitialSetupOptions: reads base.Company options through the injected store', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  const options = {
    Countries: [{ Code: 'CN', Name: 'China', DefaultCurrencyCode: 'CNY' }],
    Currencies: [{ Code: 'CNY', Name: 'Chinese Yuan' }],
    Timezones: [{ Value: 'Asia/Shanghai', Label: 'Asia/Shanghai' }],
    Languages: [{ Code: 'zh_CN', Name: '简体中文' }],
    Defaults: {
      OrganizationName: 'My Company',
      CountryCode: 'CN',
      CurrencyCode: 'CNY',
      Timezone: 'Asia/Shanghai',
    },
  };
  const requests: unknown[] = [];
  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    createCompanyStore: () => ({
      GetInitialSetupOptions: async (req: Record<string, never>) => {
        requests.push(req);
        return options;
      },
    }),
  });
  expect(await actions.getInitialSetupOptions()).toEqual(options);
  expect(requests).toEqual([{}]);
});

test('getInitialSetupOptions: keeps FailedPrecondition from the setup gate', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  const gateErr = new ChoysumError({ domain: 'setupgate', code: 'FAILED_PRECONDITION', message: 'initial setup is not pending' });
  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    createCompanyStore: () => ({
      GetInitialSetupOptions: async () => {
        throw gateErr;
      },
    }),
  });
  let caught: unknown = null;
  try {
    await actions.getInitialSetupOptions();
  } catch (err) {
    caught = err;
  }
  expect(caught).toBe(gateErr);
});

test('getInitialSetupOptions: falls back to the store registry for base.Company', async () => {
  // FE unit runs resolve product imports of the store registry to a stub whose stores
  // have no RPC verbs, so the default path surfaces as a wrapped initialization error.
  const actions = defineAuthActions(buildMockState() as any, buildMockHelpers() as any, { isClient: true });
  let caught: ChoysumError | null = null;
  try {
    await actions.getInitialSetupOptions();
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught instanceof ChoysumError).toBe(true);
  expect(String(caught?.code || '')).toBe('INITIALIZATION_FAILED');
});

async function setupOptionsError(GetInitialSetupOptions: () => Promise<unknown>): Promise<ChoysumError | null> {
  const actions = defineAuthActions(buildMockState() as any, buildMockHelpers() as any, {
    isClient: true,
    createCompanyStore: () => ({ GetInitialSetupOptions }) as any,
  });
  try {
    await actions.getInitialSetupOptions();
  } catch (err) {
    return err as ChoysumError;
  }
  return null;
}

test('getInitialSetupOptions: rejects a response without Defaults', async () => {
  const err = await setupOptionsError(async () => ({ Countries: [] }));
  expect(err instanceof ChoysumError).toBe(true);
  expect(String(err?.code || '')).toBe('INITIALIZATION_FAILED');
});

test('getInitialSetupOptions: wraps non-Choysum failures', async () => {
  const err = await setupOptionsError(async () => {
    throw new Error('network down');
  });
  expect(err instanceof ChoysumError).toBe(true);
  expect(String(err?.code || '')).toBe('INITIALIZATION_FAILED');
});

test('ensureSetupStateReady: does not cache a pending probe result', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.GetInitialSetupState.resolve({ Complete: false });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  expect(await actions.ensureSetupStateReady()).toBe(false);
  expect(await actions.ensureSetupStateReady()).toBe(false);
  expect(mockState._recorders.GetInitialSetupState.calls.length).toBe(2);
});

test('ensureSetupStateReady: caches Ready and accepts lowercase complete', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.GetInitialSetupState.resolve({ complete: true });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  expect(await actions.ensureSetupStateReady()).toBe(true);
  expect(await actions.ensureSetupStateReady()).toBe(true);
  expect(mockState._recorders.GetInitialSetupState.calls.length).toBe(1);

  const failing = buildMockState();
  failing._recorders.GetInitialSetupState.reject(new Error('probe down'));
  const failingActions = defineAuthActions(failing as any, buildMockHelpers() as any, { isClient: true });
  let caught: any;
  try {
    await failingActions.ensureSetupStateReady();
  } catch (err) {
    caught = err;
  }
  expect(String(caught?.code || '')).toBe('INITIALIZATION_FAILED');
});

test('ensureSetupStateReady: clearAuth invalidates the cached Ready probe', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.GetInitialSetupState.resolve({ Complete: true });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    clearAuthStorage: () => {},
  });
  expect(await actions.ensureSetupStateReady()).toBe(true);
  expect(mockState._recorders.GetInitialSetupState.calls.length).toBe(1);

  actions.clearAuth();
  mockState._recorders.GetInitialSetupState.resolve({ Complete: false });
  expect(await actions.ensureSetupStateReady()).toBe(false);
  expect(mockState._recorders.GetInitialSetupState.calls.length).toBe(2);
});

test('completeInitialSetup: rejects an invalid response and clears setup cache', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.GetInitialSetupState.resolve({ Complete: true });
  mockState._recorders.CompleteInitialSetup.resolve({ UserId: '' });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  expect(await actions.ensureSetupStateReady()).toBe(true);
  expect(mockState._recorders.GetInitialSetupState.calls.length).toBe(1);

  let caught: any;
  try {
    await actions.completeInitialSetup(setupInput());
  } catch (err) {
    caught = err;
  }
  expect(String(caught?.code || '')).toBe('UNKNOWN');

  mockState._recorders.GetInitialSetupState.resolve({ Complete: false });
  expect(await actions.ensureSetupStateReady()).toBe(false);
  expect(mockState._recorders.GetInitialSetupState.calls.length).toBe(2);
});

test('register: wraps a Register RPC failure', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.Register.reject(new Error('rpc down'));

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  let caught: any;
  try {
    await actions.register('ada', 'ada@example.com', 'secret');
  } catch (err) {
    caught = err;
  }
  expect(String(caught?.code || '')).toBe('REGISTRATION_FAILED');
});

test('switchCompanyScope: sends the company scope envelope', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = { accessToken: 'access', refreshToken: 'refresh', expiresAt: Date.now() + 3600_000 };
  mockState.identity.value = { userId: 'usr_1' } as any;
  mockState._recorders.SwitchCompanyScope.resolve({
    accessToken: 'next-access',
    refreshToken: 'next-refresh',
    expiresAt: Date.now() + 3600_000,
  });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  const ok = await actions.switchCompanyScope('cmp_main', null);
  expect(ok).toBe(true);
  expect(mockState._recorders.SwitchCompanyScope.calls[0].args[0]).toEqual({
    ActiveCompanyId: 'cmp_main',
    EnabledCompanyIds: undefined,
  });
});

test('refreshToken: discards a late RefreshTokens result after SwitchCompanyScope', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'access',
    refreshToken: 'refresh-before',
    expiresAt: Date.now() + 3600_000,
  };
  mockState.identity.value = { userId: 'usr_1', metadata: { activeCompanyId: 'cmp_a' } } as any;

  let resolveRefresh: (value: unknown) => void = () => {};
  const refreshDeferred = new Promise(resolve => {
    resolveRefresh = resolve;
  });
  mockState._recorders.RefreshTokens.returnValue(refreshDeferred);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  const refreshPromise = actions.refreshToken(false);

  // Company switch replaces the token pair while refresh is still in flight.
  mockState.tokens.value = {
    accessToken: 'switched-access',
    refreshToken: 'switched-refresh',
    expiresAt: Date.now() + 3600_000,
  };
  mockState.identity.value = { userId: 'usr_1', metadata: { activeCompanyId: 'cmp_b' } } as any;

  resolveRefresh({
    accessToken: 'stale-access',
    refreshToken: 'stale-refresh',
    expiresAt: Date.now() + 3600_000,
  });

  const result = await refreshPromise;
  expect(result.refreshToken).toBe('switched-refresh');
  expect(mockState.tokens.value?.refreshToken).toBe('switched-refresh');
  expect(mockHelpers._recorders.updateTokenIdentity.calls.length).toBe(0);
});

test('refreshToken: superseded refresh failure keeps the switched session', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'access',
    refreshToken: 'refresh-before',
    expiresAt: Date.now() + 3600_000,
  };

  let rejectRefresh: (reason: unknown) => void = () => {};
  const refreshDeferred = new Promise((_resolve, reject) => {
    rejectRefresh = reject;
  });
  mockState._recorders.RefreshTokens.returnValue(refreshDeferred);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    clearAuthStorage: () => {},
  });
  const refreshPromise = actions.refreshToken(false);

  mockState.tokens.value = {
    accessToken: 'switched-access',
    refreshToken: 'switched-refresh',
    expiresAt: Date.now() + 3600_000,
  };

  rejectRefresh(new Error('refresh aborted'));

  const result = await refreshPromise;
  expect(result.refreshToken).toBe('switched-refresh');
  expect(mockState.tokens.value?.accessToken).toBe('switched-access');
  expect(mockHelpers._recorders.resetAuthState.calls.length).toBe(0);
});

test('refreshToken: superseded invalid RefreshTokens response keeps the switched session', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'access',
    refreshToken: 'refresh-before',
    expiresAt: Date.now() + 3600_000,
  };

  let resolveRefresh: (value: unknown) => void = () => {};
  const refreshDeferred = new Promise(resolve => {
    resolveRefresh = resolve;
  });
  mockState._recorders.RefreshTokens.returnValue(refreshDeferred);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    clearAuthStorage: () => {},
  });
  const refreshPromise = actions.refreshToken(false);

  mockState.tokens.value = {
    accessToken: 'switched-access',
    refreshToken: 'switched-refresh',
    expiresAt: Date.now() + 3600_000,
  };

  resolveRefresh({ accessToken: '', refreshToken: 'ignored', expiresAt: Date.now() + 3600_000 });

  const result = await refreshPromise;
  expect(result.refreshToken).toBe('switched-refresh');
  expect(mockHelpers._recorders.resetAuthState.calls.length).toBe(0);
});

test('loadPermissionState: late failure after company switch keeps the newer snapshot', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'access',
    refreshToken: 'refresh-before',
    expiresAt: Date.now() + 3600_000,
  };
  mockState.identity.value = {
    userId: 'usr_1',
    metadata: {
      permStateVersion: 1,
      activeCompanyId: 'cmp_a',
      enabledCompanyIds: ['cmp_a'],
    },
  } as any;
  mockState.permissionState.value = {
    permStateVersion: 1,
    byCompany: { cmp_a: {} },
  } as any;

  let rejectPermission: (reason: unknown) => void = () => {};
  const permissionDeferred = new Promise((_resolve, reject) => {
    rejectPermission = reject;
  });
  mockState._recorders.GetPermissionState.returnValue(permissionDeferred);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  const loadPromise = actions.loadPermissionState(true);

  // Company switch changes scope while the older permission load is in flight.
  mockState.tokens.value = {
    accessToken: 'switched-access',
    refreshToken: 'switched-refresh',
    expiresAt: Date.now() + 3600_000,
  };
  mockState.identity.value = {
    userId: 'usr_1',
    metadata: {
      permStateVersion: 2,
      activeCompanyId: 'cmp_b',
      enabledCompanyIds: ['cmp_b'],
    },
  } as any;
  mockState.permissionState.value = {
    permStateVersion: 2,
    byCompany: { cmp_b: {} },
  } as any;

  rejectPermission(new Error('stale permission load'));

  let caught: unknown;
  try {
    await loadPromise;
  } catch (err) {
    caught = err;
  }
  expect(Boolean(caught)).toBe(true);
  expect((mockState.permissionState.value as any)?.permStateVersion).toBe(2);
});

test('loadPermissionState: same-scope failure still clears the permission snapshot', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'access',
    refreshToken: 'refresh',
    expiresAt: Date.now() + 3600_000,
  };
  mockState.identity.value = {
    userId: 'usr_1',
    metadata: {
      permStateVersion: 1,
      activeCompanyId: 'cmp_a',
      enabledCompanyIds: ['cmp_a'],
    },
  } as any;
  mockState.permissionState.value = {
    permStateVersion: 1,
    byCompany: { cmp_a: {} },
  } as any;
  mockState._recorders.GetPermissionState.reject(new Error('permission down'));

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  let caught: unknown;
  try {
    await actions.loadPermissionState(true);
  } catch (err) {
    caught = err;
  }
  expect(Boolean(caught)).toBe(true);
  expect(mockState.permissionState.value).toBe(null);
});

test('loadPermissionState: older same-scope response does not clobber a newer permission version', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'access',
    refreshToken: 'refresh',
    expiresAt: Date.now() + 3600_000,
  };
  mockState.identity.value = {
    userId: 'usr_1',
    metadata: {
      permStateVersion: 1,
      activeCompanyId: 'cmp_a',
      enabledCompanyIds: ['cmp_a'],
    },
  } as any;

  let resolvePermission: (value: unknown) => void = () => {};
  const permissionDeferred = new Promise(resolve => {
    resolvePermission = resolve;
  });
  mockState._recorders.GetPermissionState.returnValue(permissionDeferred);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  const loadPromise = actions.loadPermissionState(true);

  // Newer permission epoch lands for the same company scope while v1 is in flight.
  mockState.identity.value = {
    userId: 'usr_1',
    metadata: {
      permStateVersion: 2,
      activeCompanyId: 'cmp_a',
      enabledCompanyIds: ['cmp_a'],
    },
  } as any;
  mockState.permissionState.value = {
    permStateVersion: 2,
    byCompany: { cmp_a: { canRead: true } },
  } as any;

  resolvePermission({
    permStateVersion: 1,
    byCompany: { cmp_a: { canRead: false } },
  });

  expect(await loadPromise).toBe(true);
  expect((mockState.permissionState.value as any)?.permStateVersion).toBe(2);
  expect((mockState.permissionState.value as any)?.byCompany?.cmp_a?.canRead).toBe(true);
});

test('loadPermissionState: routine refresh-token rotation still applies the result', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'access',
    refreshToken: 'refresh-before',
    expiresAt: Date.now() + 3600_000,
  };
  mockState.identity.value = {
    userId: 'usr_1',
    metadata: {
      permStateVersion: 1,
      activeCompanyId: 'cmp_a',
      enabledCompanyIds: ['cmp_a'],
    },
  } as any;

  let resolvePermission: (value: unknown) => void = () => {};
  const permissionDeferred = new Promise(resolve => {
    resolvePermission = resolve;
  });
  mockState._recorders.GetPermissionState.returnValue(permissionDeferred);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  const loadPromise = actions.loadPermissionState(true);

  // Token rotation without company-scope change must not discard the result.
  mockState.tokens.value = {
    accessToken: 'rotated-access',
    refreshToken: 'rotated-refresh',
    expiresAt: Date.now() + 3600_000,
  };

  resolvePermission({
    permStateVersion: 1,
    byCompany: { cmp_a: { canRead: true } },
  });

  expect(await loadPromise).toBe(true);
  expect((mockState.permissionState.value as any)?.byCompany?.cmp_a?.canRead).toBe(true);
});

test('refreshToken: failure after this refresh installs tokens still clears auth', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'access',
    refreshToken: 'refresh-before',
    expiresAt: Date.now() + 3600_000,
  };
  mockState.identity.value = {
    userId: 'usr_1',
    metadata: { userVersion: 1, permStateVersion: 1 },
  } as any;
  mockState._recorders.RefreshTokens.resolve({
    accessToken: 'refreshed-access',
    refreshToken: 'refreshed-refresh',
    expiresAt: Date.now() + 3600_000,
  });
  mockHelpers.updateTokenIdentity = () => {
    mockState.identity.value = {
      userId: 'usr_1',
      metadata: { userVersion: 2, permStateVersion: 1 },
    } as any;
    throw new Error('identity boom');
  };

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    clearAuthStorage: () => {},
  });

  let caught: unknown;
  try {
    await actions.refreshToken(false);
  } catch (err) {
    caught = err;
  }
  expect(Boolean(caught)).toBe(true);
  expect(mockHelpers._recorders.resetAuthState.calls.length).toBeGreaterThan(0);
});

test('logout: sends the logout envelope', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState._recorders.Logout.resolve(true);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  const ok = await actions.logout('access-token', true, 'device-1');
  expect(ok).toBe(true);
  expect(mockState._recorders.Logout.calls[0].args[0]).toEqual({
    Token: 'access-token',
    AllDevices: true,
    DeviceInfo: 'device-1',
  });
});

test('loadUser: applies LanguageId via injected language store', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.identity.value = { userId: 'usr_1' } as any;
  mockState._recorders.Browse.resolve({
    Id: 'usr_1',
    Username: 'ada',
    Email: 'ada@example.com',
    LanguageId: 'lang_zh',
    Preferences: { display: { dateFormat: 'YYYY-MM-DD' } },
  });
  const langBrowse = makeFn();
  langBrowse.resolve({ Code: 'zh_CN' });
  const setUiKey = makeFn();
  setUiKey.resolve(undefined);
  const setDisplayOverrides = makeFn();
  setDisplayOverrides.returnValue(undefined);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    createLanguageStore: () => ({
      Browse: langBrowse.fn,
      Search: async () => [],
    }),
    importI18nStore: async () => ({
      useI18nStore: () => ({
        setUiKey: setUiKey.fn,
        setDisplayOverrides: setDisplayOverrides.fn,
      }),
      langToUiKey: (lang: string) => `ui:${lang}`,
    }),
  });

  const ok = await actions.loadUser(true);
  expect(ok).toBe(true);
  expect(mockState._recorders.Browse.calls[0].args[1]).toEqual([
    'Id',
    'Username',
    'Email',
    'LanguageId',
    'Timezone',
    'Preferences',
  ]);
  expect((mockState.currentUser.value as any)?.LanguageId).toBe('lang_zh');
  expect(langBrowse.calls.length).toBe(1);
  expect(setUiKey.calls[0].args[0]).toBe('ui:zh_CN');
  expect(setDisplayOverrides.calls[0].args[0]).toEqual({ dateFormat: 'YYYY-MM-DD' });
});

test('loadUser: swallows i18n apply failures', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.identity.value = { userId: 'usr_1' } as any;
  mockState._recorders.Browse.resolve({ Id: 'usr_1', LanguageId: 'lang_zh' });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    createLanguageStore: () => ({
      Browse: async () => {
        throw new Error('browse down');
      },
      Search: async () => [],
    }),
    importI18nStore: async () => ({
      useI18nStore: () => ({
        setUiKey: async () => undefined,
        setDisplayOverrides: () => undefined,
      }),
      langToUiKey: (lang: string) => lang,
    }),
  });

  const ok = await actions.loadUser(true);
  expect(ok).toBe(true);
  expect((mockState.currentUser.value as any)?.Id).toBe('usr_1');
});

test('loadUser: still applies display overrides when the language store is missing', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.identity.value = { userId: 'usr_1' } as any;
  mockState._recorders.Browse.resolve({
    Id: 'usr_1',
    LanguageId: 'lang_zh',
    Preferences: { display: { dateFormat: 'YYYY-MM-DD' } },
  });
  const setDisplayOverrides = makeFn();
  setDisplayOverrides.returnValue(undefined);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    createLanguageStore: () => {
      throw new Error('registry down');
    },
    importI18nStore: async () => ({
      useI18nStore: () => ({
        setUiKey: async () => undefined,
        setDisplayOverrides: setDisplayOverrides.fn,
      }),
      langToUiKey: (lang: string) => lang,
    }),
  });

  const ok = await actions.loadUser(true);
  expect(ok).toBe(true);
  expect(setDisplayOverrides.calls[0].args[0]).toEqual({ dateFormat: 'YYYY-MM-DD' });
});

test('loadUser: exercises default i18nStore and Language registry imports', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.identity.value = { userId: 'usr_1' } as any;
  mockState._recorders.Browse.resolve({
    Id: 'usr_1',
    Username: 'ada',
    LanguageId: 'lang_zh',
    Preferences: { display: null },
  });

  // No createLanguageStore / importI18nStore — hit the production dynamic-import branches.
  const actions = defineAuthActions(mockState as any, mockHelpers as any, { isClient: true });
  const ok = await actions.loadUser(true);
  expect(ok).toBe(true);
  expect((mockState.currentUser.value as any)?.Id).toBe('usr_1');
});

test('persistLanguagePreference: writes LanguageId and refreshes tokens', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.isAuthenticated.value = true;
  mockState.currentUser.value = { Id: 'usr_1', LanguageId: null } as any;
  mockState.tokens.value = { accessToken: 'a', refreshToken: 'r', expiresAt: Date.now() + 3600_000 };
  mockState._recorders.UpdateById.resolve({ Id: 'usr_1', LanguageId: 'lang_en' });
  mockState._recorders.RefreshTokens.resolve({
    accessToken: 'a2',
    refreshToken: 'r2',
    expiresAt: Date.now() + 3600_000,
  });
  const langSearch = makeFn();
  langSearch.resolve([{ Id: 'lang_en' }]);

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    createLanguageStore: () => ({
      Browse: async () => null,
      Search: langSearch.fn,
    }),
  });

  await actions.persistLanguagePreference('en_US');
  expect(langSearch.calls.length).toBe(1);
  expect(mockState._recorders.UpdateById.calls[0].args[1]).toEqual({ LanguageId: 'lang_en' });
  expect((mockState.currentUser.value as any).LanguageId).toBe('lang_en');
  expect(mockState._recorders.RefreshTokens.calls.length).toBe(1);
});

test('persistLanguagePreference: no-ops when anonymous or language missing', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    createLanguageStore: () => ({
      Browse: async () => null,
      Search: async () => {
        throw new Error('should not search');
      },
    }),
  });
  await actions.persistLanguagePreference('en_US');
  expect(mockState._recorders.UpdateById.calls.length).toBe(0);

  mockState.isAuthenticated.value = true;
  mockState.currentUser.value = { Id: 'usr_1' } as any;
  const actions2 = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    createLanguageStore: () => ({
      Browse: async () => null,
      Search: async () => [],
    }),
  });
  await actions2.persistLanguagePreference('en_US');
  expect(mockState._recorders.UpdateById.calls.length).toBe(0);
});

test('persistLanguagePreference: no-ops when language lookup throws', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.isAuthenticated.value = true;
  mockState.currentUser.value = { Id: 'usr_1' } as any;
  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    createLanguageStore: () => ({
      Browse: async () => null,
      Search: async () => {
        throw new Error('registry down');
      },
    }),
  });
  await actions.persistLanguagePreference('en_US');
  expect(mockState._recorders.UpdateById.calls.length).toBe(0);
});

test('initAuth: hydrates currentUser from Browse after recovering persisted tokens', async () => {
  const mockState = buildMockState();
  const mockHelpers = buildMockHelpers();
  mockState.tokens.value = {
    accessToken: 'access',
    refreshToken: 'refresh',
    expiresAt: Date.now() + 3600_000,
  };
  mockState.shouldRefreshToken.value = false;
  mockHelpers.updateTokenIdentity = () => {
    mockState.identity.value = { userId: 'usr_1', tokenId: 'tok_1' } as any;
    return true;
  };
  mockState._recorders.Browse.resolve({
    Id: 'usr_1',
    Username: 'admin',
    Email: 'admin@example.com',
  });

  const actions = defineAuthActions(mockState as any, mockHelpers as any, {
    isClient: true,
    createLanguageStore: () => ({
      Browse: async () => null,
      Search: async () => [],
    }),
    importI18nStore: async () => ({
      useI18nStore: () => ({
        setUiKey: async () => undefined,
        setDisplayOverrides: () => undefined,
      }),
      langToUiKey: (lang: string) => lang,
    }),
  });

  await actions.ensureAuthReady();

  expect(mockState.initialized.value).toBe(true);
  expect(mockState._recorders.Browse.calls.length).toBe(1);
  expect((mockState.currentUser.value as any)?.Username).toBe('admin');
});

test('initAuth: keeps session when profile Browse fails', async () => {
  const prevWarn = console.warn;
  console.warn = () => {};
  try {
    const mockState = buildMockState();
    const mockHelpers = buildMockHelpers();
    mockState.tokens.value = {
      accessToken: 'access',
      refreshToken: 'refresh',
      expiresAt: Date.now() + 3600_000,
    };
    mockState.shouldRefreshToken.value = false;
    mockHelpers.updateTokenIdentity = () => {
      mockState.identity.value = { userId: 'usr_1', tokenId: 'tok_1' } as any;
      return true;
    };
    mockState._recorders.Browse.reject(new Error('browse down'));

    const actions = defineAuthActions(mockState as any, mockHelpers as any, {
      isClient: true,
      clearAuthStorage: () => {},
    });

    await actions.ensureAuthReady();

    expect(mockState.initialized.value).toBe(true);
    expect(mockState.tokens.value?.refreshToken).toBe('refresh');
    expect(mockState.currentUser.value).toBeNull();
    expect(mockHelpers._recorders.resetAuthState.calls.length).toBe(0);
  } finally {
    console.warn = prevWarn;
  }
});
