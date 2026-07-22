// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isClient } from '@vueuse/core';
import { createTranslate } from '@/web/web/i18n';
import { hashPasswordClient, getCsrfTokenFromCookie, getDeviceInfo, withLoading } from './utils';
import { newAuthError, wrapAuthError, AuthErrCode, isAuthError } from '../../error';
import type { AuthState } from './state';
import type { AuthHelpers } from './helpers';
import type { AuthOptions } from './options';
import { authStorage } from './storage';
import type { PermissionState } from '@/auth/web/permission';

const { _t } = createTranslate('auth', { scope: 'web/stores/auth/actions' });

/**
 * Build the auth store action set.
 */
export function defineAuthActions(state: AuthState, helpers: AuthHelpers) {
  // Ensure auth initialization only runs once at a time.
  // Declared at the top of the function scope so it is visible to all
  // inner functions that reference it (loginImpl, ensureAuthReady, etc.).
  let initInFlight: Promise<void> | null = null;

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

    // Drop the cached permission snapshot.
    state.permissionState.value = null;

    // Clear Preferences.display format overrides (guest sessions use Language/catalog only).
    try {
      void import('@/web/web/stores/i18nStore').then(({ useI18nStore }) => {
        useI18nStore().setDisplayOverrides(null);
      });
    } catch {
      // Best-effort.
    }

    // Clear persisted auth storage in the browser.
    if (isClient) {
      authStorage.clearAuthStorage();
    }
  }

  /**
   * Load or refresh the backend permission snapshot.
   */
  async function loadPermissionStateImpl(forceRefresh = false): Promise<boolean> {
    try {
      if (!state.identity.value?.userId) return false;

      const meta = (state.identity.value as any)?.metadata as any;
      const expectedV = typeof meta?.permStateVersion === 'number' ? meta.permStateVersion : null;
      const currentV = state.permissionState.value?.permStateVersion ?? null;
      if (!forceRefresh && expectedV !== null && currentV === expectedV) return true;

      const resp = (await state.userStore.GetPermissionState()) as any as PermissionState;
      if (!resp || typeof (resp as any).byCompany !== 'object') {
        state.permissionState.value = { permStateVersion: 0, byCompany: {} };
        return true;
      }
      state.permissionState.value = resp;
      return true;
    } catch (error) {
      // Keep fail-closed semantics when permission loading fails.
      state.permissionState.value = null;
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

      const resp = (await state.userStore.SwitchCompanyScope(activeCompanyId, enabledCompanyIds)) as any;
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
   * Register a new user through the auth service.
   */
  async function registerImpl(username: string, email: string, password: string, additionalData: Record<string, unknown> = {}): Promise<any> {
    try {
      // Hash the password client-side when the feature is enabled.
      const hashedPassword = await hashPasswordClient(password, username);

      // Build the payload expected by the Register RPC.
      const userData = {
        Username: username,
        Email: email,
        ...additionalData,
      };

      // Forward the hashed password to the backend Register RPC.
      const result = await state.userStore.Register(userData, hashedPassword);
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

      // Hash the password client-side when the feature is enabled.
      const hashedPassword = await hashPasswordClient(password, username);

      // Resolve the device info payload that should accompany the login.
      const actualDeviceInfo = getDefaultDeviceInfo(deviceInfo);

      // Call the Login RPC with the hashed password.
      const response = await state.userStore.Login(username, hashedPassword, ipAddress, actualDeviceInfo, shouldRemember);

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
          await state.userStore.Logout(actualToken, allDevices, actualDeviceInfo);
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
      try {
        const prevIdentity = state.identity.value || null;

        const response = await state.userStore.RefreshTokens(state.tokens.value!.refreshToken);
        if (!response || !response.accessToken) {
          throw newAuthError({ code: AuthErrCode.REFRESH_FAILED, message: _t('RefreshTokens returned an invalid response') });
        }

        // Persist the refreshed token pair and identity.
        state.tokens.value = response;
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
        // Drop local auth state on unrecoverable refresh failures.
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
      const user = await state.userStore.Browse(userId, ['Id', 'Username', 'Email', 'Language', 'Timezone', 'Preferences']);
      state.currentUser.value = user;
      // Align FE UI key with User.Language (covers initAuth refresh paths; Login also applies this).
      try {
        const { useI18nStore, langToUiKey } = await import('@/web/web/stores/i18nStore');
        const i18nStore = useI18nStore();
        const preferredLang = String((user as any)?.Language || '').trim();
        if (preferredLang) {
          await i18nStore.setUiKey(langToUiKey(preferredLang));
        }
        i18nStore.setDisplayOverrides((user as any)?.Preferences?.display ?? null);
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
    if (!isClient) return;

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
   * Persist terminology language preference for the logged-in user (User.Language).
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
    await state.userStore.UpdateById(userId, { Language: terminologyLang } as any, ['Id', 'Language'] as any);
    if (state.currentUser.value) {
      (state.currentUser.value as any).Language = terminologyLang;
    }
    // Sync JWT metadata.language after preference write.
    await refreshTokenImpl(true);
  }

  // Wrap public async actions with shared loading-state bookkeeping.
  const register = withLoading(registerImpl, state.loading);
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
    loadUser,
    loadPermissionState,
    switchCompanyScope,
    register,
    getCsrfToken,
    persistLanguagePreference,
  };
}

export type AuthActions = ReturnType<typeof defineAuthActions>;
