// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * After ensureAuthReady, redirect only if the user is still on the path captured
 * at mount start and the session is already authenticated.
 */
export function shouldRedirectAfterAuthInit(
  pathAtMount: string,
  pathNow: string,
  isAuthenticated: boolean
): boolean {
  return pathNow === pathAtMount && isAuthenticated;
}
