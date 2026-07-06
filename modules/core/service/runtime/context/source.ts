// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Immutable business context snapshot exposed to runtime callers.
 */
export type Context = Readonly<JsBusinessContext & ObjectRecord>;

import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';

function deepFreeze<T>(obj: T): T {
  if (!obj || typeof obj !== 'object' || Object.isFrozen(obj)) return obj;
  const stack: unknown[] = [obj as unknown];
  while (stack.length) {
    const cur = stack.pop();
    const record = asObjectRecord(cur);
    if (!record || Object.isFrozen(record)) continue;
    Object.freeze(record);
    for (const key of Object.keys(record)) {
      const val = record[key];
      if (val && typeof val === 'object' && !Object.isFrozen(val)) {
        stack.push(val);
      }
    }
  }
  return obj;
}

/**
 * Test seam for deep-freezing arbitrary values.
 */
export function __deepFreezeForTest<T>(obj: T): T {
  return deepFreeze(obj);
}

// Runtime-injected global $choysum object from QuickJS.
declare const $choysum: unknown | undefined;

function resolveRuntimeRoot(): ObjectRecord | undefined {
  const runtimeRoot = typeof $choysum !== 'undefined' ? $choysum : (globalThis as { $choysum?: unknown }).$choysum;
  return asObjectRecord(runtimeRoot);
}

function resolveRequestContextFromRoot(root: ObjectRecord): ObjectRecord | undefined {
  const getRequestContext = asObjectRecord(root)?.getRequestContext;
  if (typeof getRequestContext === 'function') {
    try {
      const fromAccessor = asObjectRecord((getRequestContext as () => unknown)());
      if (fromAccessor) return fromAccessor;
    } catch {
      // ignore accessor failures and continue to legacy fallbacks
    }
  }

  const getActiveRequest = asObjectRecord(root)?.getActiveRequest;
  if (typeof getActiveRequest === 'function') {
    try {
      const req = asObjectRecord((getActiveRequest as () => unknown)());
      const reqCtx = asObjectRecord(req?.context);
      if (reqCtx) return reqCtx;
    } catch {
      // ignore accessor failures and continue to legacy fallbacks
    }
  }

  const request = asObjectRecord(root.request);
  return asObjectRecord(request?.context) ?? asObjectRecord(root.context) ?? root;
}

/**
 * Resolves the jsCtx root object from the runtime carrier.
 */
export function getJsCtxRoot(): JsCtx | undefined {
  try {
    const root = resolveRuntimeRoot();
    if (!root) return undefined;
    return resolveRequestContextFromRoot(root) as JsCtx;
  } catch {
    return undefined;
  }
}

/**
 * Returns the readonly identity snapshot from jsCtx.
 */
export function getIdentity(): Readonly<JsIdentitySnapshot> {
  const jsCtx = getJsCtxRoot();
  return deepFreeze({ ...(asObjectRecord(jsCtx?.identity) ?? {}) }) as Readonly<JsIdentitySnapshot>;
}

/**
 * Returns the readonly request metadata snapshot from jsCtx.
 */
export function getReqMeta(): Readonly<JsRequestMeta> {
  const jsCtx = getJsCtxRoot();
  return deepFreeze({ ...(asObjectRecord(jsCtx?.req) ?? {}) }) as Readonly<JsRequestMeta>;
}

/**
 * Returns the current user Id from jsCtx, if available.
 */
export function getUserId(): string | undefined {
  const jsCtx = getJsCtxRoot();
  const identity = asObjectRecord(jsCtx?.identity);
  return typeof identity?.userId === 'string' ? identity.userId : undefined;
}
