// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { cn, type ClassValue } from '../../lib/utils';

/** Surface busy presentation (see modules/web/web/docs/choy-loading-feedback.md). */
export type ViewBusyMode = 'stale' | 'skeleton' | 'block';

/** Host attrs for a busy view-body region (aria + optional inert + soft-stale marker). */
export function viewBusyHostAttrs(
  busy: boolean,
  mode: ViewBusyMode = 'block',
): {
  'aria-busy'?: 'true';
  inert?: true;
  'data-busy-mode'?: ViewBusyMode;
} {
  if (!busy) return {};
  if (mode === 'stale') {
    // Keep content readable/focusable; search lives outside this host.
    return { 'aria-busy': 'true', 'data-busy-mode': 'stale' };
  }
  return { 'aria-busy': 'true', inert: true, 'data-busy-mode': mode };
}

export function viewBusyHostClass(
  busy: boolean,
  mode: ViewBusyMode = 'block',
  extra?: ClassValue,
): string {
  return cn(
    // Soft stale: opacity transition lives on the host (see index.css).
    'relative min-h-0 choy-view-busy-host',
    busy && mode === 'stale' && 'opacity-70',
    busy && mode !== 'stale' && 'pointer-events-none',
    extra,
  );
}
