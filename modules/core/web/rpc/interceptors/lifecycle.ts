// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Interceptor } from '@connectrpc/connect';
import { RandomIdGenerator } from '@opentelemetry/sdk-trace-base';
import type { RpcRequestContext } from '../../../rpc/types';
import { getLifecycleProvider } from '../providers';

const idGen = new RandomIdGenerator();

function buildTraceparent(traceId: string, spanId: string, sampled = true): string {
  return `00-${traceId}-${spanId}-${sampled ? '01' : '00'}`;
}

export function createLifecycleInterceptor(): Interceptor | undefined {
  const lifecycleProvider = getLifecycleProvider();
  if (!lifecycleProvider) {
    return undefined;
  }

  return next => async req => {
    const serviceName = req.service.typeName || req.service.constructor.name || 'unknown';
    const methodName = req.method.name;
    const args = req.stream ? [req.message] : [req.message];

    const traceId = idGen.generateTraceId();
    const spanId = idGen.generateSpanId();
    req.header.set('traceparent', buildTraceparent(traceId, spanId, true));

    const context: RpcRequestContext = { serviceName, methodName, traceId, spanId, args };

    try {
      lifecycleProvider.onStart?.(context);
      const result = await next(req);
      lifecycleProvider.onSuccess?.(context, result);
      return result;
    } catch (error) {
      throw lifecycleProvider.onError?.(context, error) || error;
    } finally {
      lifecycleProvider.onFinish?.(context);
    }
  };
}
