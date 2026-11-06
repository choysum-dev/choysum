// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CreateWebClient } from '../rpc/client_factory';
import { TipHub, type Tip } from './pb/tip_pb';

type TipCallOptions = { signal?: AbortSignal };

/** Locator for SubscribeModuleOp: prefer operationId; jobId remains transitional. */
export type SubscribeModuleOpLocator = {
  operationId?: string;
  jobId?: string;
};

export type TipHubClient = {
  subscribeThread(req: { model: string; resId: string }, options?: TipCallOptions): AsyncIterable<Tip>;
  subscribeNotifications(req?: object, options?: TipCallOptions): AsyncIterable<Tip>;
  subscribeModuleOp(req: SubscribeModuleOpLocator, options?: TipCallOptions): AsyncIterable<Tip>;
};

export type TipClientDeps = {
  /** Injected hub (tests / alternate transports). Default: CreateWebClient(TipHub). */
  hub?: TipHubClient;
};

export type TipClient = {
  subscribeThread(model: string, resId: string, signal?: AbortSignal): AsyncIterable<Tip>;
  subscribeNotifications(signal?: AbortSignal): AsyncIterable<Tip>;
  /**
   * Subscribe to meta.module_op.changed tips.
   * Prefer `{ operationId }`. A bare string is treated as transitional `jobId`.
   */
  subscribeModuleOp(locator: string | SubscribeModuleOpLocator, signal?: AbortSignal): AsyncIterable<Tip>;
};

function callOptions(signal?: AbortSignal): TipCallOptions | undefined {
  if (signal == null) {
    return undefined;
  }
  return { signal };
}

function normalizeModuleOpLocator(locator: string | SubscribeModuleOpLocator): SubscribeModuleOpLocator {
  if (typeof locator === 'string') {
    return { jobId: locator };
  }
  const out: SubscribeModuleOpLocator = {};
  const operationId = String(locator?.operationId || '').trim();
  const jobId = String(locator?.jobId || '').trim();
  if (operationId) {
    out.operationId = operationId;
  }
  if (jobId) {
    out.jobId = jobId;
  }
  return out;
}

/** Composable TipHub API. Pass `hub` in tests instead of mocking CreateWebClient. */
export function createTipClient(deps: TipClientDeps = {}): TipClient {
  const defaultHub = CreateWebClient(TipHub);
  const hub = (): TipHubClient => deps.hub ?? (defaultHub() as unknown as TipHubClient);

  return {
    subscribeThread(model, resId, signal) {
      return hub().subscribeThread({ model, resId }, callOptions(signal));
    },
    subscribeNotifications(signal) {
      return hub().subscribeNotifications({}, callOptions(signal));
    },
    subscribeModuleOp(locator, signal) {
      return hub().subscribeModuleOp(normalizeModuleOpLocator(locator), callOptions(signal));
    },
  };
}

const defaultTipClient = createTipClient();

export const subscribeThread = defaultTipClient.subscribeThread;
export const subscribeNotifications = defaultTipClient.subscribeNotifications;
export const subscribeModuleOp = defaultTipClient.subscribeModuleOp;

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
