// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, computed, reactive, ComputedRef } from 'vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import type User from '@/auth/service/models/user';
import type { ClientModel } from '@/core/rpc';
import { AuthOptions, DEFAULT_AUTH_OPTIONS, createAuthOptions } from './options';
import type { PermissionState } from '@/auth/web/permission';

let globalUserStore: any | null = null;

/**
 * Lazily create the shared auth.User store instance.
 */
function getGlobalUserStore(): any {
  if (globalUserStore) return globalUserStore;
  globalUserStore = createStoreByModel<typeof User>('auth.User');
  return globalUserStore;
}

/**
 * Create the reactive auth state container used by the auth store.
 */
export function defineAuthState(options?: Partial<AuthOptions>) {
  const userStore = getGlobalUserStore();

  // Core state.

  /**
   * Current access and refresh token pair.
   */
  const tokens = ref<TokenPair | null>(null);

  /**
   * Current authenticated user details.
   */
  const currentUser = ref<ClientModel<User> | null>(null);

  /**
   * Whether an auth action is currently loading.
   */
  const loading = ref<boolean>(false);

  /**
   * Whether the current session should be remembered.
   */
  const rememberMe = ref<boolean>(false);

  /**
   * Identity extracted from the current access token.
   */
  const identity = ref<TokenIdentity | null>(null);

  /**
   * Cached permission snapshot used for client-side UX trimming.
   */
  const permissionState = ref<PermissionState | null>(null);

  /**
   * Whether auth initialization has completed.
   */
  const initialized = ref<boolean>(false);

  // Internal state.

  /**
   * Internal refresh bookkeeping.
   */
  const refreshState = reactive({
    timerId: null as number | null,
    refreshing: false,
    lastRefreshTime: 0,
  });

  /**
   * Auth options kept as a plain object because they rarely change at runtime.
   */
  const authOptions = createAuthOptions(options);

  // Computed state.

  /**
   * Whether the current access token is present and not expired.
   */
  const isAuthenticated: ComputedRef<boolean> = computed(() => {
    return !!tokens.value?.accessToken && tokens.value.expiresAt > Date.now();
  });

  /**
   * Whether the current access token is still valid.
   */
  const isAccessTokenValid = computed(() => {
    return !!tokens.value?.accessToken && tokens.value.expiresAt > Date.now();
  });

  /**
   * Whether the current refresh token is still valid.
   */
  const isRefreshTokenValid = computed(() => {
    return !!tokens.value?.refreshToken && tokens.value.refreshExpiresAt > Date.now();
  });

  /**
   * Whether the store should schedule or trigger a token refresh.
   */
  const shouldRefreshToken = computed(() => {
    // Avoid re-entrant refresh scheduling while a refresh is already running.
    if (refreshState.refreshing) return false;

    const { accessToken, expiresAt, refreshToken } = tokens.value || {};
    const now = Date.now();
    const hasValidAccessToken = !!accessToken && !!expiresAt && expiresAt > now;
    const hasRefreshToken = !!refreshToken;

    let ret = false;
    if (hasValidAccessToken) {
      // Refresh shortly before the access token crosses the configured threshold.
      ret = expiresAt - now < authOptions.refreshThreshold;
    } else if (hasRefreshToken) {
      // Recover a missing or expired access token through the refresh token when possible.
      ret = true;
    }

    return ret;
  });

  return {
    // Core state.
    tokens,
    currentUser,
    loading,
    rememberMe,
    identity,
    permissionState,
    initialized,

    // Internal state exposed to store helpers.
    refreshState,
    authOptions,

    // Computed state.
    isAuthenticated,
    isAccessTokenValid,
    isRefreshTokenValid,
    shouldRefreshToken,

    // Dependencies.
    userStore,
  };
}

export type AuthState = ReturnType<typeof defineAuthState>;
