// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Run the Login page's mount-time auth readiness check.
 *
 * Captures the route path before await so a mid-flight navigation away from
 * login does not trigger an unwanted redirect when init finishes.
 */
export async function runLoginAuthReady(opts: {
  ensureAuthReady: () => Promise<void>;
  getRoutePath: () => string;
  isAuthenticated: () => boolean;
  redirect: () => void;
}): Promise<void> {
  const currentPath = opts.getRoutePath();
  try {
    await opts.ensureAuthReady();
  } catch {
    // Stale tokens are cleared by initAuth internally; continue to login.
  }
  if (opts.getRoutePath() === currentPath && opts.isAuthenticated()) {
    opts.redirect();
  }
}
