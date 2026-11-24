// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { resolveConfirmChoy, useConfirmChoyStore } from '@/web/web/composables/confirmChoyAction';
import { getGlobalActionChecker, setGlobalActionChecker } from '@/web/web/directives/action';
import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import ChoyActionButton from './ChoyActionButton.vue';

describe('ChoyActionButton', () => {
  afterEach(() => {
    setGlobalActionChecker(undefined);
    try {
      useConnectivityStore().resetForTests();
    } catch {
      /* pinia may be absent */
    }
  });

  test('emits execute on click when allowed', async () => {
    let runs = 0;
    const w = mountApp(ChoyActionButton as any, {
      props: { 'data-testid': 'act' },
      on: {
        onExecute: () => {
          runs += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      expect(typeof onClick).toBe('function');
      await onClick?.(new Event('click'));
      await flushPromises();
      expect(runs).toBe(1);
    } finally {
      w.unmount();
    }
  });

  test('prefers run prop over emit execute', async () => {
    let runs = 0;
    let executes = 0;
    const w = mountApp(ChoyActionButton as any, {
      props: {
        'data-testid': 'act-run',
        run: async () => {
          runs += 1;
        },
      },
      on: {
        onExecute: () => {
          executes += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      await onClick?.(new Event('click'));
      await flushPromises();
      expect(runs).toBe(1);
      expect(executes).toBe(0);
    } finally {
      w.unmount();
    }
  });

  test('as=button still stopPropagation when enabled', async () => {
    const w = mountApp(ChoyActionButton as any, {
      props: { as: 'button', 'data-testid': 'act-btn' },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      let stopped = false;
      const ev = {
        stopPropagation() {
          stopped = true;
        },
        preventDefault() {},
      } as unknown as Event;
      await onClick?.(ev);
      await flushPromises();
      expect(stopped).toBe(true);
    } finally {
      w.unmount();
    }
  });

  test('as=span does not stopPropagation when enabled (label file chrome)', async () => {
    const w = mountApp(ChoyActionButton as any, {
      props: { as: 'span', 'data-testid': 'act-span' },
      slots: { default: () => 'Pick' },
    });
    try {
      await flushPromises();
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      let stopped = false;
      const ev = {
        stopPropagation() {
          stopped = true;
        },
        preventDefault() {},
      } as unknown as Event;
      await onClick?.(ev);
      await flushPromises();
      expect(stopped).toBe(false);
    } finally {
      w.unmount();
    }
  });

  test('as=span stops propagation when disabled', async () => {
    const w = mountApp(ChoyActionButton as any, {
      props: { as: 'span', disabled: true, 'data-testid': 'act-span-dis' },
      slots: { default: () => 'Pick' },
    });
    try {
      await flushPromises();
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      let stopped = false;
      let prevented = false;
      const ev = {
        stopPropagation() {
          stopped = true;
        },
        preventDefault() {
          prevented = true;
        },
      } as unknown as Event;
      await onClick?.(ev);
      await flushPromises();
      expect(stopped).toBe(true);
      expect(prevented).toBe(true);
    } finally {
      w.unmount();
    }
  });

  test('swallows click when disabled or busy', async () => {
    let runs = 0;
    const w = mountApp(ChoyActionButton as any, {
      props: { disabled: true, 'data-testid': 'act-dis' },
      on: { onExecute: () => { runs += 1; } },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      const event = new Event('click', { cancelable: true });
      await onClick?.(event);
      expect(runs).toBe(0);

      w.unmount();
    } finally {
      /* done */
    }

    const busy = mountApp(ChoyActionButton as any, {
      props: { busy: true, 'data-testid': 'act-busy' },
      on: { onExecute: () => { runs += 1; } },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onClick = busy.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      await onClick?.(new Event('click'));
      expect(runs).toBe(0);
    } finally {
      busy.unmount();
    }
  });

  test('hides when actionId denied in hide mode', async () => {
    const w = mountApp(ChoyActionButton as any, {
      props: {
        actionId: 'denied.action',
        hasAction: () => false,
        actionMode: 'hide',
        'data-testid': 'act-hide',
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      expect(w.q('[data-testid="act-hide"]')).toBeNull();
      expect(w.setupState()?.aclVisible).toBe(false);
    } finally {
      w.unmount();
    }
  });

  test('disable mode keeps control disabled when ACL denies', async () => {
    let runs = 0;
    const w = mountApp(ChoyActionButton as any, {
      props: {
        actionId: 'denied.action',
        hasAction: () => false,
        actionMode: 'disable',
        'data-testid': 'act-acl-dis',
      },
      on: { onExecute: () => { runs += 1; } },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      expect(w.q('[data-testid="act-acl-dis"]')).not.toBeNull();
      expect(w.setupState()?.effectiveDisabled).toBe(true);
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      await onClick?.(new Event('click'));
      expect(runs).toBe(0);
    } finally {
      w.unmount();
    }
  });

  test('uses global action checker when hasAction omitted', async () => {
    setGlobalActionChecker((id) => id === 'ok.action');
    const hidden = mountApp(ChoyActionButton as any, {
      props: { actionId: 'nope', 'data-testid': 'g-hide' },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      expect(hidden.q('[data-testid="g-hide"]')).toBeNull();
    } finally {
      hidden.unmount();
    }

    let runs = 0;
    const shown = mountApp(ChoyActionButton as any, {
      props: { actionId: 'ok.action', 'data-testid': 'g-ok' },
      on: { onExecute: () => { runs += 1; } },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      expect(shown.q('[data-testid="g-ok"]')).not.toBeNull();
      const onClick = shown.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      await onClick?.(new Event('click'));
      expect(runs).toBe(1);
    } finally {
      shown.unmount();
    }
  });

  test('blocks when unreachable and requiresConnectivity default true', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().resetForTests();
    useConnectivityStore().markUnreachable();

    let runs = 0;
    const w = mountApp(ChoyActionButton as any, {
      props: { 'data-testid': 'act-off' },
      plugins: [pinia],
      on: { onExecute: () => { runs += 1; } },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      await nextTick();
      expect(w.setupState()?.offlineBlocked).toBe(true);
      expect(w.setupState()?.effectiveDisabled).toBe(true);
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      await onClick?.(new Event('click'));
      expect(runs).toBe(0);
    } finally {
      w.unmount();
    }
  });

  test('requiresConnectivity false stays clickable while unreachable', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().markUnreachable();

    let runs = 0;
    const w = mountApp(ChoyActionButton as any, {
      props: {
        requiresConnectivity: false,
        'data-testid': 'act-local',
      },
      plugins: [pinia],
      on: { onExecute: () => { runs += 1; } },
      slots: { default: () => 'Cancel' },
    });
    try {
      await flushPromises();
      expect(w.setupState()?.offlineBlocked).toBe(false);
      expect(w.setupState()?.effectiveDisabled).toBe(false);
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      await onClick?.(new Event('click'));
      expect(runs).toBe(1);
    } finally {
      w.unmount();
    }
  });

  test('without Pinia treats session as reachable', async () => {
    setActivePinia(undefined as any);
    let runs = 0;
    const w = mountApp(ChoyActionButton as any, {
      props: { 'data-testid': 'act-nopinia' },
      on: { onExecute: () => { runs += 1; } },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      expect(w.setupState()?.offlineBlocked).toBe(false);
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      await onClick?.(new Event('click'));
      expect(runs).toBe(1);
    } finally {
      w.unmount();
    }
  });

  test('confirm string gates execute until confirmed', async () => {
    let runs = 0;
    const w = mountApp(ChoyActionButton as any, {
      props: { confirm: 'Sure?', 'data-testid': 'act-conf' },
      on: { onExecute: () => { runs += 1; } },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      const pending = onClick?.(new Event('click'));
      expect(useConfirmChoyStore().open).toBe(true);
      resolveConfirmChoy('dismiss');
      await pending;
      expect(runs).toBe(0);

      const pending2 = onClick?.(new Event('click'));
      resolveConfirmChoy('confirm');
      await pending2;
      expect(runs).toBe(1);
    } finally {
      w.unmount();
    }
  });

  test('confirm object passes destructive and title', async () => {
    let runs = 0;
    const w = mountApp(ChoyActionButton as any, {
      props: {
        confirm: { message: 'Delete?', title: 'Confirm delete', destructive: true },
        'data-testid': 'act-conf-obj',
      },
      on: { onExecute: () => { runs += 1; } },
      slots: { default: () => 'Del' },
    });
    try {
      await flushPromises();
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      const pending = onClick?.(new Event('click'));
      const store = useConfirmChoyStore();
      expect(store.title).toBe('Confirm delete');
      expect(store.destructive).toBe(true);
      resolveConfirmChoy('confirm');
      await pending;
      expect(runs).toBe(1);
    } finally {
      w.unmount();
    }
  });

  test('pending clears after async run settles', async () => {
    let release!: () => void;
    const gate = new Promise<void>((r) => {
      release = r;
    });
    const w = mountApp(ChoyActionButton as any, {
      props: {
        run: () => gate,
        'data-testid': 'act-pend',
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onClick = w.setupState()?.onClick as ((e: Event) => Promise<void>) | undefined;
      const pendingClick = onClick?.(new Event('click'));
      await nextTick();
      expect(w.setupState()?.pending).toBe(true);
      expect(w.setupState()?.effectiveDisabled).toBe(true);
      release();
      await pendingClick;
      expect(w.setupState()?.pending).toBe(false);
    } finally {
      w.unmount();
    }
  });

  test('empty actionId does not hide (ungated)', async () => {
    const w = mountApp(ChoyActionButton as any, {
      props: { actionId: '  ', hasAction: () => false, 'data-testid': 'act-empty-id' },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      expect(w.q('[data-testid="act-empty-id"]')).not.toBeNull();
    } finally {
      w.unmount();
    }
  });
});
