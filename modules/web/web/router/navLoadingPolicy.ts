// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteLocationNormalized } from 'vue-router';

/** Nav chrome settle policy for a destination route. */
export type NavLoadingPolicy = 'content' | 'route' | 'none';

/**
 * Resolves meta.navLoading for the destination. Omit defaults to content
 * (idle grace until first surface owns chrome).
 */
export function resolveNavLoadingPolicy(
  to: Pick<RouteLocationNormalized, 'meta'>
): NavLoadingPolicy {
  const value = to.meta?.navLoading;
  if (value === 'route' || value === 'none' || value === 'content') return value;
  return 'content';
}
