// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useDebounceFn } from '@vueuse/core';

/**
 * Debounced function wrapper with cancellation helpers.
 */
export type DebouncedCancelable<T extends (...args: any[]) => any = (...args: any[]) => any> = ((...args: Parameters<T>) => void) & {
  cancel: () => void;
  flush: () => void;
  pending: () => boolean;
};

/**
 * Creates a debounced function that exposes cancel, flush, and pending helpers.
 */
export function useDebouncedFnCancelable<T extends (...args: any[]) => any>(fn: T, wait = 400): DebouncedCancelable<T> {
  let token = 0;

  const inner = useDebounceFn((callToken: number, ...args: Parameters<T>) => {
    if (callToken === token) {
      return fn(...args);
    }
  }, wait) as unknown as {
    (...args: any[]): void;
    cancel?: () => void;
    flush?: () => void;
    pending?: () => boolean;
  };

  const debounced = ((...args: Parameters<T>) => {
    token += 1;
    inner(token, ...args);
  }) as DebouncedCancelable<T>;

  debounced.cancel = () => {
    token += 1;
    inner.cancel?.();
  };
  debounced.flush = () => {
    token += 1;
    inner.flush?.();
  };
  debounced.pending = () => inner.pending?.() ?? false;

  return debounced;
}
