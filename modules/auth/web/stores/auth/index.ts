// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineStore, PiniaPluginContext } from 'pinia';
import { defineAuthState } from './state';
import { defineAuthHelpers } from './helpers';
import { defineAuthActions } from './actions';
import { authStorage } from './storage';
import { AUTH_STORAGE_KEY, PERSIST_PATHS } from './options';

/**
 * Auth state store.
 */
export const useAuthStore = defineStore(
  'choysum.auth',
  () => {
    // Create the reactive state first.
    const state = defineAuthState();

    // Build helper methods on top of the state.
    const helpers = defineAuthHelpers(state);

    // Build actions after state and helpers are available.
    const actions = defineAuthActions(state, helpers);

    // Return the public store surface.
    return {
      // Core state.
      tokens: state.tokens,
      currentUser: state.currentUser,
      rememberMe: state.rememberMe,
      loading: state.loading,
      initialized: state.initialized,
      identity: state.identity,
      permissionState: state.permissionState,

      // Derived state.
      isAuthenticated: state.isAuthenticated,
      isAccessTokenValid: state.isAccessTokenValid,
      isRefreshTokenValid: state.isRefreshTokenValid,
      shouldRefreshToken: state.shouldRefreshToken,

      // Public actions.
      loadUser: actions.loadUser,
      loadPermissionState: actions.loadPermissionState,
      switchCompanyScope: actions.switchCompanyScope,
      login: actions.login,
      logout: actions.logout,
      refreshToken: actions.refreshToken,
      clearAuth: actions.clearAuth,
      initAuth: actions.initAuth,
      ensureAuthReady: actions.ensureAuthReady,
      register: actions.register,
      getCsrfToken: actions.getCsrfToken,
      persistLanguagePreference: actions.persistLanguagePreference,
    };
  },
  {
    // Pinia persistence configuration.
    persist: {
      storage: authStorage,
      key: AUTH_STORAGE_KEY,
      pick: PERSIST_PATHS,
      serializer: {
        serialize: JSON.stringify,
        deserialize: JSON.parse,
      },
      beforeHydrate: (_ctx: PiniaPluginContext) => {},
      afterHydrate: (_ctx: PiniaPluginContext) => {},
    },
  }
);
