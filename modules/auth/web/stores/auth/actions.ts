// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isClient } from '@vueuse/core';
import { createTranslate } from '@/web/web/i18n';
import { hashPasswordClient, getCsrfTokenFromCookie, getDeviceInfo, withLoading } from './utils';
import { newAuthError, wrapAuthError, AuthErrCode, isAuthError, ChoysumError } from '../../error';
import type { AuthState } from './state';
import type { CompleteInitialSetupInput, InitialSetupOptions } from './setup_types';
import type { AuthHelpers } from './helpers';
import type { AuthOptions } from './options';
import { authStorage } from './storage';
import type { PermissionState } from '@/auth/web/permission';

const { _t } = createTranslate('auth', { scope: 'web/stores/auth/actions' });

/** Optional client/storage/language-store injection for FE unit tests. */
export type AuthActionDeps = {
  isClient?: boolean;
  clearAuthStorage?: () => void;
  /** When set, skips dynamic registry lookup for base.Language. */
  createLanguageStore?: () => {
    Browse: (id: string, fields?: string[]) => Promise<{ Code?: string } | null | undefined>;
    Search: (domain: unknown, opts?: unknown) => Promise<Array<{ Id?: string }>>;
  };
  /** When set, skips dynamic registry lookup for base.Company. */
  createCompanyStore?: () => {
    GetInitialSetupOptions: (req: Record<string, never>) => Promise<InitialSetupOptions>;
  };
  /** When set, skips dynamic import of `@/web/web/stores/i18nStore`. */
  importI18nStore?: () => Promise<{
    useI18nStore: () => {
      setUiKey: (uiKey: string) => Promise<unknown> | unknown;
      setDisplayOverrides: (overrides: unknown) => void;
    };
    langToUiKey: (terminologyLang: string) => string;
  }>;
};

/**
 * Build the auth store action set.
 */
export function defineAuthActions(state: AuthState, helpers: AuthHelpers, deps?: AuthActionDeps) {
  // Ensure auth initialization only runs once at a time.
  // Declared at the top of the function scope so it is visible to all
  // inner functions that reference it (loginImpl, ensureAuthReady, etc.).
  let initInFlight: Promise<void> | null = null;
  // Cached GetInitialSetupState result for route guards (null = not probed yet).
  let setupCompleteCache: boolean | null = null;
  const clientSide = deps?.isClient ?? isClient;
  const clearStoredAuth = deps?.clearAuthStorage ?? (() => authStorage.clearAuthStorage());

  async function getLanguageStore() {
    if (deps?.createLanguageStore) return deps.createLanguageStore();
    const { createStoreByModel } = await import('@/web/web/stores/registry');
    return createStoreByModel('base.Language') as {
      Browse: (id: string, fields?: string[]) => Promise<{ Code?: string } | null | undefined>;
      Search: (domain: unknown, opts?: unknown) => Promise<Array<{ Id?: string }>>;
    };
  }

  /**
   * Resolve the device info payload that should be sent with auth RPCs.
   */
  function getDefaultDeviceInfo(providedInfo?: string): string {
    if (providedInfo) return providedInfo;
    return state.authOptions.attachDeviceInfo ? getDeviceInfo() : '';
  }

  /**
   * Clear local auth state and any persisted auth storage.
   */
  function clearAuth(): void {
    // Reset in-memory auth state first.
    helpers.resetAuthState();

    // Re-probe setup phase after logout / credential wipe.
    setupCompleteCache = null;

    // Drop the cached permission snapshot.
    state.permissionState.value = null;

    // Clear Preferences.display format overrides (guest sessions use Language/catalog only).
    try {
      void import('@/web/web/stores/i18nStore')
        .then(({ useI18nStore }) => {
          useI18nStore().setDisplayOverrides(null);
        })
        .catch(() => {
          // Best-effort when Pinia/i18n is unavailable (FE unit host).
        });
    } catch {
      // Best-effort.
    }

    // Clear persisted auth storage in the browser.
    if (clientSide) {
      clearStoredAuth();
    }
  }

  /**
   * Load or refresh the backend permission snapshot.
   */
  async function loadPermissionStateImpl(forceRefresh = false): Promise<boolean> {
    // Capture user + company scope for this request. Routine RefreshTokens rotation
    // must not discard a still-valid permission result; SwitchCompanyScope must.
    const enabledScopeKey = (meta: Record<string, unknown>): string =>
      Array.isArray(meta.enabledCompanyIds)
        ? meta.enabledCompanyIds.map(x => String(x ?? '').trim()).filter(Boolean).slice().sort().join(',')
        : '';
    const requestUserId = String(state.identity.value?.userId || '').trim();
    const requestMeta = ((state.identity.value as any)?.metadata ?? {}) as Record<string, unknown>;
    const requestActiveCompanyId = String(requestMeta.activeCompanyId ?? '').trim();
    const requestEnabledKey = enabledScopeKey(requestMeta);
    const requestPermVersion =
      typeof requestMeta.permStateVersion === 'number' ? requestMeta.permStateVersion : null;
    // Snapshot present when this request started; failed/empty paths must not clear a newer install.
    const requestSnapshot = state.permissionState.value;
    const stillOwnsCompanyScope = () => {
      if (!requestUserId || !requestActiveCompanyId) return false;
      if (String(state.identity.value?.userId || '').trim() !== requestUserId) return false;
      const meta = ((state.identity.value as any)?.metadata ?? {}) as Record<string, unknown>;
      const active = String(meta.activeCompanyId ?? '').trim();
      return active === requestActiveCompanyId && enabledScopeKey(meta) === requestEnabledKey;
    };
    /** True when this request still matches the identity permission epoch. */
    const canApplyPermissionResult = () => {
      if (!stillOwnsCompanyScope()) return false;
      const meta = ((state.identity.value as any)?.metadata ?? {}) as Record<string, unknown>;
      const identityPermV =
        typeof meta.permStateVersion === 'number' ? meta.permStateVersion : null;
      // Identity moved to a newer permission epoch (refresh/switch) while we were in flight.
      if (
        requestPermVersion !== null
        && identityPermV !== null
        && identityPermV !== requestPermVersion
      ) {
        return false;
      }
      return true;
    };

    try {
      if (!requestUserId) return false;

      const meta = (state.identity.value as any)?.metadata as any;
      const expectedV = typeof meta?.permStateVersion === 'number' ? meta.permStateVersion : null;
      const currentV = state.permissionState.value?.permStateVersion ?? null;
      if (!forceRefresh && expectedV !== null && currentV === expectedV) return true;

      const resp = (await state.userStore.GetPermissionState()) as any as PermissionState;
      if (!canApplyPermissionResult()) return true;
      if (!resp || typeof (resp as any).byCompany !== 'object') {
        if (state.permissionState.value === requestSnapshot) {
          state.permissionState.value = { permStateVersion: 0, byCompany: {} };
        }
        return true;
      }
      // Prefer newer snapshots; never downgrade an already-installed higher version.
      const respV = typeof (resp as any).permStateVersion === 'number' ? (resp as any).permStateVersion : null;
      const snapshotV = state.permissionState.value?.permStateVersion ?? null;
      if (snapshotV !== null && respV !== null && respV < snapshotV) return true;
      state.permissionState.value = resp;
      return true;
    } catch (error) {
      // Only fail-closed when this request still owns the epoch and nobody replaced the snapshot.
      if (canApplyPermissionResult() && state.permissionState.value === requestSnapshot) {
        state.permissionState.value = null;
      }
      throw wrapAuthError(error, {
        code: AuthErrCode.USER_LOADING_FAILED,
        message: _t('Failed to load permission state'),
      });
    }
  }

  /**
   * Switch the active company scope and refresh dependent auth state.
   */
  async function switchCompanyScopeImpl(activeCompanyId: string, enabledCompanyIds?: string[] | null): Promise<boolean> {
    try {
      if (!state.tokens.value?.accessToken || !state.identity.value?.userId) {
        throw newAuthError({
          code: AuthErrCode.UNKNOWN,
          message: _t('Not authenticated; cannot switch company scope'),
        });
      }

      const resp = (await state.userStore.SwitchCompanyScope({
        ActiveCompanyId: activeCompanyId,
        EnabledCompanyIds: enabledCompanyIds ?? undefined,
      })) as any;
      if (!resp || !resp.accessToken) {
        throw newAuthError({
          code: AuthErrCode.UNKNOWN,
          message: _t('SwitchCompanyScope returned an invalid response'),
        });
      }

      state.tokens.value = resp;
      helpers.updateTokenIdentity();

      // Re-schedule refresh timers against the new token deadlines.
      if (state.authOptions.autoRefresh) {
        helpers.setupRefreshTimer(() => refreshToken(false));
      }

      // Force-refresh PermissionState because company scope changes can alter the snapshot.
      if (state.identity.value?.userId) {
        try {
          await loadPermissionState(true);
        } catch {
          // ignore
        }
      }

      return true;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.UNKNOWN,
        message: _t('Failed to switch company scope'),
      });
    }
  }

  /**
   * Resolve whether initial setup is complete (anonymous probe).
   * Only the terminal Ready result is cached; pending always re-probes so a
   * cross-tab CompleteInitialSetup is visible on the next navigation.
   */
  async function ensureSetupStateReady(): Promise<boolean> {
    if (setupCompleteCache === true) {
      return true;
    }
    try {
      const resp = (await state.userStore.GetInitialSetupState({})) as {
        Complete?: boolean;
        complete?: boolean;
      } | null;
      const complete = !!(resp && (resp.Complete ?? resp.complete));
      if (complete) {
        setupCompleteCache = true;
      }
      return complete;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.INITIALIZATION_FAILED,
        message: _t('Failed to load initial setup state'),
      });
    }
  }

  async function getCompanyStore() {
    if (deps?.createCompanyStore) return deps.createCompanyStore();
    const { createStoreByModel } = await import('@/web/web/stores/registry');
    return createStoreByModel('base.Company') as unknown as {
      GetInitialSetupOptions: (req: Record<string, never>) => Promise<InitialSetupOptions>;
    };
  }

  /**
   * Load setup-window choices (currencies, countries, timezones, languages, defaults).
   * Only callable while setup is pending; errors keep their server code and message.
   */
  async function getInitialSetupOptions(): Promise<InitialSetupOptions> {
    try {
      const companyStore = await getCompanyStore();
      const resp = await companyStore.GetInitialSetupOptions({});
      if (!resp || !resp.Defaults) {
        throw newAuthError({
          code: AuthErrCode.INITIALIZATION_FAILED,
          message: _t('GetInitialSetupOptions returned an invalid response'),
        });
      }
      return resp;
    } catch (error) {
      if (error instanceof ChoysumError) throw error;
      throw wrapAuthError(error, {
        code: AuthErrCode.INITIALIZATION_FAILED,
        message: _t('Failed to load initial setup options'),
      });
    }
  }

  /**
   * Complete initial setup (admin credentials, organization, setup.completed_at).
   * Does not create a session; callers Login afterward (same shape as Register).
   * Server errors are rethrown unchanged so callers can read `metadata.Field`.
   */
  async function completeInitialSetupImpl(req: CompleteInitialSetupInput): Promise<any> {
    try {
      const normalizedUsername = String(req.Admin.Username ?? '').trim();
      const hashedPassword = await hashPasswordClient(req.Admin.Password, normalizedUsername);
      const result = await state.userStore.CompleteInitialSetup({
        Admin: {
          ...req.Admin,
          Username: normalizedUsername,
          Password: hashedPassword,
        },
        Organization: { ...req.Organization },
      } as any);
      if (!result || typeof (result as any).UserId !== 'string' || (result as any).UserId === '') {
        throw newAuthError({
          code: AuthErrCode.UNKNOWN,
          message: _t('CompleteInitialSetup returned an invalid response'),
        });
      }
      setupCompleteCache = true;
      return result;
    } catch (error) {
      // Another client may have finished setup; force the next guard to re-probe.
      setupCompleteCache = null;
      if (error instanceof ChoysumError) throw error;
      throw wrapAuthError(error, {
        code: AuthErrCode.UNKNOWN,
        message: _t('Initial setup failed'),
      });
    }
  }

  /**
   * Register a new user through the auth service.
   */
  async function registerImpl(username: string, email: string, password: string, additionalData: Record<string, unknown> = {}): Promise<any> {
    try {
      // Salt must match the trimmed Username the backend stores.
      const normalizedUsername = username.trim();
      const hashedPassword = await hashPasswordClient(password, normalizedUsername);

      // Build the payload expected by the Register RPC.
      const userData: Record<string, unknown> = {
        ...additionalData,
        // Identity arguments stay authoritative over additionalData.
        Username: normalizedUsername,
        Email: email.trim(),
      };
      // Register.vue historically passed camelCase fullName; map to FirstName when present.
      if (typeof userData.fullName === 'string' && userData.fullName.trim() && !userData.FirstName) {
        userData.FirstName = String(userData.fullName).trim();
      }
      delete userData.fullName;

      // Forward the hashed password to the backend Register RPC (shape B: { UserId }).
      const result = await state.userStore.Register({
        User: userData as any,
        Password: hashedPassword,
      });
      if (!result || typeof (result as any).UserId !== 'string' || (result as any).UserId === '') {
        throw newAuthError({
          code: AuthErrCode.REGISTRATION_FAILED,
          message: _t('Register returned an invalid response'),
        });
      }
      return result;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.REGISTRATION_FAILED,
        message: _t('Registration failed'),
      });
    }
  }

  /**
   * Authenticate a user and initialize local auth state.
   */
  async function loginImpl(username: string, password: string, ipAddress = '', deviceInfo = '', shouldRemember = false): Promise<any> {
    try {
      // If a previous mount (e.g. Login.vue onMounted) has already started
      // initialization, wait for it to finish so it does not race with the
      // clearAuth below. Do not start a new initAuth here — any tokens it
      // would produce are immediately discarded by the unconditional clear.
      if (initInFlight) {
        try {
          await initInFlight;
        } catch {
          // initAuth clears stale state internally on failure; ignore.
        }
      }

      // Unconditionally clear any existing auth state before login so the
      // auth interceptor does not attempt to use or refresh old tokens
      // (which may be unexpired but invalid after a key rotation or DB reset)
      // during the Login RPC.
      clearAuth();

      // Same trim as Register so a padded identifier still matches the stored hash.
      const normalizedUsername = username.trim();
      const hashedPassword = await hashPasswordClient(password, normalizedUsername);

      // Resolve the device info payload that should accompany the login.
      const actualDeviceInfo = getDefaultDeviceInfo(deviceInfo);

      // Call the Login RPC with the hashed password.
      const response = await state.userStore.Login({
        UsernameOrEmail: normalizedUsername,
        Password: hashedPassword,
        IpAddress: ipAddress,
        DeviceInfo: actualDeviceInfo,
        RememberMe: shouldRemember,
      });

      if (!response || !response.accessToken) {
        throw newAuthError({
          code: AuthErrCode.INVALID_CREDENTIALS,
          message: _t('Invalid login response'),
        });
      }

      // Persist the returned token pair in local state.
      state.tokens.value = response;

      // Track the remember-me preference used for this login.
      state.rememberMe.value = shouldRemember;

      // Refresh token identity and fetch the user profile when possible.
      helpers.updateTokenIdentity();

      if (state.identity.value?.userId) {
        await loadUser(true);
      }

      // Arm automatic token refresh after a successful login.
      if (state.authOptions.autoRefresh) {
        helpers.setupRefreshTimer(() => refreshToken(false));
      }

      // Record the last auth refresh boundary for refresh throttling.
      state.refreshState.lastRefreshTime = Date.now();

      return response;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.INVALID_CREDENTIALS,
        message: _t('Login failed'),
      });
    }
  }

  /**
   * Log out locally and, when possible, invalidate server-side auth state.
   */
  async function logoutImpl(token = '', allDevices = false, deviceInfo = ''): Promise<boolean> {
    try {
      // Prefer the explicit token argument, then fall back to the in-memory access token.
      const actualToken = token || state.tokens.value?.accessToken || '';
      const actualDeviceInfo = getDefaultDeviceInfo(deviceInfo);

      // Ask the backend to revoke the current session when a token is available.
      if (actualToken) {
        try {
          await state.userStore.Logout({
            Token: actualToken,
            AllDevices: allDevices,
            DeviceInfo: actualDeviceInfo,
          });
        } catch (error) {
          clearAuth();
          throw wrapAuthError(error, {
            code: AuthErrCode.LOGOUT_FAILED,
            message: _t('Server-side logout failed, but local state was cleared'),
          });
        }
      }

      clearAuth();
      return true;
    } catch (error) {
      clearAuth();
      throw wrapAuthError(error, {
        code: AuthErrCode.LOGOUT_FAILED,
        message: _t('Logout failed, but local state was cleared'),
      });
    }
  }

  /**
   * Refresh result type with any nested Promise wrapper removed.
   */
  type RefreshResult = Awaited<ReturnType<typeof state.userStore.RefreshTokens>>;

  /**
   * In-flight refresh promise used to deduplicate concurrent refresh calls.
   */
  let refreshInFlight: Promise<RefreshResult> | null = null;

  /**
   * Refresh the current token pair and deduplicate overlapping refresh requests.
   */
  async function refreshTokenImpl(loadUserAfterRefresh: boolean = false): Promise<RefreshResult> {
    // Reuse the same promise while a refresh is already running.
    if (refreshInFlight) {
      return await refreshInFlight;
    }
    if (!state.tokens.value?.refreshToken) {
      throw newAuthError({ code: AuthErrCode.REFRESH_FAILED, message: _t('No refresh token is available') });
    }

    state.refreshState.refreshing = true;

    // Build a single refresh flow and share it across concurrent callers.
    refreshInFlight = (async (): Promise<RefreshResult> => {
      const refreshTokenUsed = state.tokens.value!.refreshToken;
      // Track the pair this refresh installs so our own write is not treated as
      // a superseding writer when a later step (identity / permission load) fails.
      let installedRefreshToken: string | null = null;
      const isSupersededByOther = () => {
        const current = state.tokens.value?.refreshToken;
        if (!current) return false;
        if (installedRefreshToken !== null) return current !== installedRefreshToken;
        return current !== refreshTokenUsed;
      };
      try {
        const prevIdentity = state.identity.value || null;

        const response = await state.userStore.RefreshTokens({
          RefreshToken: refreshTokenUsed,
        });

        if (!response || !response.accessToken) {
          if (isSupersededByOther() && state.tokens.value) {
            return state.tokens.value as RefreshResult;
          }
          throw newAuthError({ code: AuthErrCode.REFRESH_FAILED, message: _t('RefreshTokens returned an invalid response') });
        }

        // Discard late refresh results after another writer replaced the session.
        if (isSupersededByOther()) {
          return state.tokens.value as RefreshResult;
        }

        // Persist the refreshed token pair and identity.
        state.tokens.value = response;
        installedRefreshToken = String(response.refreshToken ?? '');
        helpers.updateTokenIdentity();
        state.refreshState.lastRefreshTime = Date.now();

        const nextIdentity = state.identity.value || null;

        // Detect profile or permission changes carried by the refreshed token metadata.
        const prevUserV = (prevIdentity as any)?.metadata?.userVersion;
        const nextUserV = (nextIdentity as any)?.metadata?.userVersion;
        const userChanged = typeof nextUserV === 'number' && nextUserV !== prevUserV;

        const prevPermV = (prevIdentity as any)?.metadata?.permStateVersion;
        const nextPermV = (nextIdentity as any)?.metadata?.permStateVersion;
        const permChanged = typeof nextPermV === 'number' && nextPermV !== prevPermV;

        if ((loadUserAfterRefresh || userChanged) && state.identity.value?.userId) {
          await loadUser(true);
        }
        if (permChanged && state.identity.value?.userId) {
          await loadPermissionState(true);
        }

        return response;
      } catch (error) {
        // Keep the current session only when another writer replaced our pair.
        if (isSupersededByOther() && state.tokens.value) {
          return state.tokens.value as RefreshResult;
        }
        // Drop local auth state on unrecoverable refresh failures that still own the session.
        clearAuth();
        throw wrapAuthError(error, {
          code: AuthErrCode.REFRESH_FAILED,
          message: _t('Failed to refresh token'),
        });
      } finally {
        state.refreshState.refreshing = false;
        refreshInFlight = null;
      }
    })();

    return await refreshInFlight;
  }

  /**
   * Load the current user profile into local auth state.
   */
  async function loadUserImpl(forceRefresh = false): Promise<boolean> {
    try {
      const userId = state.currentUser.value?.Id || state.identity.value?.userId || null;
      if (!userId) {
        throw newAuthError({
          code: AuthErrCode.USER_LOADING_FAILED,
          message: _t('Cannot load user details: missing userId'),
        });
      }

      if (!forceRefresh && state.currentUser.value) return true;

      // Fetch the current user profile from the backend store.
      const user = await state.userStore.Browse(userId, [
        'Id',
        'Username',
        'Email',
        'LanguageId',
        'Timezone',
        'Theme',
        'Preferences',
      ]);
      state.currentUser.value = user;
      // Align shell theme with User.Theme / Preferences.theme (localStorage as fallback).
      try {
        const { applyChoyThemePreference, readChoyThemePreference } = await import(
          '@/web/web/composables/applyChoyThemePreference'
        );
        const { resolveUserThemeMode, resolvePreferencesThemeMode } = await import(
          '@/auth/web/components/preferences/preferences_theme'
        );
        applyChoyThemePreference({
          theme: resolveUserThemeMode(user as any, resolvePreferencesThemeMode(readChoyThemePreference().theme)),
        });
      } catch {
        // Best-effort; auth must not fail because of theme wiring.
      }
      // Align FE UI key with the user's LanguageId (covers initAuth refresh paths; Login also applies this).
      try {
        const { useI18nStore, langToUiKey } = await (deps?.importI18nStore
          ? deps.importI18nStore()
          : import('@/web/web/stores/i18nStore'));
        const { applyUserLanguagePreference } = await import('./language_preference');
        const i18nStore = useI18nStore();
        let browseLanguage: (id: string, fields: string[]) => Promise<{ Code?: string; IsActive?: boolean } | null | undefined> =
          async () => null;
        try {
          const languageStore = await getLanguageStore();
          browseLanguage = (id, fields) => languageStore.Browse(id, fields);
        } catch {
          // Language registry unavailable; display overrides still apply below.
        }
        await applyUserLanguagePreference({
          languageId: (user as any)?.LanguageId,
          displayOverrides: (user as any)?.Preferences?.display ?? null,
          browseLanguage,
          setUiKey: key => i18nStore.setUiKey(key),
          setDisplayOverrides: overrides => i18nStore.setDisplayOverrides(overrides as any),
          langToUiKey,
        });
      } catch {
        // Best-effort; auth must not fail because of i18n wiring.
      }
      return true;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.USER_LOADING_FAILED,
        message: _t('Failed to load user details'),
      });
    }
  }

  /**
   * Initialize auth state from persisted tokens in the browser.
   */
  async function initAuth(options?: Partial<AuthOptions>): Promise<void> {
    // Merge any initialization-time option overrides.
    if (options) {
      Object.assign(state.authOptions, options);
    }

    // Initialization is only meaningful in the browser.
    if (!clientSide) return;

    try {
      // Without a refresh token, there is no persisted auth session to recover.
      if (!state.tokens.value?.refreshToken) {
        clearAuth();
        state.initialized.value = true;
        return;
      }

      // Recover identity and refresh token if needed. Failures here
      // (malformed tokens, key rotation, DB resets) are expected — clear
      // stale state and mark init complete without re-throwing.
      try {
        helpers.updateTokenIdentity();

        // Refresh when the current token is expired or near expiry.
        if (state.shouldRefreshToken.value) {
          await refreshToken(false);
        }
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        console.warn('[auth] Auth initialization recovery failed. Clearing stale auth state:', msg);
        clearAuth();
        state.initialized.value = true;
        return;
      }

      // Tokens alone do not carry a display name; hydrate the profile for shell chrome.
      // Failure here must not clear the session — labels fall back to a generic Account string.
      if (state.identity.value?.userId) {
        try {
          await loadUser(false);
        } catch (err) {
          const msg = err instanceof Error ? err.message : String(err);
          console.warn('[auth] Failed to load user profile during auth initialization:', msg);
        }
      }

      // Re-arm automatic refresh scheduling when enabled.
      if (state.authOptions.autoRefresh) {
        helpers.setupRefreshTimer(() => refreshToken(false));
      }

      // Mark initialization as complete on success.
      state.initialized.value = true;
    } catch (error) {
      clearAuth();
      state.initialized.value = true;
      throw wrapAuthError(error, {
        code: AuthErrCode.INITIALIZATION_FAILED,
        message: _t('Failed to initialize auth'),
      });
    }
  }

  /**
   * Read the CSRF token from cookies.
   */
  async function getCsrfToken(): Promise<string | null> {
    return getCsrfTokenFromCookie();
  }

  /**
   * Ensure auth initialization has completed before callers depend on auth state.
   */
  async function ensureAuthReady(options?: Partial<AuthOptions>): Promise<void> {
    if (state.initialized.value) return;
    if (!initInFlight) {
      initInFlight = (async () => {
        try {
          await initAuth(options);
        } finally {
          // Clear the in-flight marker regardless of the initialization outcome.
          initInFlight = null;
        }
      })();
    }
    await initInFlight;
  }

  /**
   * Persist terminology language preference for the logged-in user (User.LanguageId).
   * Anonymous callers no-op; FE still keeps locale in i18nStore localStorage.
   */
  async function persistLanguagePreference(lang: string): Promise<void> {
    const terminologyLang = String(lang || '').trim();
    if (!terminologyLang) {
      return;
    }
    if (!state.isAuthenticated.value) {
      return;
    }
    const userId = state.currentUser.value?.Id || state.identity.value?.userId || null;
    if (!userId) {
      return;
    }
    let languageId = '';
    try {
      const languageStore = await getLanguageStore();
      const rows = (await languageStore.Search(
        { And: [['Code', '=', terminologyLang], ['IsActive', '=', true]] } as any,
        { fields: ['Id'], limit: 1 } as any
      )) as Array<{ Id?: string }>;
      languageId = String(rows?.[0]?.Id || '').trim();
    } catch {
      // Best-effort preference write; FE locale can still live in i18nStore localStorage.
      return;
    }
    if (!languageId) return;
    await state.userStore.UpdateById(userId, { LanguageId: languageId } as any, ['Id', 'LanguageId'] as any);
    if (state.currentUser.value) {
      (state.currentUser.value as any).LanguageId = languageId;
    }
    // Sync JWT metadata.language after preference write.
    await refreshTokenImpl(true);
  }

  // Wrap public async actions with shared loading-state bookkeeping.
  const register = withLoading(registerImpl, state.loading);
  const completeInitialSetup = withLoading(completeInitialSetupImpl, state.loading);
  const login = withLoading(loginImpl, state.loading);
  const logout = withLoading(logoutImpl, state.loading);
  const refreshToken = refreshTokenImpl;
  const loadUser = withLoading(loadUserImpl, state.loading);
  const loadPermissionState = withLoading(loadPermissionStateImpl, state.loading);
  const switchCompanyScope = withLoading(switchCompanyScopeImpl, state.loading);

  return {
    clearAuth,
    login,
    logout,
    refreshToken,
    initAuth,
    ensureAuthReady,
    ensureSetupStateReady,
    loadUser,
    loadPermissionState,
    switchCompanyScope,
    register,
    completeInitialSetup,
    getInitialSetupOptions,
    getCsrfToken,
    persistLanguagePreference,
  };
}

export type AuthActions = ReturnType<typeof defineAuthActions>;
