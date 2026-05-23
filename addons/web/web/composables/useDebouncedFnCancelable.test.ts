// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { useDebouncedFnCancelable } from './useDebouncedFnCancelable';

describe('useDebouncedFnCancelable', () => {
  it('debounces calls and only runs the latest scheduled call', () => {
    vi.useFakeTimers();

    const calls: number[] = [];
    const fn = (n: number) => {
      calls.push(n);
    };

    const debounced = useDebouncedFnCancelable(fn, 100);

    debounced(1);
    debounced(2);
    debounced(3);

    expect(calls).toEqual([]);

    vi.advanceTimersByTime(99);
    expect(calls).toEqual([]);

    vi.advanceTimersByTime(1);
    expect(calls).toEqual([3]);

    vi.useRealTimers();
  });

  it('cancel prevents the pending call from running', () => {
    vi.useFakeTimers();

    const calls: number[] = [];
    const debounced = useDebouncedFnCancelable((n: number) => calls.push(n), 100);

    debounced(1);
    debounced.cancel();

    vi.runAllTimers();
    expect(calls).toEqual([]);

    vi.useRealTimers();
  });
});
