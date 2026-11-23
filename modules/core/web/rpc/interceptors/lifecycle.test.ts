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
  const started: RpcRequestContext[] = [];
  const finished: RpcRequestContext[] = [];
  const errored: RpcRequestContext[] = [];
  const provider: RequestLifecycleProvider = {
    onStart(context) {
      started.push(context);
    },
    onFinish(context) {
      finished.push(context);
    },
    onError(context, error) {
      errored.push(context);
      return error;
    },
  };
  setLifecycleProvider(provider);
  try {
    const intercept = createLifecycleInterceptor();
    const nextOk = async (req: { stream: boolean }) => ({ stream: req.stream, body: 'ok' });
    const unary = await intercept(nextOk as any)(makeReq(false) as any);
    const streamed = await intercept(nextOk as any)(makeReq(true) as any);
    expect(unary).toEqual({ stream: false, body: 'ok' });
    expect(streamed).toEqual({ stream: true, body: 'ok' });
    expect(started).toHaveLength(2);
    expect(finished).toHaveLength(2);
    expect(started[0]?.stream).toBe(false);
    expect(started[0]?.methodName).toBe('Ping');
    expect(started[1]?.stream).toBe(true);
    expect(started[1]?.methodName).toBe('SubscribeNotifications');
    expect(started[1]?.serviceName).toBe('tip.TipHub');
    expect(finished[0]?.stream).toBe(false);
    expect(finished[1]?.stream).toBe(true);

    const nextFail = async () => {
      throw new Error('stream boom');
    };
    let caught: unknown;
    try {
      await intercept(nextFail as any)(makeReq(true) as any);
    } catch (err) {
      caught = err;
    }
    expect(caught instanceof Error && caught.message === 'stream boom').toBe(true);
    expect(errored).toHaveLength(1);
    expect(errored[0]?.stream).toBe(true);
    expect(finished).toHaveLength(3);
    expect(finished[2]?.stream).toBe(true);

    let unaryCaught: unknown;
    try {
      await intercept(nextFail as any)(makeReq(false) as any);
    } catch (err) {
      unaryCaught = err;
    }
    expect(unaryCaught instanceof Error && unaryCaught.message === 'stream boom').toBe(true);
    expect(errored).toHaveLength(2);
    expect(errored[1]?.stream).toBe(false);
    expect(finished).toHaveLength(4);
    expect(finished[3]?.stream).toBe(false);
  } finally {
    setLifecycleProvider(null);
  }
});
