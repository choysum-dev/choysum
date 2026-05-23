// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Code, ConnectError } from '@connectrpc/connect';
import { describe, expect, it } from 'vitest';
import type { RpcRequestContext } from '../../rpc/types';
import { createLifecycleProvider, toPublicApiState } from './lifecycle';
import { createApiStateScope, type ApiStateInternal } from './state';

function makeContext(methodName: string, spanId: string): RpcRequestContext {
  return {
    serviceName: 'demo.Service',
    methodName,
    traceId: 'trace-1',
    spanId,
    args: [],
  };
}

describe('client lifecycle', () => {
  it('exposes only public api state fields', () => {
    const internal = createApiStateScope('lifecycle-test') as ApiStateInternal;
    const publicState = toPublicApiState(internal);

    expect((publicState as any)._internal).toBeUndefined();
    expect(typeof publicState.clearError).toBe('function');
    expect(typeof publicState.clearAllErrors).toBe('function');
    expect(typeof publicState.hasErrors).toBe('function');
  });

  it('updates loading state on start and finish', () => {
    const state = createApiStateScope('lifecycle-test') as ApiStateInternal;
    const provider = createLifecycleProvider(state);
    const context = makeContext('Fetch', 'span-1');

    provider.onStart?.(context);
    expect(state.loading['demo.Service.Fetch']).toBe(true);
    expect(state.isLoading.value).toBe(true);

    provider.onFinish?.(context);
    expect(state.loading['demo.Service.Fetch']).toBe(false);
    expect(state.isLoading.value).toBe(false);
  });

  it('accepts success callbacks without mutating state', () => {
    const state = createApiStateScope('lifecycle-test') as ApiStateInternal;
    const provider = createLifecycleProvider(state);

    provider.onSuccess?.(undefined as any, undefined);

    expect(state.isLoading.value).toBe(false);
    expect(state.hasErrors()).toBe(false);
  });

  it('ignores canceled and abort-like errors', () => {
    const state = createApiStateScope('lifecycle-test') as ApiStateInternal;
    const provider = createLifecycleProvider(state);
    const canceledContext = makeContext('Canceled', 'span-2');
    const abortContext = makeContext('Abort', 'span-3');

    provider.onStart?.(canceledContext);
    const canceledError = Object.assign(Object.create(ConnectError.prototype), {
      code: Code.Canceled,
      rawMessage: 'canceled',
      message: 'canceled',
      metadata: new Headers(),
    }) as ConnectError;

    const returnedCanceled = provider.onError?.(canceledContext, canceledError);
    expect(returnedCanceled).toBe(canceledError);
    expect(state.errors['demo.Service.Canceled']).toBeNull();

    provider.onStart?.(abortContext);
    const abortError = { name: 'AbortError', message: 'aborted' };
    const returnedAbort = provider.onError?.(abortContext, abortError);
    expect(returnedAbort).toBe(abortError);
    expect(state.errors['demo.Service.Abort']).toBeNull();
  });

  it('normalizes non-ignored errors through api state handler', () => {
    const state = createApiStateScope('lifecycle-test') as ApiStateInternal;
    const provider = createLifecycleProvider(state);
    const context = makeContext('Fail', 'span-4');

    provider.onStart?.(context);
    const error = new Error('failed');
    const returned = provider.onError?.(context, error);

    expect(returned).toBe(error);
    expect(state.errors['demo.Service.Fail']?.code).toBe('UNKNOWN');
    expect(state.errors['demo.Service.Fail']?.message).toBe('failed');
  });
});
