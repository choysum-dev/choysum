// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, effectScope, h, ref } from 'vue';

import { fnRecorder, flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useChatterThreadTips } from './useChatterThreadTips';

const POLL_MS = 25;

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

describe('useChatterThreadTips', () => {
  const onTips = fnRecorder(async () => undefined);
  const subscribeThread = fnRecorder(() => ({} as any));

  function tipDeps() {
    return {
      onTips: onTips as any,
      subscribeThread: subscribeThread as any,
      pollFallbackMs: POLL_MS,
    };
  }

  beforeEach(() => {
    onTips.mockReset();
    onTips.mockImplementation(async () => undefined);
    subscribeThread.mockReset();
    subscribeThread.mockImplementation(() => ({} as any));
  });

  test('starts poll fallback when the tip stream ends without abort', async () => {
    onTips.mockImplementation(async () => undefined);
    const refresh = fnRecorder(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh, tipDeps()));
    await flushPromises();
    refresh.mockClear();
    await sleep(POLL_MS + 20);
    expect(refresh.calls.length).toBeGreaterThan(0);
    scope.stop();
  });

  test('starts poll fallback when the tip stream rejects', async () => {
    onTips.mockImplementation(async () => {
      throw new Error('stream down');
    });
    const refresh = fnRecorder(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh, tipDeps()));
    await flushPromises();
    refresh.mockClear();
    await sleep(POLL_MS + 20);
    expect(refresh.calls.length).toBeGreaterThan(0);
    scope.stop();
  });

  test('skips tips when the thread identity is empty', async () => {
    const refresh = fnRecorder(async () => undefined);
    const model = ref('  ');
    const resId = ref<string | undefined>('  ');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh, tipDeps()));
    await flushPromises();
    expect(onTips.calls.length).toBe(0);
    scope.stop();
  });

  test('refreshes when thread tips fire', async () => {
    onTips.mockImplementation(async (_stream, callback: () => Promise<void>) => {
      await callback();
    });
    const refresh = fnRecorder(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh, tipDeps()));
    await flushPromises();
    expect(refresh.calls.length).toBeGreaterThan(0);
    scope.stop();
  });

  test('stops polling when the thread identity becomes empty', async () => {
    onTips.mockImplementation(async () => undefined);
    const refresh = fnRecorder(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh, tipDeps()));
    await flushPromises();
    model.value = '';
    resId.value = '';
    await flushPromises();
    refresh.mockClear();
    await sleep(POLL_MS + 20);
    expect(refresh.calls.length).toBe(0);
    scope.stop();
  });

  test('stops tips when the host component unmounts', async () => {
    onTips.mockImplementation(async () => undefined);
    const refresh = fnRecorder(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const Host = defineComponent({
      setup() {
        useChatterThreadTips(model, resId, refresh, tipDeps());
        return () => h('div');
      },
    });
    const mounted = mountApp(Host);
    await flushPromises();
    refresh.mockClear();
    mounted.unmount();
    await sleep(POLL_MS + 20);
    expect(refresh.calls.length).toBe(0);
  });

  test('does not start poll fallback when a tip stream is aborted by a newer subscription', async () => {
    let resolveFirst: (() => void) | undefined;
    let call = 0;
    onTips.mockImplementation(() => {
      call += 1;
      if (call === 1) {
        return new Promise<void>(resolve => {
          resolveFirst = resolve;
        });
      }
      return Promise.resolve(undefined);
    });
    const refresh = fnRecorder(async () => undefined);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    scope.run(() => useChatterThreadTips(model, resId, refresh, tipDeps()));
    await flushPromises();
    resId.value = 'r2';
    await flushPromises();
    resolveFirst?.();
    await flushPromises();
    refresh.mockClear();
    await sleep(POLL_MS + 20);
    // Only the second (non-aborted) subscription should poll.
    expect(refresh.calls.length).toBeGreaterThan(0);
    scope.stop();
  });
});
