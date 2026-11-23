// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RpcRequestContext, RequestLifecycleProvider } from '../../../rpc/types';
import { ensureHeaders } from '../../testing/qjs_polyfills';
import { setLifecycleProvider } from '../providers';
import { createLifecycleInterceptor } from './lifecycle';

ensureHeaders();

function makeReq(stream: boolean) {
  return {
    stream,
    message: { ok: true },
    service: { typeName: 'tip.TipHub' },
    method: { name: stream ? 'SubscribeNotifications' : 'Ping' },
    header: new Headers(),
  };
}

test('lifecycle interceptor: marks stream context and still runs next', async () => {
  const seen: RpcRequestContext[] = [];
  const provider: RequestLifecycleProvider = {
    onStart(context) {
      seen.push(context);
    },
  };
  setLifecycleProvider(provider);
  try {
    const intercept = createLifecycleInterceptor();
    const next = async (req: { stream: boolean }) => ({ stream: req.stream, body: 'ok' });
    const unary = await intercept(next as any)(makeReq(false) as any);
    const streamed = await intercept(next as any)(makeReq(true) as any);
    expect(unary).toEqual({ stream: false, body: 'ok' });
    expect(streamed).toEqual({ stream: true, body: 'ok' });
    expect(seen).toHaveLength(2);
    expect(seen[0]?.stream).toBe(false);
    expect(seen[0]?.methodName).toBe('Ping');
    expect(seen[1]?.stream).toBe(true);
    expect(seen[1]?.methodName).toBe('SubscribeNotifications');
    expect(seen[1]?.serviceName).toBe('tip.TipHub');
  } finally {
    setLifecycleProvider(null);
  }
});
