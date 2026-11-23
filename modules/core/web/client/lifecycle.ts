// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ConnectError, Code } from '@connectrpc/connect';
import type { RpcRequestContext, RequestLifecycleProvider } from '../../rpc/types';
import { asObjectRecord } from '../../utils/object';
import { isTransportOutage } from '../rpc/transportOutage';
import { getConnectivityHooks } from './connectivityHooks';
import { loadingRegistryOnRpcFinish, loadingRegistryOnRpcStart } from './loadingRegistry';
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
      // Tip/subscribe streams stay pending for the session; they must not keep the
      // top bar or bottom slow-hint spinning after page data has already loaded.
      if (context.stream) return;
      const { serviceName, methodName, spanId } = context;
      const stateKey = apiState._internal.createStateKey(serviceName, methodName);
      apiState._internal.startRequest(spanId, stateKey);
      loadingRegistryOnRpcStart(spanId);
    },

    onSuccess: () => {
      try {
        getConnectivityHooks().onTransportSuccess?.();
      } catch {
        // Connectivity hooks must never break RPC success.
      }
    },

    onError: (context: RpcRequestContext, error: unknown) => {
      // Streams never called startRequest — do not invent unpaired error state.
      // Tip subscribe errors are handled by the inbox (poll fallback), not apiState.
      // Tip stream death is also not connectivity-authoritative; unary/chrome RPCs own that.
      if (context.stream) return error;

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

      try {
        if (isTransportOutage(error)) {
          getConnectivityHooks().onTransportOutage?.();
        } else if (error instanceof ConnectError) {
          // Server answered with a non-transport outcome — streak is not consecutive.
          getConnectivityHooks().onTransportSuccess?.();
        }
      } catch {
        // Connectivity hooks must never break RPC error propagation.
      }

      return apiState._internal.handleError(stateKey, error);
    },

    onFinish: (context: RpcRequestContext) => {
      if (context.stream) return;
      try {
        loadingRegistryOnRpcFinish(context.spanId);
      } finally {
        apiState._internal.finishRequest(context.spanId);
      }
    },
  };
}
