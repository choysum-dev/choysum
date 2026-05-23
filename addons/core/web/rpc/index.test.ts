// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import * as rpcApi from './index';

describe('core/web rpc entrypoint export surface', () => {
  it('keeps the rpc facade limited to stable providers and grpc builders', () => {
    expect(Object.keys(rpcApi).sort()).toEqual(['CreateWebApiService', 'CreateWebClient', 'setCSRFProvider', 'setLifecycleProvider', 'setTokenProvider']);
  });

  it('exposes live runtime bindings', () => {
    expect(typeof rpcApi.setTokenProvider).toBe('function');
    expect(typeof rpcApi.setCSRFProvider).toBe('function');
    expect(typeof rpcApi.setLifecycleProvider).toBe('function');
    expect(typeof rpcApi.CreateWebClient).toBe('function');
    expect(typeof rpcApi.CreateWebApiService).toBe('function');
  });

  it('supports safe module replay without cache mutation', async () => {
    const replay = await import('./index');

    expect(replay.setTokenProvider).toBe(rpcApi.setTokenProvider);
    expect(replay.CreateWebClient).toBe(rpcApi.CreateWebClient);
    expect(replay.CreateWebApiService).toBe(rpcApi.CreateWebApiService);
  });
});
