// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { asObjectRecord } from '../../../utils/object';
import { getJsCtxRoot } from './source';

/** Request-scoped userId override stack (nested / concurrent withUser). */
const USER_ID_OVERRIDE_KEY = Symbol.for('choysum.userid.override');

type UserIdOverrideEntry = { userId: string };

/** Process-level override stack used when no jsCtx is available (scripts / background). */
const processLevelUserIdStack: UserIdOverrideEntry[] = [];

type UserIdCarrier = Record<PropertyKey, unknown>;

function asUserIdCarrier(value: unknown): UserIdCarrier | undefined {
  const record = asObjectRecord(value);
  return record ? (record as UserIdCarrier) : undefined;
}

function normalizeUserId(userId: unknown): string {
  const normalized = typeof userId === 'string' ? userId.trim() : String(userId ?? '').trim();
  if (!normalized) {
    throw new Error('withUser requires a non-empty userId');
  }
  return normalized;
}

function isPromiseLike<T = unknown>(value: unknown): value is PromiseLike<T> {
  return !!value && typeof (value as { then?: unknown }).then === 'function';
}

function peekStackTop(stack: unknown): string | undefined {
  if (!Array.isArray(stack) || stack.length === 0) return undefined;
  const top = stack[stack.length - 1] as UserIdOverrideEntry | string | undefined;
  if (top && typeof top === 'object' && typeof top.userId === 'string' && top.userId) {
    return top.userId;
  }
  // Legacy string entries (should not appear after this change).
  return typeof top === 'string' && top ? top : undefined;
}

function removeStackEntry(stack: UserIdOverrideEntry[], entry: UserIdOverrideEntry): void {
  const idx = stack.indexOf(entry);
  if (idx !== -1) stack.splice(idx, 1);
}

function peekOverrideFromCarrier(carrier: UserIdCarrier | undefined): string | undefined {
  return peekStackTop(carrier?.[USER_ID_OVERRIDE_KEY]);
}

/**
 * Returns the active userId override (stack top), if any.
 * Prefer request-scoped stack; fall back to process-level stack.
 */
export function peekUserIdOverride(): string | undefined {
  const fromRequest = peekOverrideFromCarrier(asUserIdCarrier(getJsCtxRoot()));
  if (fromRequest) return fromRequest;
  return peekStackTop(processLevelUserIdStack);
}

/**
 * Returns the current user Id: withUser override stack top, else jsCtx.identity.userId.
 */
export function getUserId(): string | undefined {
  const override = peekUserIdOverride();
  if (override) return override;

  const jsCtx = getJsCtxRoot();
  const identity = asObjectRecord(jsCtx?.identity);
  return typeof identity?.userId === 'string' ? identity.userId : undefined;
}

/**
 * Runs a function with a temporary userId override for authz / getUserId().
 *
 * Does not wrap withContext and does not elevate privileges (use Model.sudo for bypass).
 * Sync and async `fn` are both supported (aligned with withContext).
 * Concurrent sibling withUser calls remove their own stack entry by identity (not LIFO pop).
 */
export function withUser<R>(userId: string, fn: () => R): R {
  const normalized = normalizeUserId(userId);
  const entry: UserIdOverrideEntry = { userId: normalized };
  const jsCtx = asUserIdCarrier(getJsCtxRoot());

  if (jsCtx) {
    let stack = jsCtx[USER_ID_OVERRIDE_KEY] as UserIdOverrideEntry[] | undefined;
    if (!Array.isArray(stack)) {
      stack = [];
      jsCtx[USER_ID_OVERRIDE_KEY] = stack;
    }
    stack.push(entry);

    const restore = () => {
      removeStackEntry(stack!, entry);
      if (stack!.length === 0) delete jsCtx[USER_ID_OVERRIDE_KEY];
    };

    try {
      const result = fn();
      if (isPromiseLike(result)) {
        return Promise.resolve(result).finally(restore) as unknown as R;
      }
      restore();
      return result;
    } catch (error) {
      restore();
      throw error;
    }
  }

  processLevelUserIdStack.push(entry);
  const restoreProcess = () => {
    removeStackEntry(processLevelUserIdStack, entry);
  };
  try {
    const result = fn();
    if (isPromiseLike(result)) {
      return Promise.resolve(result).finally(restoreProcess) as unknown as R;
    }
    restoreProcess();
    return result;
  } catch (error) {
    restoreProcess();
    throw error;
  }
}
