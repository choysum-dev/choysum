// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../rpc/client_factory', () => {
  const subscribeThread = vi.fn(async function* () {
    yield { topic: 'message.thread.changed', model: 'message.thread', resId: '42' };
  });
  const subscribeNotifications = vi.fn(async function* () {
    yield { topic: 'message.notification.user', userId: 'u1' };
  });
  return {
    CreateWebClient: () => () => ({ subscribeThread, subscribeNotifications }),
  };
});

import { onTips, subscribeNotifications, subscribeThread } from './client';

describe('core/web tip TipHub client', () => {
  it('subscribes through CreateWebClient and yields tips', async () => {
    const tips: Array<{ topic: string; resId?: string; userId?: string }> = [];
    await onTips(subscribeThread('message.thread', '42'), async (tip) => {
      tips.push(tip);
    });
    await onTips(subscribeNotifications(), async (tip) => {
      tips.push(tip);
    });

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

  it('does not use CreateWebApiService', () => {
    const src = readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'client.ts'), 'utf8');
    expect(src).toContain('CreateWebClient');
    expect(src).not.toContain('CreateWebApiService');
  });
});
