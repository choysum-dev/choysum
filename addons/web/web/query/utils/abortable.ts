// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Code, ConnectError } from '@connectrpc/connect';

export class CancellationError extends Error {
  constructor(message = 'Operation canceled') {
    super(message);
    this.name = 'CancellationError';
  }
}

export function isCancellation(e: any): boolean {
  if (!e) return false;
  if (e instanceof CancellationError) return true;
  if (typeof e === 'object') {
    if ((e as any).name === 'AbortError') return true;
    if ((e as any).code === 'ABORT_ERR') return true;
    if (e instanceof ConnectError && e.code === Code.Canceled) return true;
  }
  return false;
}

export type AbortableTask<T> = (signal: AbortSignal, token: number) => Promise<T>;

/**
 * createAbortableRequests
 * - Maintain an AbortController per key
 * - Calling execute(key) aborts previous inflight task of the same key
 * - Provides isCancellation() utility
 */
export function createAbortableRequests() {
  const registry = new Map<string, { ctrl: AbortController; token: number }>();

  function execute<T>(key: string, task: AbortableTask<T>): Promise<T> {
    // Abort previous
    const prev = registry.get(key);
    if (prev) {
      try {
        prev.ctrl.abort();
      } catch {}
    }
    // New controller and token
    const ctrl = new AbortController();
    const token = (prev ? prev.token : 0) + 1;
    registry.set(key, { ctrl, token });

    const run = async (): Promise<T> => {
      // Early abort check
      if (ctrl.signal.aborted) throw new CancellationError();
      try {
        const res = await task(ctrl.signal, token);
        // If superseded meanwhile, treat as canceled
        const cur = registry.get(key);
        if (!cur || cur.token !== token) throw new CancellationError('Superseded by a newer request');
        return res;
      } catch (e) {
        if (isCancellation(e)) throw e;
        // If superseded meanwhile, still surface as cancellation
        const cur = registry.get(key);
        if (!cur || cur.token !== token) throw new CancellationError('Superseded by a newer request');
        throw e;
      }
    };

    return run();
  }

  function cancel(key?: string) {
    if (key) {
      const r = registry.get(key);
      if (r) {
        try {
          r.ctrl.abort();
        } finally {
          registry.delete(key);
        }
      }
      return;
    }
    // Cancel all
    for (const [k, r] of registry) {
      try {
        r.ctrl.abort();
      } finally {
        registry.delete(k);
      }
    }
  }

  return { execute, cancel };
}
