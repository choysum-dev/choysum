// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  subscribeThread: vi.fn(async function* () {
    yield { topic: 'message.thread.changed', model: 'message.thread', resId: '42' };
  }),
  subscribeNotifications: vi.fn(async function* () {
    yield { topic: 'message.notification.user', userId: 'u1' };
  }),
}));

vi.mock('../rpc/client_factory', () => ({
  CreateWebClient: () => () => ({
    subscribeThread: mocks.subscribeThread,
    subscribeNotifications: mocks.subscribeNotifications,
  }),
}));

import { onTips, subscribeNotifications, subscribeThread } from './client';

describe('core/web tip TipHub client', () => {
  it('subscribes through CreateWebClient and yields tips', async () => {
    const tips: Array<{ topic: string; resId?: string; userId?: string }> = [];
    const signal = new AbortController().signal;
    await onTips(subscribeThread('message.thread', '42', signal), async (tip) => {
      tips.push(tip);
    });
    await onTips(subscribeNotifications(signal), async (tip) => {
      tips.push(tip);
    });

    expect(mocks.subscribeThread).toHaveBeenCalledWith({ model: 'message.thread', resId: '42' }, { signal });
    expect(mocks.subscribeNotifications).toHaveBeenCalledWith({}, { signal });
    expect(tips.map((tip) => tip.topic)).toEqual(['message.thread.changed', 'message.notification.user']);
    expect(tips[0]?.resId).toBe('42');
    expect(tips[1]?.userId).toBe('u1');
  });

  it('stops onTips when the abort signal is already aborted', async () => {
    const refresh = vi.fn();
    const controller = new AbortController();
    controller.abort();

    await onTips(
      (async function* () {
        yield { topic: 'message.thread.changed' } as never;
      })(),
      refresh,
      controller.signal,
    );

    expect(refresh).not.toHaveBeenCalled();
  });

  it('cancels an idle iterator when the abort signal fires', async () => {
    const refresh = vi.fn();
    const controller = new AbortController();
    let returned = false;
    let settleNext: ((result: IteratorResult<{ topic: string }>) => void) | undefined;
    const tips: AsyncIterable<{ topic: string }> = {
      [Symbol.asyncIterator]() {
        return {
          next: () =>
            new Promise((resolve) => {
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
    expect(refresh).not.toHaveBeenCalled();
  });

  it('swallows iterator errors after abort', async () => {
    const refresh = vi.fn();
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
    expect(refresh).not.toHaveBeenCalled();
  });

  it('rethrows iterator failures when not aborted', async () => {
    await expect(
      onTips(
        (async function* () {
          throw new Error('boom');
        })(),
        async () => {},
      ),
    ).rejects.toThrow('boom');
  });

  it('does not use CreateWebApiService', () => {
    const src = readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'client.ts'), 'utf8');
    expect(src).toContain('CreateWebClient');
    expect(src).not.toContain('CreateWebApiService');
  });
});
