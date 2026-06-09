// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { setLifecycleProvider } from '../rpc';
import { createLifecycleInterceptor } from '../rpc/interceptors';
import { clearContextStack, getCurrentRequestContext } from '../../rpc/context';
import { webApiService } from './service';
import { createApiRuntime, initializeApiRuntime, useApiState } from './runtime';

describe('client service/runtime split', () => {
  it('does not bind a lifecycle provider on module import alone', () => {
    setLifecycleProvider(undefined as any);

    expect(createLifecycleInterceptor()).toBeUndefined();
  });

  it('lazily initializes the default runtime when api state is requested', () => {
    setLifecycleProvider(undefined as any);

    expect(createLifecycleInterceptor()).toBeUndefined();

    const state = useApiState();

    expect(state.hasErrors()).toBe(false);
    expect(typeof createLifecycleInterceptor()).toBe('function');
  });

  it('supports explicit runtime binding and service state scoping', async () => {
    setLifecycleProvider(undefined as any);

    const runtime = createApiRuntime('scoped');
    const service = {
      Fetch: async () => 'ok',
    };

    const api = webApiService('demo.Service', service, runtime);

    expect(createLifecycleInterceptor()).toBeUndefined();
    expect(api.loading).toBe(runtime.state.loading);
    expect(api.errors).toBe(runtime.state.errors);
    expect(api.getMethodStateKey('Fetch')).toBe('demo.Service.Fetch');

    api.setContext({ companyId: 'C1' });
    expect(api.getContext()).toEqual({ companyId: 'C1' });
    await expect(api.Fetch()).resolves.toBe('ok');

    initializeApiRuntime(runtime);
    expect(typeof createLifecycleInterceptor()).toBe('function');
  });

  it('uses default runtime when no runtime is provided', () => {
    setLifecycleProvider(undefined as any);

    const service = {
      Ping: async () => 'pong',
    };

    const api = webApiService('demo.Service', service);
    const defaultState = useApiState();

    expect(api.loading).toBe(defaultState.loading);
    expect(api.errors).toBe(defaultState.errors);
    expect(typeof api.getMethodStateKey).toBe('function');
  });

  it('reads and clears method-scoped loading/error state', () => {
    const runtime = createApiRuntime('method-state');
    const service = {
      Fetch: async () => 'ok',
    };

    const api = webApiService('demo.Service', service, runtime);
    const key = api.getMethodStateKey('Fetch');

    runtime.state.loading[key] = true;
    runtime.state.errors[key] = {
      code: 'UNKNOWN',
      message: 'boom',
      timestamp: Date.now(),
    };

    expect(api.isMethodLoading('Fetch')).toBe(true);
    expect(api.getMethodError('Fetch')?.message).toBe('boom');

    api.clearMethodError('Fetch');
    expect(api.getMethodError('Fetch')).toBeNull();
  });

  it('keeps non-function members and merges default context with scoped context', async () => {
    clearContextStack();

    const service = {
      version: 'v1',
      InspectContext: async () => ({ ...getCurrentRequestContext() }),
    };

    const api = webApiService('demo.Service', service, createApiRuntime('ctx-scope'));
    expect(api.version).toBe('v1');

    await expect(api.InspectContext()).resolves.toEqual({});

    api.setContext({ tenant: 'T1' });
    await expect(api.InspectContext()).resolves.toEqual({ tenant: 'T1' });

    const merged = await api.withContext({ feature: 'beta' }, async () => {
      return await api.InspectContext();
    });
    expect(merged).toEqual({ tenant: 'T1', feature: 'beta' });

    const captured = api.getContext();
    captured.tenant = 'mutated';
    expect(api.getContext()).toEqual({ tenant: 'T1' });
  });
});
