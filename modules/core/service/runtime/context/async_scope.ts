// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { asObjectRecord } from '../../../utils/object';
import { getJsCtxRoot } from './source';

/**
 * Shared guard for withContext / withUser (policy A).
 *
 * Overlapping async scopes (e.g. Promise.all of two withUser) are rejected because
 * QuickJS has no AsyncLocalStorage: getUserId / getReadonlyCtx read a shared carrier.
 *
 * Allowed:
 * - sync nesting
 * - nested enter during an outer callback's synchronous prelude (before it yields)
 * - sequential `await withX(...); await withX(...)`
 *
 * Not allowed without ALS:
 * - sibling concurrent async scopes
 * - starting a new async scope after the outer async callback has already yielded
 *   (syncDepth is 0 while outer is still pending)
 */

const ASYNC_SCOPE_KEY = Symbol.for('choysum.async.scope');

type AsyncScopeState = {
  syncDepth: number;
  pendingAsync: number;
};

const processLevelAsyncScope: AsyncScopeState = { syncDepth: 0, pendingAsync: 0 };

type AsyncScopeCarrier = Record<PropertyKey, unknown>;

function getAsyncScopeState(): AsyncScopeState {
  const jsCtx = asObjectRecord(getJsCtxRoot()) as AsyncScopeCarrier | undefined;
  if (!jsCtx) return processLevelAsyncScope;

  let state = jsCtx[ASYNC_SCOPE_KEY] as AsyncScopeState | undefined;
  if (!state || typeof state !== 'object') {
    state = { syncDepth: 0, pendingAsync: 0 };
    jsCtx[ASYNC_SCOPE_KEY] = state;
  }
  return state;
}

function isPromiseLike<T = unknown>(value: unknown): value is PromiseLike<T> {
  return !!value && typeof (value as { then?: unknown }).then === 'function';
}

/**
 * Throws when entering withContext/withUser would overlap an in-flight async scope.
 */
export function assertNoOverlappingAsyncScope(apiName: string): void {
  const state = getAsyncScopeState();
  if (state.pendingAsync > 0 && state.syncDepth === 0) {
    throw new Error(
      `overlapping async ${apiName} is not supported; await one withContext/withUser scope before starting another, or nest inside the active callback before it yields`
    );
  }
}

/**
 * Runs `fn` while marking synchronous nesting depth so nested withContext/withUser are allowed.
 * If `fn` returns a Promise, tracks it as a pending async scope until settlement.
 */
export function runWithAsyncScopeTracking<R>(fn: () => R): R {
  const state = getAsyncScopeState();
  state.syncDepth += 1;
  try {
    const result = fn();
    if (isPromiseLike(result)) {
      state.pendingAsync += 1;
      return Promise.resolve(result).finally(() => {
        state.pendingAsync -= 1;
      }) as unknown as R;
    }
    return result;
  } finally {
    state.syncDepth -= 1;
  }
}

export function isPromiseLikeResult(value: unknown): value is PromiseLike<unknown> {
  return isPromiseLike(value);
}
