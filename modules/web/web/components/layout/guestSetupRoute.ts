// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * True when the guest shell is on initial setup (`/web/setup`).
 * Home nav is omitted there so guests are not sent in a loop through `/`.
 */
export function isGuestSetupRoute(route: {
  name?: unknown
  path?: string
} | null | undefined): boolean {
  if (!route) return false
  if (route.name === 'setup') return true
  const path = String(route.path || '')
    .replace(/\/+$/, '')
    .toLowerCase()
  return path === '/setup' || path === '/web/setup'
}
