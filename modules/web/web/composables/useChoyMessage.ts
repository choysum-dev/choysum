// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { toast } from 'vue-sonner';

export type ChoyMessageLevel = 'success' | 'warning' | 'error' | 'info';

export type ChoyMessageOptions = {
  /** Optional secondary line under the title. */
  description?: string;
  /** Milliseconds; 0 means stay until dismissed. */
  duration?: number;
};

const LEVEL_PREFIX: Record<ChoyMessageLevel, string> = {
  success: 'Success',
  warning: 'Warning',
  error: 'Error',
  info: 'Info',
};

function normalizeDuration(duration: number | undefined): number | undefined {
  if (duration === undefined) return undefined;
  if (duration === 0) {
    // Explicit 0 keeps the toast open until dismissed.
    return Number.POSITIVE_INFINITY;
  }
  if (Number.isFinite(duration) && duration > 0) {
    // Cap at the 32-bit timer limit: larger delays overflow and fire ~immediately.
    return Math.min(2_147_483_647, Math.max(1, Math.floor(duration)));
  }
  // Invalid (NaN / Infinity / negative) → omit so Sonner default applies.
  return undefined;
}

function show(level: ChoyMessageLevel, title: string, options?: ChoyMessageOptions): number {
  const prefix = LEVEL_PREFIX[level];
  const text = String(title ?? '').trim();
  const payload: { description?: string; duration?: number } = {};
  if (options?.description !== undefined) {
    payload.description = options.description;
  }
  const duration = normalizeDuration(options?.duration);
  if (duration !== undefined) {
    payload.duration = duration;
  }
  const message = text ? `${prefix}: ${text}` : prefix;
  const id = toast[level](message, payload);
  return typeof id === 'number' || typeof id === 'string' ? Number(id) || 0 : 0;
}

/**
 * Domain/shell toast facade over vue-sonner.
 * Host pages must mount `<Toaster />` from vendor/ui/sonner (product `App.vue` does).
 */
export const ChoyMessage = {
  success(title: string, options?: ChoyMessageOptions): number {
    return show('success', title, options);
  },
  warning(title: string, options?: ChoyMessageOptions): number {
    return show('warning', title, options);
  },
  error(title: string, options?: ChoyMessageOptions): number {
    return show('error', title, options);
  },
  info(title: string, options?: ChoyMessageOptions): number {
    return show('info', title, options);
  },
};

/**
 * Composable alias for ChoyMessage (same facade object).
 */
export function useChoyMessage() {
  return ChoyMessage;
}
