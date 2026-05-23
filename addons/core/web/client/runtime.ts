// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { setLifecycleProvider } from '../rpc';
import { createLifecycleProvider, toPublicApiState } from './lifecycle';
import { createApiStateScope, type ApiError, type ApiState, type ApiStateInternal, type ErrorState, type LoadingState } from './state';

export type { ApiError, ApiState, ErrorState, LoadingState } from './state';
export { createApiStateScope } from './state';

export interface ApiRuntime {
  scopeName: string;
  state: ApiState;
  bindLifecycleProvider: () => void;
}

let defaultApiRuntime: ApiRuntime | undefined;

export function createApiRuntime(scopeName: string = 'global'): ApiRuntime {
  const apiState = createApiStateScope(scopeName) as ApiStateInternal;
  const provider = createLifecycleProvider(apiState);

  return {
    scopeName,
    state: toPublicApiState(apiState),
    bindLifecycleProvider: () => {
      setLifecycleProvider(provider);
    },
  };
}

export function initializeApiRuntime(runtime?: ApiRuntime): ApiRuntime {
  const nextRuntime = runtime ?? createApiRuntime('global');
  nextRuntime.bindLifecycleProvider();
  defaultApiRuntime = nextRuntime;
  return nextRuntime;
}

function ensureDefaultApiRuntime(): ApiRuntime {
  if (!defaultApiRuntime) {
    defaultApiRuntime = initializeApiRuntime();
  }
  return defaultApiRuntime;
}

export function useApiState(): ApiState {
  return ensureDefaultApiRuntime().state;
}
