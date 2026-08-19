// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CreateWebClient } from '../rpc/client_factory';
import { TipHub, type Tip } from './pb/tip_pb';

type TipCallOptions = { signal?: AbortSignal };

type TipHubClient = {
  subscribeThread(req: { model: string; resId: string }, options?: TipCallOptions): AsyncIterable<Tip>;
  subscribeNotifications(req?: object, options?: TipCallOptions): AsyncIterable<Tip>;
};

const tipHubClient = CreateWebClient(TipHub);

function tipHub(): TipHubClient {
  return tipHubClient() as unknown as TipHubClient;
}

function callOptions(signal?: AbortSignal): TipCallOptions | undefined {
  if (signal == null) {
    return undefined;
  }
  return { signal };
}

export function subscribeThread(model: string, resId: string, signal?: AbortSignal): AsyncIterable<Tip> {
  return tipHub().subscribeThread({ model, resId }, callOptions(signal));
}

export function subscribeNotifications(signal?: AbortSignal): AsyncIterable<Tip> {
  return tipHub().subscribeNotifications({}, callOptions(signal));
}

export async function onTips(
  tips: AsyncIterable<Tip>,
  refresh: (tip: Tip) => void | Promise<void>,
  signal?: AbortSignal,
): Promise<void> {
  const iterator = tips[Symbol.asyncIterator]();
  const cancelIterator = () => {
    void Promise.resolve(iterator.return?.()).catch(() => undefined);
  };
  signal?.addEventListener('abort', cancelIterator, { once: true });
  try {
    if (signal?.aborted) {
      cancelIterator();
      return;
    }
    while (true) {
      const next = await iterator.next();
      if (next.done) {
        return;
      }
      if (signal?.aborted) {
        return;
      }
      await refresh(next.value);
    }
  } catch (err) {
    if (signal?.aborted) {
      return;
    }
    throw err;
  } finally {
    signal?.removeEventListener('abort', cancelIterator);
  }
}

export { TipHub, type Tip };
