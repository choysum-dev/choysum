// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ProcessedReturnType, ResultDescriptor } from './types';
import type { ObjectRecord } from '../utils/types';

export function isEmptyRpcResultDescriptor(resultDescriptor?: ResultDescriptor): boolean {
  return !resultDescriptor || resultDescriptor.type === 'google.protobuf.Empty';
}

export function resolveRpcResultPayload(resultDescriptor: ResultDescriptor | undefined, payload: unknown): unknown {
  if (isEmptyRpcResultDescriptor(resultDescriptor)) {
    return undefined;
  }
  if (payload === undefined) {
    return undefined;
  }
  return payload;
}

export function resolveRpcResponsePayload(resultDescriptor: ResultDescriptor | undefined, response: unknown): unknown {
  const descriptor = resultDescriptor;
  if (!descriptor || descriptor.type === 'google.protobuf.Empty') {
    return undefined;
  }
  if (!response || typeof response !== 'object') {
    return undefined;
  }
  const row = response as ObjectRecord;
  return resolveRpcResultPayload(descriptor, row[descriptor.name]);
}

export function castProcessedReturnType<F extends (...args: never[]) => unknown>(payload: unknown): ProcessedReturnType<F> {
  return payload as ProcessedReturnType<F>;
}
