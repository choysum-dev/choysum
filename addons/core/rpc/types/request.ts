// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export interface ParamDescriptor {
  name: string;
  type: string;
  value: unknown;
}

export interface ResultDescriptor {
  name: string;
  type: string;
}

export interface TokenProvider {
  getToken(): Promise<string | null>;
  refreshToken(): Promise<boolean>;
  shouldRefreshToken?(): Promise<boolean> | boolean;
}

export interface CSRFProvider {
  getCSRFToken(): Promise<string | null>;
}

export interface RpcRequestContext {
  serviceName: string;
  methodName: string;
  traceId: string;
  spanId: string;
  args: unknown[];
}

/** @deprecated Prefer RpcRequestContext. Kept for compatibility. */
export type RequestContext = RpcRequestContext;

export interface RequestLifecycleProvider {
  onStart?: (context: RpcRequestContext) => void;
  onSuccess?: (context: RpcRequestContext, result: unknown) => void;
  onError?: (context: RpcRequestContext, error: unknown) => unknown;
  onFinish?: (context: RpcRequestContext) => void;
}
