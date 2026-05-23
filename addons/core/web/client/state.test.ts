// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Code, ConnectError } from '@connectrpc/connect';
import { describe, expect, it } from 'vitest';
import { createApiStateScope, type ApiStateInternal } from './state';

describe('client state', () => {
  it('tracks concurrent requests and clears loading only after the last request finishes', () => {
    const state = createApiStateScope('state-test') as ApiStateInternal;
    const key = state._internal.createStateKey('demo.Service', 'Fetch');

    state._internal.startRequest('req-1', key);
    state._internal.startRequest('req-2', key);

    expect(state.loading[key]).toBe(true);
    expect(state.isLoading.value).toBe(true);

    state._internal.finishRequest('req-1');
    expect(state.loading[key]).toBe(true);
    expect(state.isLoading.value).toBe(true);

    state._internal.finishRequest('req-2');
    expect(state.loading[key]).toBe(false);
    expect(state.isLoading.value).toBe(false);

    state._internal.finishRequest('missing');
    expect(state.loading[key]).toBe(false);
  });

  it('normalizes connect and generic errors, and supports clearing errors', () => {
    const state = createApiStateScope('state-test') as ApiStateInternal;
    const connectKey = state._internal.createStateKey('demo.Service', 'ConnectCall');
    const genericKey = state._internal.createStateKey('demo.Service', 'GenericCall');

    const connectError = Object.assign(Object.create(ConnectError.prototype), {
      code: Code.PermissionDenied,
      rawMessage: 'permission denied',
      message: 'permission denied',
      metadata: new Headers({ 'x-trace': 'trace-1' }),
    }) as ConnectError;

    const returnedConnectError = state._internal.handleError(connectKey, connectError);
    expect(returnedConnectError).toBe(connectError);
    expect(state.errors[connectKey]?.code).toBe('PermissionDenied');
    expect(state.errors[connectKey]?.message).toBe('permission denied');
    expect(state.lastError.value?.code).toBe('PermissionDenied');

    const genericError = new Error('boom');
    const returnedGenericError = state._internal.handleError(genericKey, genericError);
    expect(returnedGenericError).toBe(genericError);
    expect(state.errors[genericKey]?.code).toBe('UNKNOWN');
    expect(state.errors[genericKey]?.message).toBe('boom');
    expect(state.hasErrors()).toBe(true);

    state.clearError(genericKey);
    expect(state.errors[genericKey]).toBeNull();

    state.clearAllErrors();
    expect(state.errors[connectKey]).toBeNull();
    expect(state.errors[genericKey]).toBeNull();
    expect(state.lastError.value).toBeNull();
    expect(state.hasErrors()).toBe(false);
  });
});
