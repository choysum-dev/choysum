// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type ClientModelService, type ParamDescriptor, type ResultDescriptor, type RpcServiceFn } from '../../rpc/types';
import { castProcessedReturnType, resolveRpcResponsePayload, resolveRpcResultPayload } from '../../rpc/result_payload';
import Decimal from 'decimal.js';
import { serialize, deserialize, isDecimalLeak } from '../../utils/decimal';
import { asObjectRecord, isObjectRecord } from '../../utils/object';
import { logServerRpcError } from './server_errors';
import { tryLocalServiceCall } from './server_routing';
import type { UnknownRecord } from '../../utils/types';

function normalizeModelAwareValue(input: unknown, seen: WeakMap<object, unknown> = new WeakMap()): unknown {
  if (input == null) return input;

  const type = typeof input;
  if (type === 'string' || type === 'number' || type === 'boolean' || type === 'bigint') {
    return input;
  }

  if (input instanceof Date) return input.toISOString();

  if (input instanceof Error) {
    const errOut: UnknownRecord = {
      name: input.name,
      message: input.message,
    };
    const code = asObjectRecord(input)?.code;
    if (code !== undefined) errOut.code = code;
    return errOut;
  }

  if (type !== 'object') return input;

  if (Array.isArray(input)) {
    if (seen.has(input)) return seen.get(input);
    const out: unknown[] = [];
    seen.set(input, out);
    for (const item of input) {
      out.push(normalizeModelAwareValue(item, seen));
    }
    return out;
  }

  if (input instanceof Map) {
    const out: UnknownRecord = {};
    for (const [k, v] of input.entries()) {
      out[String(k)] = normalizeModelAwareValue(v, seen);
    }
    return out;
  }

  if (input instanceof Set) {
    const out: unknown[] = [];
    for (const v of input.values()) {
      out.push(normalizeModelAwareValue(v, seen));
    }
    return out;
  }

  if (seen.has(input)) return seen.get(input);

  if (isDecimalLeak(input)) {
    try {
      const reconstructed = new Decimal(0);
      const mutable = reconstructed as unknown as { s: number; e: number; d: number[] };
      mutable.s = input.s;
      mutable.e = input.e;
      mutable.d = input.d.slice();
      return { $bigdecimal: reconstructed.toString() };
    } catch {
      // fall through
    }
  }

  const inputRecord = asObjectRecord(input);

  if (typeof inputRecord?.toTransportObject === 'function') {
    try {
      const plain = inputRecord.toTransportObject();
      return normalizeModelAwareValue(plain, seen);
    } catch {
      // fall through
    }
  }

  if (typeof inputRecord?.toPlainObject === 'function') {
    try {
      const plain = inputRecord.toPlainObject();
      return normalizeModelAwareValue(plain, seen);
    } catch {
      // fall through
    }
  }

  if (!isObjectRecord(input)) {
    // Keep non-plain objects (e.g. Decimal instances) untouched here; encode/decode
    // normalization is handled by serialize()/deserialize() in a single shared path.
    return input;
  }

  const out: UnknownRecord = {};
  seen.set(input, out);

  const descs = Object.getOwnPropertyDescriptors(input);
  for (const key of Object.keys(descs)) {
    const d = descs[key];
    if (!d) continue;
    if ('get' in d || 'set' in d) continue;
    const v = d.value;
    if (typeof v === 'function') continue;
    out[key] = normalizeModelAwareValue(v, seen);
  }

  return out;
}

function buildRequestPayload(param: ParamDescriptor): unknown {
  if (param.type === 'google.protobuf.Empty') {
    return {};
  }

  // For server-side bridge calls we normalize model instances first, then
  // serialize Decimal/BigInt envelopes for a remote-compatible payload.
  return serialize(normalizeModelAwareValue(param.value));
}

function buildLocalCallArg(param: ParamDescriptor, requestPayload: unknown): unknown {
  if (param.type === 'google.protobuf.Empty') {
    return {};
  }

  return deserialize(requestPayload);
}

function normalizeResultPayload(payload: unknown): unknown {
  // Keep a single normalization path for local and remote results.
  return deserialize(serialize(normalizeModelAwareValue(payload)));
}

/**
 * Create API service method for Server environment (QuickJS)
 * Uses the injected Choysum.Grpc bridge to communicate with Go services.
 *
 * @param serviceName The full service name (e.g. "auth.User")
 * @param methodName The method name (e.g. "Login")
 * @param paramConverter Function to convert arguments to ParamDescriptors
 * @param resultDescriptor Descriptor for the result type
 */
export function CreateServerApiService<F extends RpcServiceFn>(
  serviceName: string,
  methodName: string,
  paramConverter: (...args: Parameters<F>) => ParamDescriptor[],
  resultDescriptor?: ResultDescriptor
): ClientModelService<F> {
  return async (...args: Parameters<F>) => {
    const paramDescriptors = paramConverter(...args);
    const request: UnknownRecord = {};
    const localArgs: unknown[] = [];

    for (const param of paramDescriptors) {
      const requestPayload = buildRequestPayload(param);
      request[param.name] = requestPayload;
      localArgs.push(buildLocalCallArg(param, requestPayload));
    }

    // === 1. Smart Routing: Local In-Process ===
    const local = await tryLocalServiceCall(serviceName, methodName, localArgs);
    if (local.handled) {
      const payload = resolveRpcResultPayload(resultDescriptor, local.value);
      if (payload === undefined) {
        return castProcessedReturnType<F>(undefined);
      }
      return castProcessedReturnType<F>(normalizeResultPayload(payload));
    }

    // === 2. Remote Call: Go Bridge (Remote gRPC) ===

    let response: unknown;
    try {
      // Invoke via Go Bridge
      // The bridge handles Smart Routing (Local vs Remote) and Proto serialization
      response = await $choysum.grpc.unary(serviceName, methodName, request);
    } catch (err) {
      logServerRpcError(serviceName, methodName, err);
      throw err;
    }

    // Handle result based on descriptor
    const payload = resolveRpcResponsePayload(resultDescriptor, response);
    if (payload === undefined) {
      return castProcessedReturnType<F>(undefined);
    }

    return castProcessedReturnType<F>(normalizeResultPayload(payload));
  };
}
