// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { runWithRequestContext, type RequestContextKV } from '../../rpc/context';
import type { ApiError, ApiState } from './state';
import { useApiState, type ApiRuntime } from './runtime';
import type { ObjectRecord } from '../../utils/types';

type ServiceMethod = (...args: unknown[]) => unknown;
type ServiceRecord = ObjectRecord;

export interface ApiServiceExtensions {
  getMethodStateKey: (methodName: string) => string;
  isMethodLoading: (methodName: string) => boolean;
  getMethodError: (methodName: string) => ApiError | null;
  clearMethodError: (methodName: string) => void;
  setContext: (ctx: RequestContextKV) => void;
  getContext: () => RequestContextKV;
  withContext: <T>(ctx: RequestContextKV, fn: () => Promise<T>) => Promise<T>;
}

export type EnhancedApiService<T> = T & ApiState & ApiServiceExtensions;

export function webApiService<T extends ServiceRecord>(serviceName: string, service: T, runtime?: ApiRuntime): EnhancedApiService<T> {
  const state = runtime ? runtime.state : useApiState();
  const name = serviceName;
  let defaultCtx: RequestContextKV = {};

  const wrapped: ServiceRecord = {};
  const serviceRecord = service as ServiceRecord;
  for (const key of Object.keys(serviceRecord)) {
    const fn = serviceRecord[key];
    if (typeof fn !== 'function') {
      wrapped[key] = fn;
      continue;
    }

    const method = fn as ServiceMethod;
    wrapped[key] = async (...args: unknown[]) => {
      const hasCtx = defaultCtx && Object.keys(defaultCtx).length > 0;
      if (!hasCtx) {
        return await method.apply(service, args);
      }

      return await runWithRequestContext(defaultCtx, async () => {
        return await method.apply(service, args);
      });
    };
  }

  const extensions: ApiServiceExtensions = {
    getMethodStateKey: (methodName: string) => `${name}.${methodName}`,

    isMethodLoading: (methodName: string) => {
      const key = `${name}.${methodName}`;
      return !!state.loading[key];
    },

    getMethodError: (methodName: string) => {
      const key = `${name}.${methodName}`;
      return state.errors[key] || null;
    },

    clearMethodError: (methodName: string) => {
      const key = `${name}.${methodName}`;
      state.clearError(key);
    },

    setContext: (ctx: RequestContextKV) => {
      defaultCtx = { ...(ctx || {}) };
    },

    getContext: (): RequestContextKV => {
      return { ...defaultCtx };
    },

    withContext: async <T>(ctx: RequestContextKV, fn: () => Promise<T>): Promise<T> => {
      const mergedCtx = { ...(defaultCtx || {}), ...(ctx || {}) };
      return await runWithRequestContext(mergedCtx, fn);
    },
  };

  return {
    ...wrapped,
    ...state,
    ...extensions,
  } as EnhancedApiService<T>;
}
