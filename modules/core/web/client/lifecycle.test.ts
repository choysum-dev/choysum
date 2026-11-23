// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Code, ConnectError } from '@connectrpc/connect';
import type { RpcRequestContext } from '../../rpc/types';
import { ensureHeaders } from '../testing/qjs_polyfills';
import { resetConnectivityHooksForTests, setConnectivityHooks } from './connectivityHooks';
import {
  getLoadingRegistryDebugState,
  resetLoadingRegistryForTests,
} from './loadingRegistry';
import { createLifecycleProvider, toPublicApiState } from './lifecycle';
import { createApiStateScope, type ApiStateInternal } from './state';

ensureHeaders();

function makeContext(
  methodName: string,
  spanId: string,
  extras: Partial<RpcRequestContext> = {},
): RpcRequestContext {
  return {
    serviceName: 'demo.Service',
    methodName,
    traceId: 'trace-1',
    spanId,
    args: [],
    ...extras,
  };
}

test('client lifecycle: exposes only public api state fields', () => {
  const internal = createApiStateScope('lifecycle-test') as ApiStateInternal;
  const publicState = toPublicApiState(internal);

  expect((publicState as any)._internal).toBeUndefined();
  expect(typeof publicState.clearError).toBe('function');
  expect(typeof publicState.clearAllErrors).toBe('function');
  expect(typeof publicState.hasErrors).toBe('function');
});

test('client lifecycle: updates loading state on start and finish', () => {
  resetLoadingRegistryForTests();
  const state = createApiStateScope('lifecycle-test') as ApiStateInternal;
  const provider = createLifecycleProvider(state);
  const context = makeContext('Fetch', 'span-1');

  provider.onStart?.(context);
  expect(state.loading['demo.Service.Fetch']).toBe(true);
  expect(state.isLoading.value).toBe(true);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(1);

  provider.onFinish?.(context);
  expect(state.loading['demo.Service.Fetch']).toBe(false);
  expect(state.isLoading.value).toBe(false);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
});

test('client lifecycle: streaming RPCs do not own api loading or chrome', () => {
  resetLoadingRegistryForTests();
  const state = createApiStateScope('lifecycle-stream') as ApiStateInternal;
  const provider = createLifecycleProvider(state);
  const context = makeContext('SubscribeNotifications', 'span-stream', { stream: true });

  provider.onStart?.(context);
  expect(state.loading['demo.Service.SubscribeNotifications']).toBeFalsy();
  expect(state.isLoading.value).toBe(false);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
  expect(getLoadingRegistryDebugState().barCount).toBe(0);

  provider.onFinish?.(context);
  expect(state.isLoading.value).toBe(false);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
});

test('client lifecycle: streaming onError does not record api error state', () => {
  resetLoadingRegistryForTests();
  const state = createApiStateScope('lifecycle-stream-err') as ApiStateInternal;
  const provider = createLifecycleProvider(state);
  const context = makeContext('SubscribeNotifications', 'span-stream-err', { stream: true });

  provider.onStart?.(context);
  const err = new Error('stream dropped');
  const returned = provider.onError?.(context, err);
  expect(returned).toBe(err);
  expect(state.isLoading.value).toBe(false);
  expect(state.errors['demo.Service.SubscribeNotifications']).toBeFalsy();
  expect(state.hasErrors()).toBe(false);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
});

test('client lifecycle: accepts success callbacks without mutating state', () => {
  const state = createApiStateScope('lifecycle-test') as ApiStateInternal;
  const provider = createLifecycleProvider(state);

  provider.onSuccess?.(undefined as any, undefined);

  expect(state.isLoading.value).toBe(false);
  expect(state.hasErrors()).toBe(false);
});

test('client lifecycle: notifies connectivity hooks on success and transport outage', () => {
  resetConnectivityHooksForTests();
  const successes: number[] = [];
  const outages: number[] = [];
  setConnectivityHooks({
    onTransportSuccess: () => successes.push(1),
    onTransportOutage: () => outages.push(1),
  });
  const state = createApiStateScope('lifecycle-test') as ApiStateInternal;
  const provider = createLifecycleProvider(state);
  provider.onSuccess?.(undefined as any, undefined);
  expect(successes).toHaveLength(1);

  const context = makeContext('Down', 'span-outage');
  const unavailable = Object.assign(Object.create(ConnectError.prototype), {
    code: Code.Unavailable,
    rawMessage: 'unavailable',
    message: 'unavailable',
    metadata: new Headers(),
    findDetails: () => [],
  }) as ConnectError;
  provider.onError?.(context, unavailable);
  expect(outages).toHaveLength(1);

  const business = Object.assign(Object.create(ConnectError.prototype), {
    code: Code.InvalidArgument,
    rawMessage: 'bad',
    message: 'bad',
    metadata: new Headers(),
    findDetails: () => [],
  }) as ConnectError;
  provider.onError?.(makeContext('Biz', 'span-biz'), business);
  expect(successes).toHaveLength(2);
  resetConnectivityHooksForTests();
});

test('client lifecycle: ignores canceled and abort-like errors', () => {
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

test('client lifecycle: normalizes non-ignored errors through api state handler', () => {
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
