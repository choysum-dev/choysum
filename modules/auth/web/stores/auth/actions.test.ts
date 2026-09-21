// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineAuthActions } from './actions';
import { hashPasswordClient } from './utils';

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
      RefreshTokens: RefreshTokens.fn,
      Browse: Browse.fn,
      UpdateById: UpdateById.fn,
      GetPermissionState: GetPermissionState.fn,
      SwitchCompanyScope: SwitchCompanyScope.fn,
    },
    _recorders: { Login, Logout, Register, RefreshTokens, Browse, UpdateById, GetPermissionState, SwitchCompanyScope },
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
