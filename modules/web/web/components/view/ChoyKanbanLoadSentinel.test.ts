// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { nextTick } from 'vue';
import ChoyKanbanLoadSentinel from './ChoyKanbanLoadSentinel.vue';

type SentinelSetup = {
  isArmed: () => boolean;
  handleIntersectionForTest: (intersecting: boolean) => void;
};

describe('ChoyKanbanLoadSentinel', () => {
  test('connects an IntersectionObserver and re-arms from its callback', async () => {
    const originals = globalThis.IntersectionObserver;
    const seen: boolean[] = [];
    class FakeObserver {
      static last: FakeObserver | null = null;
      callback: (entries: Array<{ isIntersecting: boolean }>) => void;
      constructor(callback: (entries: Array<{ isIntersecting: boolean }>) => void) {
        this.callback = callback;
        FakeObserver.last = this;
      }
      observe() {}
      disconnect() {}
      emit(intersecting: boolean) {
        this.callback([{ isIntersecting: intersecting }]);
      }
    }
    (globalThis as any).IntersectionObserver = FakeObserver;
    try {
      const visibles: number[] = [];
      const mounted = mountApp(ChoyKanbanLoadSentinel as any, {
        props: { disabled: false },
        reactiveProps: true,
        on: { onVisible: () => visibles.push(1) },
      });
      await flushPromises();
      const observer = FakeObserver.last;
      expect(observer).toBeTruthy();
      observer!.emit(false);
      observer!.emit(true);
      observer!.emit(true);
      expect(visibles.length).toBe(1);
      seen.push(true);

      mounted.props.disabled = true;
      await nextTick();
      mounted.props.disabled = false;
      await nextTick();
      mounted.unmount();
    } finally {
      if (originals) (globalThis as any).IntersectionObserver = originals;
      else delete (globalThis as any).IntersectionObserver;
    }
    expect(seen).toEqual([true]);
  });

  test('emits once per intersecting stay; re-arms after leave', async () => {
    const visibles: number[] = [];
    const mounted = mountApp(ChoyKanbanLoadSentinel as any, {
      props: { disabled: false },
      on: {
        onVisible: () => visibles.push(1),
      },
    });
    await flushPromises();
    const state = mounted.root as SentinelSetup;

    state.handleIntersectionForTest(true);
    state.handleIntersectionForTest(true);
    expect(visibles.length).toBe(1);
    expect(state.isArmed()).toBe(false);

    state.handleIntersectionForTest(false);
    expect(state.isArmed()).toBe(true);
    state.handleIntersectionForTest(true);
    expect(visibles.length).toBe(2);

    mounted.unmount();
  });

  test('disabled toggle does not burst-emit without re-arm cooldown', async () => {
    const visibles: number[] = [];
    const mounted = mountApp(ChoyKanbanLoadSentinel as any, {
      props: { disabled: false },
      reactiveProps: true,
      on: {
        onVisible: () => visibles.push(1),
      },
    });
    await flushPromises();
    const state = mounted.root as SentinelSetup;

    state.handleIntersectionForTest(true);
    expect(visibles.length).toBe(1);

    // Simulate load-more busy: still intersecting, disarmed.
    mounted.props.disabled = true;
    await nextTick();
    state.handleIntersectionForTest(true);
    expect(visibles.length).toBe(1);

    // Busy clears: must not emit again in the same tick.
    mounted.props.disabled = false;
    await nextTick();
    state.handleIntersectionForTest(true);
    expect(visibles.length).toBe(1);
    expect(state.isArmed()).toBe(false);

    // After cooldown re-arm, one more emit is allowed while still intersecting.
    await new Promise<void>((resolve) => setTimeout(resolve, 300));
    await flushPromises();
    expect(visibles.length).toBe(2);

    mounted.unmount();
  });
});
