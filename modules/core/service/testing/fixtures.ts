// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { ensureServiceError } from '../error/runtime';
import type { Context } from '../runtime/context/source';
import { withContext } from '../runtime/context/scope';

/**
 * Build a typed row for tests from a partial. The `as unknown as T` escape hatch
 * lives only here so production code does not scatter test casts.
 */
export function fakeRow<T>(partial: Partial<T>): T {
  return partial as unknown as T;
}

/**
 * Narrow unknown errors to {@link ChoysumError} (wraps non-Choysum errors).
 */
export function asChoysumError(err: unknown): ChoysumError {
  return ensureServiceError(err);
}

/**
 * Run `fn` under a merged business context patch (test convenience wrapper).
 */
export function fakeCtx<R>(partial: Partial<Context>, fn: () => R): R {
  return withContext(partial, fn, { merge: true });
}
