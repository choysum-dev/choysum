// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createClient, type Client, type Interceptor } from '@connectrpc/connect';
import type { DescService } from '@bufbuild/protobuf';
import { createWebTransport } from './transport_factory';

const clientCache = new Map<DescService, unknown>();

export function CreateWebClient<T extends DescService>(service: T, additionalInterceptors: Interceptor[] = []): () => Client<T> {
  return () => {
    const cachedClient = clientCache.get(service) as Client<T> | undefined;
    if (cachedClient) {
      return cachedClient;
    }

    const client = createClient(service, createWebTransport(additionalInterceptors));
    clientCache.set(service, client);
    return client;
  };
}
