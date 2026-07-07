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
  try {
    delete (jsCtx as Record<PropertyKey, unknown>)[symbol];
  } catch {
    // ignore
  }
}
