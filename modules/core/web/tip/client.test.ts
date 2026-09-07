// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTipClient, onTips, type TipHubClient } from './client';
import { ensureAbortController } from '../testing/qjs_polyfills';

ensureAbortController();

type Call = { args: unknown[] };

function makeAsyncGenFn(yieldValue: unknown) {
  const calls: Call[] = [];
  const fn = async function* (...args: unknown[]) {
    calls.push({ args });
    yield yieldValue;
  };
  return { fn, calls };
}

function installClient() {
  const subscribeThreadFn = makeAsyncGenFn({ topic: 'message.thread.changed', model: 'message.thread', resId: '42' });
  const subscribeNotificationsFn = makeAsyncGenFn({ topic: 'message.notification.user', userId: 'u1' });
  const subscribeModuleOpFn = makeAsyncGenFn({ topic: 'meta.module_op.changed', resId: 'job-1', userId: 'u1' });
  const hub: TipHubClient = {
    subscribeThread: subscribeThreadFn.fn as any,
    subscribeNotifications: subscribeNotificationsFn.fn as any,
    subscribeModuleOp: subscribeModuleOpFn.fn as any,
  };
  const client = createTipClient({ hub });
  return { client, subscribeThreadFn, subscribeNotificationsFn, subscribeModuleOpFn };
}

test('core/web tip TipHub client: subscribes through CreateWebClient and yields tips', async () => {
  const { client, subscribeThreadFn, subscribeNotificationsFn, subscribeModuleOpFn } = installClient();
  const tips: Array<{ topic: string; resId?: string; userId?: string }> = [];
  const signal = new AbortController().signal;
  await onTips(client.subscribeThread('message.thread', '42', signal), async tip => {
    tips.push(tip);
  });
  await onTips(client.subscribeNotifications(signal), async tip => {
    tips.push(tip);
  });
  await onTips(client.subscribeModuleOp('job-1', signal), async tip => {
    tips.push(tip);
  });

  expect(subscribeThreadFn.calls[0]?.args).toEqual([{ model: 'message.thread', resId: '42' }, { signal }]);
  expect(subscribeNotificationsFn.calls[0]?.args).toEqual([{}, { signal }]);
  expect(subscribeModuleOpFn.calls[0]?.args).toEqual([{ jobId: 'job-1' }, { signal }]);
  expect(tips.map(tip => tip.topic)).toEqual([
    'message.thread.changed',
    'message.notification.user',
    'meta.module_op.changed',
  ]);
  expect(tips[0]?.resId).toBe('42');
  expect(tips[1]?.userId).toBe('u1');
  expect(tips[2]?.resId).toBe('job-1');
});

test('core/web tip TipHub client: omits CallOptions when no abort signal is provided', async () => {
  const { client, subscribeThreadFn, subscribeNotificationsFn, subscribeModuleOpFn } = installClient();
  await onTips(client.subscribeThread('message.thread', '7'), async () => {});
  await onTips(client.subscribeNotifications(), async () => {});
  await onTips(client.subscribeModuleOp('job-7'), async () => {});

  expect(subscribeThreadFn.calls[subscribeThreadFn.calls.length - 1]?.args).toEqual([
    { model: 'message.thread', resId: '7' },
    undefined,
  ]);
  expect(subscribeNotificationsFn.calls[subscribeNotificationsFn.calls.length - 1]?.args).toEqual([{}, undefined]);
  expect(subscribeModuleOpFn.calls[subscribeModuleOpFn.calls.length - 1]?.args).toEqual([{ jobId: 'job-7' }, undefined]);
});

test('core/web tip TipHub client: stops onTips when the abort signal is already aborted', async () => {
  let refreshCalls = 0;
  const refresh = () => {
    refreshCalls += 1;
  };
  const controller = new AbortController();
  controller.abort();

  await onTips(
    (async function* () {
      yield { topic: 'message.thread.changed' } as never;
    })(),
    refresh,
    controller.signal,
  );

  expect(refreshCalls).toBe(0);
});

test('core/web tip TipHub client: cancels an idle iterator when the abort signal fires', async () => {
  let refreshCalls = 0;
  const refresh = () => {
    refreshCalls += 1;
  };
  const controller = new AbortController();
  let returned = false;
  let settleNext: ((result: IteratorResult<{ topic: string }>) => void) | undefined;
  const tips: AsyncIterable<{ topic: string }> = {
    [Symbol.asyncIterator]() {
      return {
        next: () =>
          new Promise(resolve => {
            settleNext = resolve;
          }),
        return: () => {
          returned = true;
          settleNext?.({ done: true, value: undefined });
          return Promise.resolve({ done: true, value: undefined });
        },
      };
    },
  };

  const done = onTips(tips as never, refresh, controller.signal);
  await Promise.resolve();
  controller.abort();
  await done;

  expect(returned).toBe(true);
  expect(refreshCalls).toBe(0);
});

test('core/web tip TipHub client: cancels cleanup when iterator.return is missing', async () => {
  let refreshCalls = 0;
  const refresh = () => {
    refreshCalls += 1;
  };
  const controller = new AbortController();
  controller.abort();
  const tips: AsyncIterable<{ topic: string }> = {
    [Symbol.asyncIterator]() {
      return {
        next: async () => ({ done: false, value: { topic: 'message.thread.changed' } }),
      };
    },
  };

  await onTips(tips as never, refresh, controller.signal);
  expect(refreshCalls).toBe(0);
});

test('core/web tip TipHub client: ignores rejected iterator cleanup on abort', async () => {
  let refreshCalls = 0;
  const refresh = () => {
    refreshCalls += 1;
  };
  const controller = new AbortController();
  let settleNext: ((result: IteratorResult<{ topic: string }>) => void) | undefined;
  const tips: AsyncIterable<{ topic: string }> = {
    [Symbol.asyncIterator]() {
      return {
        next: () =>
          new Promise(resolve => {
            settleNext = resolve;
          }),
        return: () => {
          settleNext?.({ done: true, value: undefined });
          return Promise.reject(new Error('close failed'));
        },
      };
    },
  };

  const done = onTips(tips as never, refresh, controller.signal);
  await Promise.resolve();
  controller.abort();
  await done;
  expect(refreshCalls).toBe(0);
});

test('core/web tip TipHub client: stops after a received tip when aborted during refresh', async () => {
  const controller = new AbortController();
  let refreshCalls = 0;
  const refresh = async () => {
    refreshCalls += 1;
    controller.abort();
  };

  await onTips(
    (async function* () {
      yield { topic: 'message.thread.changed' } as never;
      yield { topic: 'should-not-refresh' } as never;
    })(),
    refresh,
    controller.signal,
  );

  expect(refreshCalls).toBe(1);
});

test('core/web tip TipHub client: skips refresh when aborted after a tip arrives', async () => {
  let refreshCalls = 0;
  const refresh = () => {
    refreshCalls += 1;
  };
  const controller = new AbortController();
  const tips: AsyncIterable<{ topic: string }> = {
    [Symbol.asyncIterator]() {
      return {
        next: async () => {
          controller.abort();
          return { done: false, value: { topic: 'message.thread.changed' } };
        },
        return: () => Promise.resolve({ done: true, value: undefined }),
      };
    },
  };

  await onTips(tips as never, refresh, controller.signal);
  expect(refreshCalls).toBe(0);
});

test('core/web tip TipHub client: swallows iterator errors after abort', async () => {
  let refreshCalls = 0;
  const refresh = () => {
    refreshCalls += 1;
  };
  const controller = new AbortController();
  const tips: AsyncIterable<{ topic: string }> = {
    [Symbol.asyncIterator]() {
      return {
        next: () =>
          new Promise((_, reject) => {
            controller.signal.addEventListener('abort', () => reject(new Error('aborted')), { once: true });
          }),
        return: () => Promise.resolve({ done: true, value: undefined }),
      };
    },
  };

  const done = onTips(tips as never, refresh, controller.signal);
  await Promise.resolve();
  controller.abort();
  await done;
  expect(refreshCalls).toBe(0);
});

test('core/web tip TipHub client: rethrows iterator failures when not aborted', async () => {
  await expectRejects(
    () =>
      onTips(
        (async function* () {
          throw new Error('boom');
        })(),
        async () => {},
      ),
    'boom',
  );
});
