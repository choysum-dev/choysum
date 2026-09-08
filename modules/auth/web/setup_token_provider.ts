// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { TokenProvider } from '@/core/rpc/types';
import { setTokenProvider } from '@/core/web/rpc';
import type { useAuthStore } from './stores/auth';

/**
 * Register the RPC token provider backed by the auth store.
 *
 * @param authStore - Auth state store.
 */
export function setupTokenProvider(authStore: ReturnType<typeof useAuthStore>): void {
  const tokenProvider: TokenProvider = {
    // Return the current access token for outgoing RPC calls.
    getToken: async () => authStore.tokens?.accessToken || null,

    // Refresh the token pair when the transport requires a new access token.
    refreshToken: async () => {
      if (!authStore.refreshToken) {
        if (import.meta.env.DEV) {
          console.debug('[Auth] No refresh token available; skipping refresh');
        }
        return false;
      }

      try {
        await authStore.refreshToken();

        if (import.meta.env.DEV) {
          console.debug('[Auth] Token refresh succeeded');
        }

        return !!authStore.tokens?.accessToken;
      } catch (error) {
        // Token refresh failures are expected after key rotation or database
        // resets. Log at warning level instead of error to avoid noise.
        // The store action (refreshTokenImpl) already calls clearAuth before
        // rethrowing, so tokens are already null — no need to attempt logout.
        console.warn('[Auth] Token refresh failed:', error);

        return false;
      }
    },

    // Expose the store-level refresh decision derived from expiresAt.
    shouldRefreshToken: async () => !!authStore.shouldRefreshToken,
  };

  setTokenProvider(tokenProvider);

  if (import.meta.env.DEV) {
    console.debug('[Auth] Token provider configured');
  }
}
