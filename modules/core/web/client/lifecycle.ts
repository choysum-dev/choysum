// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ConnectError, Code } from '@connectrpc/connect';
import type { RpcRequestContext, RequestLifecycleProvider } from '../../rpc/types';
import { asObjectRecord } from '../../utils/object';
import type { ApiState, ApiStateInternal } from './state';

export function toPublicApiState(apiState: ApiStateInternal): ApiState {
  return {
    loading: apiState.loading,
    errors: apiState.errors,
    isLoading: apiState.isLoading,
    lastError: apiState.lastError,
    clearError: apiState.clearError,
    clearAllErrors: apiState.clearAllErrors,
    hasErrors: apiState.hasErrors,
  };
}

export function createLifecycleProvider(apiState: ApiStateInternal): RequestLifecycleProvider {
  return {
    onStart: (context: RpcRequestContext) => {
      const { serviceName, methodName, spanId } = context;
      const stateKey = apiState._internal.createStateKey(serviceName, methodName);
      apiState._internal.startRequest(spanId, stateKey);
    },

    onSuccess: () => {},

    onError: (context: RpcRequestContext, error: unknown) => {
      const { serviceName, methodName } = context;
      const stateKey = apiState._internal.createStateKey(serviceName, methodName);

      try {
        if (error instanceof ConnectError && error.code === Code.Canceled) {
          return error;
        }
        const errorRecord = asObjectRecord(error);
        if (errorRecord && (errorRecord.name === 'AbortError' || errorRecord.code === 'ABORT_ERR')) {
          return error;
        }
      } catch {}

      return apiState._internal.handleError(stateKey, error);
    },

    onFinish: (context: RpcRequestContext) => {
      apiState._internal.finishRequest(context.spanId);
    },
  };
}
