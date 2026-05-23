// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Interceptor } from '@connectrpc/connect';
import { createGrpcWebTransport } from '@connectrpc/connect-web';
import { buildWebInterceptors } from './interceptors';

const baseUrl = '/';

export function createWebTransport(additionalInterceptors: Interceptor[] = []) {
  return createGrpcWebTransport({
    baseUrl,
    interceptors: buildWebInterceptors(additionalInterceptors),
  });
}