// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getRuntimeGlobalPoolValue } from '@/core/utils/env';
import { asObjectRecord, asRuntimeCarrier } from '@/core/utils/object';

type LocalServicePool = {
  get(name: string): unknown;
};

function asLocalServicePool(value: unknown): LocalServicePool | undefined {
  const record = asRuntimeCarrier(value) ?? asObjectRecord(value);
  if (!record || typeof record.get !== 'function') return undefined;
  const get = record.get as (this: unknown, name: string) => unknown;
  return {
    get(name: string): unknown {
      return get.call(value, name);
    },
  };
}

export async function tryLocalServiceCall(serviceName: string, methodName: string, args: unknown[]): Promise<{ handled: boolean; value?: unknown }> {
  const pool = asLocalServicePool(getRuntimeGlobalPoolValue());
  if (!pool) {
    return { handled: false };
  }

  const localModel = pool.get(serviceName);
  const localModelRecord = asRuntimeCarrier(localModel) ?? asObjectRecord(localModel);
  const method = localModelRecord?.[methodName];
  if (typeof method !== 'function') {
    return { handled: false };
  }

  try {
    const value = await (method as (this: unknown, ...input: unknown[]) => unknown).call(localModel, ...args);
    return { handled: true, value };
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error(`[SmartRouter] Local call failed for ${serviceName}.${methodName}`, error);
    throw error;
  }
}
