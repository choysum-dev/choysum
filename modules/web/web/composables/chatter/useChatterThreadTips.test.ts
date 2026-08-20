// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope, ref } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const onTips = vi.fn();
const subscribeThread = vi.fn(() => ({}));

vi.mock('@/core/web/tip', () => ({
  onTips: (...args: unknown[]) => onTips(...args),
  subscribeThread: (...args: unknown[]) => subscribeThread(...args),
}));

import { useChatterThreadTips } from './useChatterThreadTips';

describe('useChatterThreadTips', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    onTips.mockReset();
    subscribeThread.mockReset();
    subscribeThread.mockReturnValue({});
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('starts poll fallback when the tip stream ends without abort', async () => {
    onTips.mockResolvedValue(undefined);
    const refresh = vi.fn(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh));
    await Promise.resolve();
    await Promise.resolve();
    refresh.mockClear();
    await vi.advanceTimersByTimeAsync(30_000);
    expect(refresh).toHaveBeenCalled();
    scope.stop();
  });

  it('starts poll fallback when the tip stream rejects', async () => {
    onTips.mockRejectedValue(new Error('stream down'));
    const refresh = vi.fn(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh));
    await Promise.resolve();
    await Promise.resolve();
    refresh.mockClear();
    await vi.advanceTimersByTimeAsync(30_000);
    expect(refresh).toHaveBeenCalled();
    scope.stop();
  });
});
