// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import {
  peekSearchFirstFrameGateForTests,
  resetSearchFirstFrameRegistryForTests,
  useSearchFirstFrameHost,
  useSearchFirstFrameReporter,
} from './searchFirstFrame';

afterEach(() => {
  resetSearchFirstFrameRegistryForTests();
});

test('host and reporter join the same storeId gate across slot boundaries', async () => {
  const store = { storeId: 'demo.Task' };
  let hostDefer = false;
  let reporterMark: (() => void) | null = null;

  const Host = defineComponent({
    name: 'FirstFrameHost',
    setup() {
      const gate = useSearchFirstFrameHost(store, { expectFirstFrame: true });
      hostDefer = gate.shouldDefer.value;
      return () => h('div', [h(Reporter)]);
    },
  });
  const Reporter = defineComponent({
    name: 'FirstFrameReporter',
    setup() {
      const api = useSearchFirstFrameReporter(store, { initialEmit: true });
      reporterMark = api.markEmitted;
      return () => h('div');
    },
  });

  const { unmount } = mountApp(Host);
  await flushPromises();
  expect(hostDefer).toBe(true);
  expect(peekSearchFirstFrameGateForTests('demo.Task')?.phase.value).toBe('pending');
  reporterMark!();
  expect(peekSearchFirstFrameGateForTests('demo.Task')?.phase.value).toBe('emitted');
  expect(peekSearchFirstFrameGateForTests('demo.Task')?.shouldDefer.value).toBe(true);
  unmount();
  await flushPromises();
  expect(peekSearchFirstFrameGateForTests('demo.Task')).toBeNull();
});

test('reporter with initialEmit=false clears defer', async () => {
  const store = { storeId: 'demo.NoEmit' };
  let gateDefer = true;
  const Host = defineComponent({
    setup() {
      const gate = useSearchFirstFrameHost(store, { expectFirstFrame: true });
      const Child = defineComponent({
        setup() {
          useSearchFirstFrameReporter(store, { initialEmit: false });
          gateDefer = gate.shouldDefer.value;
          return () => h('div');
        },
      });
      return () => h(Child);
    },
  });
  const { unmount } = mountApp(Host);
  await flushPromises();
  expect(gateDefer).toBe(false);
  unmount();
});

test('host and reporter join via WeakMap when storeId is missing', async () => {
  const store = {} as { storeId?: string };
  let hostGate: ReturnType<typeof useSearchFirstFrameHost> | null = null;
  let reporterMark: (() => void) | null = null;

  const Host = defineComponent({
    setup() {
      hostGate = useSearchFirstFrameHost(store, { expectFirstFrame: true });
      const Reporter = defineComponent({
        setup() {
          const api = useSearchFirstFrameReporter(store, { initialEmit: true });
          reporterMark = api.markEmitted;
          return () => h('div');
        },
      });
      return () => h('div', [h(Reporter)]);
    },
  });

  const { unmount } = mountApp(Host);
  await flushPromises();
  expect(hostGate!.shouldDefer.value).toBe(true);
  expect(hostGate!.phase.value).toBe('pending');
  reporterMark!();
  expect(hostGate!.phase.value).toBe('emitted');
  expect(hostGate!.shouldDefer.value).toBe(true);
  unmount();
});

test('null store uses a private gate; setExpectFirstFrame can clear pending', async () => {
  let gate: ReturnType<typeof useSearchFirstFrameHost> | null = null;
  const Host = defineComponent({
    setup() {
      gate = useSearchFirstFrameHost(null, { expectFirstFrame: true });
      return () => h('div');
    },
  });
  const { unmount } = mountApp(Host);
  await flushPromises();
  expect(gate!.phase.value).toBe('pending');
  gate!.setExpectFirstFrame(false);
  expect(gate!.phase.value).toBe('none');
  gate!.setExpectFirstFrame(true);
  expect(gate!.phase.value).toBe('pending');
  unmount();
});
