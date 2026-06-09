// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { reactive, ref, shallowRef, type Ref } from 'vue';
import { ConnectError, Code } from '@connectrpc/connect';
import { asObjectRecord } from '../../utils/object';

export interface ApiError {
  code: string;
  message: string;
  details?: unknown;
  timestamp: number;
  originalError?: unknown;
}

export type LoadingState = Record<string, boolean>;

export type ErrorState = Record<string, ApiError | null>;

interface RequestTracker {
  stateKey: string;
  active: boolean;
}

export interface ApiState {
  loading: LoadingState;
  errors: ErrorState;
  isLoading: Readonly<Ref<boolean>>;
  lastError: Readonly<Ref<ApiError | null>>;
  clearError: (key: string) => void;
  clearAllErrors: () => void;
  hasErrors: () => boolean;
}

export type ApiStateInternal = ApiState & {
  _internal: {
    createStateKey: (serviceName: string, methodName: string) => string;
    handleError: (stateKey: string, error: unknown) => unknown;
    startRequest: (requestKey: string, stateKey: string) => void;
    finishRequest: (requestKey: string) => void;
  };
};

function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  const errorRecord = asObjectRecord(error);
  const message = errorRecord?.message;
  return typeof message === 'string' ? message : String(error);
}

export function createApiStateScope(scopeName: string = 'global'): ApiState {
  const loading = reactive<LoadingState>({});
  const errors = reactive<ErrorState>({});
  const isLoading = ref(false);
  const lastError = shallowRef<ApiError | null>(null);
  const activeRequests = reactive<Map<string, RequestTracker>>(new Map());

  function createStateKey(serviceName: string, methodName: string): string {
    return `${serviceName}.${methodName}`;
  }

  function handleError(stateKey: string, error: unknown): unknown {
    let apiError: ApiError;

    if (error instanceof ConnectError) {
      apiError = {
        code: Code[error.code] || String(error.code),
        message: error.rawMessage || error.message,
        details: error.metadata,
        timestamp: Date.now(),
        originalError: error,
      };
    } else {
      apiError = {
        code: 'UNKNOWN',
        message: resolveErrorMessage(error),
        timestamp: Date.now(),
        originalError: error,
      };
    }

    errors[stateKey] = apiError;
    lastError.value = apiError;

    return error;
  }

  function startRequest(requestKey: string, stateKey: string): void {
    activeRequests.set(requestKey, { stateKey, active: true });
    loading[stateKey] = true;
    errors[stateKey] = null;
    isLoading.value = true;
  }

  function finishRequest(requestKey: string): void {
    const request = activeRequests.get(requestKey);
    if (!request) return;

    const { stateKey } = request;
    activeRequests.delete(requestKey);

    const hasActiveRequests = Array.from(activeRequests.values()).some(req => req.stateKey === stateKey && req.active);
    if (!hasActiveRequests) {
      loading[stateKey] = false;
    }

    isLoading.value = activeRequests.size > 0;
  }

  function clearError(key: string): void {
    if (errors[key]) {
      errors[key] = null;
    }
  }

  function clearAllErrors(): void {
    Object.keys(errors).forEach(key => {
      errors[key] = null;
    });
    lastError.value = null;
  }

  function hasErrors(): boolean {
    return Object.values(errors).some(error => error !== null);
  }

  void scopeName;

  return {
    loading,
    errors,
    isLoading,
    lastError,
    clearError,
    clearAllErrors,
    hasErrors,
    _internal: {
      createStateKey,
      handleError,
      startRequest,
      finishRequest,
    },
  } as ApiStateInternal;
}
