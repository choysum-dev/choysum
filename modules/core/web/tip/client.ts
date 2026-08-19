// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CreateWebClient } from '../rpc/client_factory';
import { TipHub, type Tip } from './pb/tip_pb';

type TipHubClient = {
  subscribeThread(req: { model: string; resId: string }): AsyncIterable<Tip>;
  subscribeNotifications(req?: object): AsyncIterable<Tip>;
};

const tipHubClient = CreateWebClient(TipHub);

function tipHub(): TipHubClient {
  return tipHubClient() as unknown as TipHubClient;
}

export function subscribeThread(model: string, resId: string): AsyncIterable<Tip> {
  return tipHub().subscribeThread({ model, resId });
}

export function subscribeNotifications(): AsyncIterable<Tip> {
  return tipHub().subscribeNotifications({});
}

export async function onTips(
  tips: AsyncIterable<Tip>,
  refresh: (tip: Tip) => void | Promise<void>,
  signal?: AbortSignal,
): Promise<void> {
  for await (const tip of tips) {
    if (signal?.aborted) {
      return;
    }
    await refresh(tip);
  }
}

export { TipHub, type Tip };
