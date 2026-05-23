// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isClient, useTimeoutFn } from '@vueuse/core';
import { extractIdentity } from './utils';
import { wrapAuthError, AuthErrCode, ChoysumError } from '../../error';
import type { AuthState } from './state';

/**
 * Build helper functions shared by auth store actions.
 */
export function defineAuthHelpers(state: AuthState) {
  let timeoutControl: ReturnType<typeof useTimeoutFn> | null = null;

  /**
   * Schedule the next token refresh attempt for the current session.
   */
  function setupRefreshTimer(refreshToken: () => Promise<unknown>): void {
    if (!isClient) return;

    // Reset any previous timer before scheduling a new one.
    clearRefreshTimer();

    // Compute the next refresh deadline relative to the configured threshold.
    const calculateNextRefresh = () => {
      if (!state.tokens.value?.expiresAt) return null;

      const now = Date.now();
      const expiryTime = state.tokens.value.expiresAt;
      const timeUntilThreshold = expiryTime - now - state.authOptions.refreshThreshold;

      // Clamp to a small positive delay once the threshold has already been crossed.
      return Math.max(timeUntilThreshold, 1000);
    };

    const scheduleRefresh = () => {
      const delay = calculateNextRefresh();
      if (delay === null) return;

      timeoutControl = useTimeoutFn(async () => {
        if (state.shouldRefreshToken.value && !state.refreshState.refreshing) {
          try {
            await refreshToken();
            // Refresh succeeded, so schedule the next deadline.
            scheduleRefresh();
          } catch {
            // Keep failures silent here and stop scheduling new refreshes.
          }
        } else {
          // Re-check later when the token is not yet ready to refresh.
          timeoutControl = useTimeoutFn(scheduleRefresh, state.authOptions.refreshInterval);
        }
      }, delay);
    };

    // Start the first scheduling cycle immediately.
    scheduleRefresh();
  }

  /**
   * Stop any pending refresh timer.
   */
  function clearRefreshTimer(): void {
    if (timeoutControl) {
      timeoutControl.stop();
      timeoutControl = null;
    }
  }

  /**
   * Refresh the cached token identity from the current access token.
   */
  function updateTokenIdentity(): boolean {
    if (!state.tokens.value?.accessToken) {
      state.identity.value = null;
      return false;
    }

    try {
      state.identity.value = extractIdentity(state.tokens.value.accessToken);
      return !!state.identity.value;
    } catch (error) {
      // Clear stale identity state before surfacing the wrapped error.
      state.identity.value = null;

      throw wrapAuthError(error, {
        code: AuthErrCode.IDENTITY_EXTRACTION_FAILED,
        message: 'Failed to extract identity from token',
      });
    }
  }

  /**
   * Reset in-memory auth state after logout or unrecoverable auth failures.
   */
  function resetAuthState(): void {
    // Clear the current identity and token state.
    state.tokens.value = null;
    state.identity.value = null;
    state.currentUser.value = null;
    state.rememberMe.value = false;

    // Reset refresh bookkeeping.
    state.refreshState.lastRefreshTime = 0;
    state.refreshState.refreshing = false;

    // Stop any scheduled refresh work.
    clearRefreshTimer();
  }

  return {
    clearRefreshTimer,
    setupRefreshTimer,
    updateTokenIdentity,
    resetAuthState,
  };
}

export type AuthHelpers = ReturnType<typeof defineAuthHelpers>;
