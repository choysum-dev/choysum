// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Auth store constants.
 */
export const AUTH_STORAGE_KEY = 'choysum.auth';
export const AUTH_HEADER_NAME = 'Authorization';
export const AUTH_HEADER_PREFIX = 'Bearer ';

/**
 * Store paths persisted across reloads.
 */
export const PERSIST_PATHS = ['tokens', 'rememberMe', 'permissionState'];

/**
 * Minimal auth module configuration options.
 */
export interface AuthOptions {
  /** Whether automatic token refresh is enabled. */
  autoRefresh: boolean;

  /** Milliseconds before expiry when refresh should start. */
  refreshThreshold: number;

  /** Milliseconds between token refresh checks. */
  refreshInterval: number;

  /** Whether to attach device information to login requests. */
  attachDeviceInfo: boolean;
}

/**
 * Default auth options.
 */
export const DEFAULT_AUTH_OPTIONS: AuthOptions = {
  autoRefresh: false,
  refreshThreshold: 120000, // Two minutes of headroom before expiry.
  refreshInterval: 30000, // Poll token state every 30 seconds.
  attachDeviceInfo: true, // Attach device information by default.
};

/**
 * Merge caller overrides into the default auth options.
 *
 * @param options - Partial caller overrides.
 * @returns The merged auth options.
 */
export function createAuthOptions(options?: Partial<AuthOptions>): AuthOptions {
  return {
    ...DEFAULT_AUTH_OPTIONS,
    ...(options || {}),
  };
}
