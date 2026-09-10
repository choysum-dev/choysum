// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Lightweight call recorders for FE unit tests. Call signatures stay intentionally
 * loose (`any`) so stubs assign to typed product callbacks without per-test casts.
 */

export type CallRecorder = { calls: unknown[][] };

/** Unused type params kept so older `FnRecorder<T, A>` imports still resolve. */
export type FnRecorder<_T = any, _A extends any[] = any[]> = CallRecorder &
  ((...args: any[]) => any) & {
    mockReset: () => void;
    mockClear: () => void;
    mockImplementation: (fn: (...args: any[]) => any) => void;
    mockReturnValue: (value: any) => void;
  };

/** @deprecated Prefer FnRecorder; kept as an alias for older imports. */
export type AnyFnRecorder = FnRecorder;

export type SyncFnRecorder<_T = any, _A extends any[] = any[]> = FnRecorder;

export type AsyncFnRecorder<_T = any, _A extends any[] = any[]> = CallRecorder &
  ((...args: any[]) => Promise<any>) & {
    mockReset: () => void;
    mockClear: () => void;
    mockImplementation: (fn: (...args: any[]) => any) => void;
    mockReturnValue: (value: any) => void;
  };

export type MaybeAsync<T> = T | Promise<T>;

export function fnRecorder(impl?: (...args: any[]) => any): FnRecorder {
  let current = impl;
  const rec = Object.assign(
    (...args: any[]) => {
      rec.calls.push(args);
      return current ? current(...args) : undefined;
    },
    {
      calls: [] as unknown[][],
      mockReset() {
        rec.calls = [];
        current = impl;
      },
      mockClear() {
        rec.calls = [];
      },
      mockImplementation(fn: (...args: any[]) => any) {
        current = fn;
      },
      mockReturnValue(value: any) {
        current = () => value;
      },
    }
  ) as FnRecorder;
  return rec;
}

/** Sync-only alias; same runtime as fnRecorder. */
export function syncFnRecorder(impl?: (...args: any[]) => any): SyncFnRecorder {
  return fnRecorder(impl);
}

/** Always returns a Promise so the stub matches `(...args) => Promise<T>` slots. */
export function asyncFnRecorder(impl?: (...args: any[]) => any): AsyncFnRecorder {
  const inner = fnRecorder(impl);
  const rec = Object.assign(
    (...args: any[]) => Promise.resolve(inner(...args)),
    {
      get calls() {
        return inner.calls;
      },
      set calls(v: unknown[][]) {
        inner.calls = v;
      },
      mockReset: () => inner.mockReset(),
      mockClear: () => inner.mockClear(),
      mockImplementation: (fn: (...args: any[]) => any) => inner.mockImplementation(fn),
      mockReturnValue: (value: any) => inner.mockReturnValue(value),
    }
  ) as AsyncFnRecorder;
  return rec;
}

/** Minimal AsyncIterable stub for tip stream subscriptions in tests. */
export function emptyAsyncIterable<T = never>(): AsyncIterable<T> {
  return {
    [Symbol.asyncIterator]() {
      return {
        next: async () => ({ done: true as const, value: undefined as T }),
      };
    },
  };
}
