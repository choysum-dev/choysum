// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, effectScope, h, ref } from 'vue';
import { mount } from '@vue/test-utils';
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

  it('skips tips when the thread identity is empty', async () => {
    const refresh = vi.fn(async () => undefined);
    const model = ref('  ');
    const resId = ref<string | undefined>('  ');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh));
    await Promise.resolve();
    expect(onTips).not.toHaveBeenCalled();
    scope.stop();
  });

  it('refreshes when thread tips fire', async () => {
    onTips.mockImplementation(async (_stream, callback: () => Promise<void>) => {
      await callback();
    });
    const refresh = vi.fn(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh));
    await Promise.resolve();
    expect(refresh).toHaveBeenCalled();
    scope.stop();
  });

  it('stops polling when the thread identity becomes empty', async () => {
    onTips.mockResolvedValue(undefined);
    const refresh = vi.fn(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh));
    await Promise.resolve();
    await Promise.resolve();
    model.value = '';
    resId.value = '';
    await Promise.resolve();
    refresh.mockClear();
    await vi.advanceTimersByTimeAsync(30_000);
    expect(refresh).not.toHaveBeenCalled();
    scope.stop();
  });

  it('stops tips when the host component unmounts', async () => {
    onTips.mockResolvedValue(undefined);
    const refresh = vi.fn(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const Host = defineComponent({
      setup() {
        useChatterThreadTips(model, resId, refresh);
        return () => h('div');
      },
    });
    const wrapper = mount(Host);
    await Promise.resolve();
    await Promise.resolve();
    refresh.mockClear();
    wrapper.unmount();
    await vi.advanceTimersByTimeAsync(30_000);
    expect(refresh).not.toHaveBeenCalled();
  });
});
