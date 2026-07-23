// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Interceptor } from '@connectrpc/connect';
import { createAuthInterceptor } from './auth';
import { createContextInterceptor } from './context';
import { createCSRFInterceptor } from './csrf';
import { createLifecycleInterceptor } from './lifecycle';

export { createAuthInterceptor } from './auth';
export { createContextInterceptor } from './context';
export { createCSRFInterceptor } from './csrf';
export { createLifecycleInterceptor } from './lifecycle';

export function buildWebInterceptors(additionalInterceptors: Interceptor[] = []): Interceptor[] {
  const interceptors = [
    createCSRFInterceptor(),
    createAuthInterceptor(),
    createContextInterceptor(),
    createLifecycleInterceptor(),
  ].filter((interceptor): interceptor is Interceptor => Boolean(interceptor));

  return [...interceptors, ...additionalInterceptors];
}
