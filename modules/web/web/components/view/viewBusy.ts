// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { cn, type ClassValue } from '../../lib/utils';

/** Host attrs for a busy view-body region (aria + inert; pair with overlay markup). */
export function viewBusyHostAttrs(busy: boolean): {
  'aria-busy'?: 'true';
  inert?: true;
} {
  if (!busy) return {};
  return { 'aria-busy': 'true', inert: true };
}

export function viewBusyHostClass(busy: boolean, extra?: ClassValue): string {
  return cn('relative min-h-0', busy && 'pointer-events-none', extra);
}
