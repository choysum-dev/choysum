// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Destinations where a dirty form cannot be saved (session exit / public auth).
 * Leave guards must not prompt — discard silently and let navigation proceed.
 */
export function isAuthExitNavigation(to: {
  name?: unknown;
  path?: unknown;
  meta?: { resourceId?: unknown; requiresAuth?: unknown } | null;
}): boolean {
  const name = String(to?.name ?? '').trim();
  if (name === 'login' || name === 'logout' || name === 'register') return true;

  const resourceId = String(to?.meta?.resourceId ?? '').trim();
  if (
    resourceId === 'auth.route.login' ||
    resourceId === 'auth.route.logout' ||
    resourceId === 'auth.route.register'
  ) {
    return true;
  }

  const path = String(to?.path ?? '').split('?')[0]!.replace(/\/+$/, '') || '/';
  if (path === '/login' || path === '/logout' || path === '/register') return true;
  // App may be mounted under /web/.
  if (/\/(login|logout|register)$/.test(path)) return true;

  return false;
}
