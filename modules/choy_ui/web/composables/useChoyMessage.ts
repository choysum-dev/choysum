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

function show(level: ChoyMessageLevel, title: string, options?: ChoyMessageOptions): number {
  const prefix =
    level === 'success'
      ? 'Success'
      : level === 'warning'
        ? 'Warning'
        : level === 'error'
          ? 'Error'
          : 'Info';
  return toast({
    title: title ? `${prefix}: ${title}` : prefix,
    description: options?.description,
    duration: options?.duration,
  });
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
