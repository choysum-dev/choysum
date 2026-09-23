// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { toast } from '../components/vendor/ui/toast/useToast';

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

function show(level: ChoyMessageLevel, title: string, options?: ChoyMessageOptions): number {
  const prefix = LEVEL_PREFIX[level];
  const payload: { title: string; description?: string; duration?: number } = {
    title: title ? `${prefix}: ${title}` : prefix,
  };
  if (options?.description !== undefined) {
    payload.description = options.description;
  }
  if (options?.duration !== undefined) {
    const duration = options.duration;
    if (duration === 0) {
      // Explicit 0 keeps the toast open until dismissed (store maps 0 to Infinity).
      payload.duration = 0;
    } else if (Number.isFinite(duration) && duration > 0) {
      payload.duration = Math.max(1, Math.floor(duration));
    }
    // Invalid (NaN / Infinity / negative) durations stay unset so the store
    // default applies instead of silently pinning a sticky toast.
  }
  return toast(payload);
}

/**
 * Domain/shell toast facade over the L2 toast store.
 * Host pages must mount `<Toaster />` (Gallery / Dogfood already do).
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
