// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Client } from '@connectrpc/connect';
import { create, type DescService } from '@bufbuild/protobuf';
import { EmptySchema } from '@bufbuild/protobuf/wkt';
import type { ClientModelService, ParamDescriptor, ResultDescriptor, RpcServiceFn } from '../../rpc/types';
import { castProcessedReturnType, resolveRpcResponsePayload } from '../../rpc/result_payload';
import { serialize } from '../../utils/decimal';
import type { UnknownRecord } from '../../utils/types';
import { normalizeTransportError } from './errors';
import { convertFromValue, convertToValue } from './value_codec';

type RpcMethodInvoker = (request: UnknownRecord) => Promise<unknown>;

function buildRequestValue(param: ParamDescriptor): unknown {
  if (param.type === 'google.protobuf.Value') {
    return convertToValue(param.value);
  }

  if (param.type === 'google.protobuf.Empty') {
    return create(EmptySchema, {});
  }

  return serialize(param.value);
}

function resolveRpcMethodInvoker(client: Client<DescService>, serviceName: string, methodName: string): RpcMethodInvoker {
  const candidate = (client as unknown as UnknownRecord)[methodName];
  if (typeof candidate !== 'function') {
    throw new Error(`RPC method ${serviceName}.${methodName} is not available`);
  }
  return candidate as RpcMethodInvoker;
}

export function CreateWebApiService<F extends RpcServiceFn>(
  clientFactory: () => Client<DescService>,
  serviceName: string,
  methodName: string,
  paramConverter: (...args: Parameters<F>) => ParamDescriptor[],
  resultDescriptor?: ResultDescriptor
): ClientModelService<F> {
  return async (...args: Parameters<F>) => {
    const client = clientFactory();
    const invokeMethod = resolveRpcMethodInvoker(client, serviceName, methodName);
    const paramDescriptors = paramConverter(...args);
    const request: UnknownRecord = {};

    for (const param of paramDescriptors) {
      request[param.name] = buildRequestValue(param);
    }

    let response: unknown;
    try {
      response = await invokeMethod(request);
    } catch (error) {
      const normalizedError = normalizeTransportError(error);
      console.debug('API Error:', normalizedError);
      throw normalizedError;
    }

    const payload = resolveRpcResponsePayload(resultDescriptor, response);
    if (payload === undefined) {
      return castProcessedReturnType<F>(undefined);
    }

    if (resultDescriptor?.type === 'google.protobuf.Value') {
      return castProcessedReturnType<F>(convertFromValue(payload));
    }

    return castProcessedReturnType<F>(payload);
  };
}
