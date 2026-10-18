// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Public auth destinations (login / logout / register).
 * Dirty leave prompts are skipped when navigating to or from these routes —
 * credential shells are not durable drafts, and session exits cannot save.
 */
export function isAuthExitNavigation(to: {
  name?: unknown;
  path?: unknown;
  /** Accept vue-router meta bags; only `resourceId` is read. */
  meta?: { resourceId?: unknown } | Record<string, unknown> | null;
}): boolean {
  const name = String(to?.name ?? '').trim();
  if (name === 'login' || name === 'logout' || name === 'register') return true;

  const resourceId = String((to?.meta as { resourceId?: unknown } | null | undefined)?.resourceId ?? '').trim();
  if (
    resourceId === 'auth.route.login' ||
    resourceId === 'auth.route.logout' ||
    resourceId === 'auth.route.register'
  ) {
    return true;
  }

  const path = String(to?.path ?? '').split('?')[0]!.replace(/\/+$/, '') || '/';
  if (path === '/login' || path === '/logout' || path === '/register') return true;
  // App may be mounted under /web/ (exact segments only — not /pos/register).
  if (/^\/web\/(login|logout|register)$/.test(path)) return true;

  return false;
}

/** Skip the dirty-leave confirm when entering or leaving the public auth shell. */
export function shouldSkipFormDirtyLeavePrompt(
  to: Parameters<typeof isAuthExitNavigation>[0],
  from?: Parameters<typeof isAuthExitNavigation>[0] | null
): boolean {
  if (isAuthExitNavigation(to)) return true;
  if (from != null && isAuthExitNavigation(from)) return true;
  return false;
}
