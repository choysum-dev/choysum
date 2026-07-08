// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { asObjectRecord } from '../../../utils/object';
import { getJsCtxRoot } from './source';

/**
 * Resolve the current request context and backing request object.
 */
export function getJsCtxAndReq(): { jsCtx: any; req: any } {
  const jsCtx = asObjectRecord(getJsCtxRoot());
  if (!jsCtx) return { jsCtx: undefined, req: undefined };

  const request = asObjectRecord(jsCtx.request);
  const requestContext = asObjectRecord(request?.context);
  const nestedContext = asObjectRecord(jsCtx.context);
  const req = asObjectRecord(jsCtx.req) ?? asObjectRecord(requestContext?.req) ?? asObjectRecord(nestedContext?.req);

  return { jsCtx, req };
}

/**
 * Resolve the current request object from the active Choysum runtime context.
 */
export function getCurrentReq(): any {
  return getJsCtxAndReq().req;
}

/**
 * Return the request-scoped service cache, creating it when needed.
 */
export function getOrInitReqServiceState(req: any): any {
  const reqRecord = asObjectRecord(req);
  if (!reqRecord) return undefined;
  if (!reqRecord.__choysumServiceState) reqRecord.__choysumServiceState = {};
  return reqRecord.__choysumServiceState;
}

/**
 * Delete all keys with a given prefix from a request-scoped state record.
 *
 * Safe to call with undefined/null state (no-op).
 */
export function deleteReqStateKeysByPrefix(state: Record<string, unknown> | undefined, prefix: string): void {
  if (!state) return;
  for (const key of Object.keys(state)) {
    if (key.startsWith(prefix)) delete state[key];
  }
}

/**
 * Invalidate a symbol-keyed cache entry on a jsCtx carrier object.
 *
 * Safe to call when jsCtx is undefined or missing the symbol (no-op).
 */
export function invalidateJsCtxSymbolCache(jsCtx: unknown, symbol: symbol): void {
  if (!jsCtx) return;
  try {
    delete (jsCtx as Record<PropertyKey, unknown>)[symbol];
  } catch {
    // ignore
  }
}

/**
 * Run fn with incremented bypass depth counters on the given request-scoped state.
 *
 * Each key in `depthKeys` is incremented before fn runs and restored afterwards.
 * When a key's previous depth was 0, the key is deleted on restore rather than
 * being set to 0, so downstream `getBypassDepth` checks can use a simple truthy
 * test.
 *
 * Safe to call with undefined state (fn runs without depth tracking).
 */
export async function withBypassDepths<T>(state: Record<string, unknown> | undefined, depthKeys: string[], fn: () => Promise<T>): Promise<T> {
  if (!state) return await fn();

  for (const key of depthKeys) {
    const current = typeof state[key] === 'number' ? (state[key] as number) : 0;
    state[key] = current + 1;
  }

  try {
    return await fn();
  } finally {
    for (const key of depthKeys) {
      const current = typeof state[key] === 'number' ? (state[key] as number) : 0;
      if (current > 1) {
        state[key] = current - 1;
      } else {
        delete state[key];
      }
    }
  }
}

/**
 * Get-or-create a memoized value in request-scoped state with Promise leak guard.
 *
 * - If the key holds a resolved value, return it directly.
 * - If the key holds a pending Promise, await it, cache the result, and return.
 * - Otherwise call `factory`, store the Promise, and on resolve replace it with
 *   the value.  On rejection the key is deleted so the next caller retries.
 *
 * When state is undefined the factory runs without caching.
 */
export async function memoizeInReqState<T>(state: Record<string, unknown> | undefined, key: string, factory: () => Promise<T>): Promise<T> {
  if (!state) return await factory();

  if (key in state) {
    const existing = state[key];
    if (typeof (existing as { then?: unknown })?.then === 'function') {
      const v = await (existing as Promise<T>);
      try {
        if (state[key] === existing) {
          state[key] = v;
        }
      } catch {
        // ignore
      }
      return v;
    }
    return existing as T;
  }

  const p = factory()
    .then((v: T) => {
      try {
        if (state[key] === p) {
          state[key] = v;
        }
      } catch {
        // ignore
      }
      return v;
    })
    .catch((e: unknown) => {
      try {
        if (state[key] === p) {
          delete state[key];
        }
      } catch {
        // ignore
      }
      throw e;
    });

  try {
    state[key] = p;
  } catch {
    // ignore
  }
  return await p;
}
