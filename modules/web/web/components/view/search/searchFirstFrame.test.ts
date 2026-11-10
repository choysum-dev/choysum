// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, ref } from 'vue';
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

test('second host acquire updates expect on an existing registry entry', async () => {
  const store = { storeId: 'demo.SecondHost' };
  let firstGate: ReturnType<typeof useSearchFirstFrameHost> | null = null;
  let secondGate: ReturnType<typeof useSearchFirstFrameHost> | null = null;

  // Reporter joins first so the registry entry exists before the second host.
  const ReporterFirst = defineComponent({
    setup() {
      useSearchFirstFrameReporter(store, { initialEmit: false });
      const HostA = defineComponent({
        setup() {
          firstGate = useSearchFirstFrameHost(store, { expectFirstFrame: false });
          return () => h('div');
        },
      });
      const HostB = defineComponent({
        setup() {
          secondGate = useSearchFirstFrameHost(store, { expectFirstFrame: true });
          return () => h('div');
        },
      });
      return () => h('div', [h(HostA), h(HostB)]);
    },
  });

  const { unmount } = mountApp(ReporterFirst);
  await flushPromises();
  expect(firstGate).not.toBeNull();
  expect(secondGate).toBe(firstGate);
  expect(secondGate!.phase.value).toBe('pending');
  expect(secondGate!.shouldDefer.value).toBe(true);
  unmount();
});

test('non-expecting host does not clear another host pending first-frame', async () => {
  const store = { storeId: 'demo.MultiHostExpect' };
  let expectingGate: ReturnType<typeof useSearchFirstFrameHost> | null = null;

  const Root = defineComponent({
    setup() {
      const HostExpect = defineComponent({
        setup() {
          expectingGate = useSearchFirstFrameHost(store, { expectFirstFrame: true });
          return () => h('div');
        },
      });
      const HostNoSearch = defineComponent({
        setup() {
          useSearchFirstFrameHost(store, { expectFirstFrame: false });
          return () => h('div');
        },
      });
      return () => h('div', [h(HostExpect), h(HostNoSearch)]);
    },
  });

  const { unmount } = mountApp(Root);
  await flushPromises();
  expect(expectingGate!.phase.value).toBe('pending');
  expect(expectingGate!.shouldDefer.value).toBe(true);
  unmount();
});

test('host expect vote flips while another host still expects', async () => {
  const store = { storeId: 'demo.ExpectVote' };
  const expectA = ref(true);
  let gate: ReturnType<typeof useSearchFirstFrameHost> | null = null;

  const Root = defineComponent({
    setup() {
      const HostA = defineComponent({
        setup() {
          gate = useSearchFirstFrameHost(store, { expectFirstFrame: () => expectA.value });
          return () => h('div');
        },
      });
      const HostB = defineComponent({
        setup() {
          useSearchFirstFrameHost(store, { expectFirstFrame: true });
          return () => h('div');
        },
      });
      return () => h('div', [h(HostA), h(HostB)]);
    },
  });

  const { unmount } = mountApp(Root);
  await flushPromises();
  expect(gate!.phase.value).toBe('pending');

  // Flip one host off; another host still expects — stay pending.
  expectA.value = false;
  await flushPromises();
  expect(peekSearchFirstFrameGateForTests('demo.ExpectVote')?.phase.value).toBe('pending');

  // Raise back to cover the true branch of updateHostExpectVote.
  expectA.value = true;
  await flushPromises();
  expect(peekSearchFirstFrameGateForTests('demo.ExpectVote')?.phase.value).toBe('pending');

  unmount();
  await flushPromises();
  expect(peekSearchFirstFrameGateForTests('demo.ExpectVote')).toBeNull();
});

test('sole host expect vote clear sets phase to none', async () => {
  const store = { storeId: 'demo.SoleExpectVote' };
  const expectFF = ref(true);
  let gate: ReturnType<typeof useSearchFirstFrameHost> | null = null;

  const Host = defineComponent({
    setup() {
      gate = useSearchFirstFrameHost(store, { expectFirstFrame: () => expectFF.value });
      return () => h('div');
    },
  });

  const { unmount } = mountApp(Host);
  await flushPromises();
  expect(gate!.phase.value).toBe('pending');

  expectFF.value = false;
  await flushPromises();
  expect(gate!.phase.value).toBe('none');
  expect(gate!.shouldDefer.value).toBe(false);

  unmount();
});

test('flip expect off then unmount does not double-decrement sibling host vote', async () => {
  const store = { storeId: 'demo.NoDoubleDec' };
  const expectA = ref(true);
  const showA = ref(true);

  const Root = defineComponent({
    setup() {
      const HostA = defineComponent({
        setup() {
          useSearchFirstFrameHost(store, { expectFirstFrame: () => expectA.value });
          return () => h('div');
        },
      });
      const HostB = defineComponent({
        setup() {
          useSearchFirstFrameHost(store, { expectFirstFrame: true });
          return () => h('div');
        },
      });
      return () => h('div', [showA.value ? h(HostA) : null, h(HostB)]);
    },
  });

  const { unmount } = mountApp(Root);
  await flushPromises();
  expect(peekSearchFirstFrameGateForTests('demo.NoDoubleDec')?.phase.value).toBe('pending');

  // Vote down via watch first; dispose must not decrement again.
  expectA.value = false;
  await flushPromises();
  expect(peekSearchFirstFrameGateForTests('demo.NoDoubleDec')?.phase.value).toBe('pending');

  showA.value = false;
  await flushPromises();
  expect(peekSearchFirstFrameGateForTests('demo.NoDoubleDec')?.phase.value).toBe('pending');
  expect(peekSearchFirstFrameGateForTests('demo.NoDoubleDec')?.shouldDefer.value).toBe(true);

  unmount();
});

test('null-store host watch updates private gate expectation', async () => {
  const expectFF = ref(true);
  let gate: ReturnType<typeof useSearchFirstFrameHost> | null = null;

  const Host = defineComponent({
    setup() {
      gate = useSearchFirstFrameHost(null, { expectFirstFrame: () => expectFF.value });
      return () => h('div');
    },
  });

  const { unmount } = mountApp(Host);
  await flushPromises();
  expect(gate!.phase.value).toBe('pending');
  expectFF.value = false;
  await flushPromises();
  expect(gate!.phase.value).toBe('none');
  expectFF.value = true;
  await flushPromises();
  expect(gate!.phase.value).toBe('pending');
  unmount();
});

test('unmounting the last expecting host clears pending while a non-expecting host remains', async () => {
  const store = { storeId: 'demo.ReleaseExpect' };
  const showExpecting = ref(true);
  let sharedGate: ReturnType<typeof useSearchFirstFrameHost> | null = null;

  const Root = defineComponent({
    setup() {
      const HostExpect = defineComponent({
        setup() {
          sharedGate = useSearchFirstFrameHost(store, { expectFirstFrame: true });
          return () => h('div');
        },
      });
      const HostIdle = defineComponent({
        setup() {
          sharedGate = useSearchFirstFrameHost(store, { expectFirstFrame: false });
          return () => h('div');
        },
      });
      return () =>
        h('div', [showExpecting.value ? h(HostExpect) : null, h(HostIdle)]);
    },
  });

  const { unmount } = mountApp(Root);
  await flushPromises();
  expect(peekSearchFirstFrameGateForTests('demo.ReleaseExpect')?.phase.value).toBe('pending');

  showExpecting.value = false;
  await flushPromises();
  expect(peekSearchFirstFrameGateForTests('demo.ReleaseExpect')?.phase.value).toBe('none');
  expect(sharedGate!.shouldDefer.value).toBe(false);
  unmount();
});
