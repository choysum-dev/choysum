// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { resolveConfirmChoy, useConfirmChoyStore } from '@/web/web/composables/confirmChoyAction';
import { setGlobalActionChecker } from '@/web/web/directives/action';
import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import ChoyActionMenuItem from './ChoyActionMenuItem.vue';

describe('ChoyActionMenuItem', () => {
  afterEach(() => {
    setGlobalActionChecker(undefined);
    try {
      useConnectivityStore().resetForTests();
    } catch {
      /* pinia may be absent */
    }
  });

  test('emits execute on select when allowed', async () => {
    let runs = 0;
    const w = mountApp(ChoyActionMenuItem as any, {
      props: { 'data-testid': 'ami' },
      on: {
        onExecute: () => {
          runs += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onSelect = w.setupState()?.onSelect as ((e: Event) => Promise<void>) | undefined;
      expect(typeof onSelect).toBe('function');
      await onSelect?.(new Event('select'));
      await flushPromises();
      expect(runs).toBe(1);
    } finally {
      w.unmount();
    }
  });

  test('prefers run prop over emit execute', async () => {
    let runs = 0;
    let executes = 0;
    const w = mountApp(ChoyActionMenuItem as any, {
      props: {
        'data-testid': 'ami-run',
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
      const onSelect = w.setupState()?.onSelect as ((e: Event) => Promise<void>) | undefined;
      await onSelect?.(new Event('select'));
      await flushPromises();
      expect(runs).toBe(1);
      expect(executes).toBe(0);
    } finally {
      w.unmount();
    }
  });

  test('swallows select when disabled or busy', async () => {
    let runs = 0;
    const w = mountApp(ChoyActionMenuItem as any, {
      props: { disabled: true, 'data-testid': 'ami-dis' },
      on: {
        onExecute: () => {
          runs += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onSelect = w.setupState()?.onSelect as ((e: Event) => Promise<void>) | undefined;
      await onSelect?.(new Event('select'));
      expect(runs).toBe(0);
    } finally {
      w.unmount();
    }

    const busy = mountApp(ChoyActionMenuItem as any, {
      props: { busy: true, 'data-testid': 'ami-busy' },
      on: {
        onExecute: () => {
          runs += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onSelect = busy.setupState()?.onSelect as ((e: Event) => Promise<void>) | undefined;
      await onSelect?.(new Event('select'));
      expect(runs).toBe(0);
    } finally {
      busy.unmount();
    }
  });

  test('hides when actionId denied in hide mode', async () => {
    const w = mountApp(ChoyActionMenuItem as any, {
      props: {
        actionId: 'denied.action',
        hasAction: () => false,
        actionMode: 'hide',
        'data-testid': 'ami-hide',
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      expect(w.q('[data-testid="ami-hide"]')).toBeNull();
      expect(w.setupState()?.aclVisible).toBe(false);
    } finally {
      w.unmount();
    }
  });

  test('disable mode keeps control disabled when ACL denies', async () => {
    let runs = 0;
    const w = mountApp(ChoyActionMenuItem as any, {
      props: {
        actionId: 'denied.action',
        hasAction: () => false,
        actionMode: 'disable',
        'data-testid': 'ami-acl-dis',
      },
      on: {
        onExecute: () => {
          runs += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      expect(w.q('[data-testid="ami-acl-dis"]')).not.toBeNull();
      expect(w.setupState()?.effectiveDisabled).toBe(true);
      const onSelect = w.setupState()?.onSelect as ((e: Event) => Promise<void>) | undefined;
      await onSelect?.(new Event('select'));
      expect(runs).toBe(0);
    } finally {
      w.unmount();
    }
  });

  test('blocks when unreachable and requiresConnectivity default true', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().resetForTests();
    useConnectivityStore().markUnreachable();

    let runs = 0;
    const w = mountApp(ChoyActionMenuItem as any, {
      props: { 'data-testid': 'ami-off' },
      plugins: [pinia],
      on: {
        onExecute: () => {
          runs += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      await nextTick();
      expect(w.setupState()?.offlineBlocked).toBe(true);
      expect(w.setupState()?.effectiveDisabled).toBe(true);
      const onSelect = w.setupState()?.onSelect as ((e: Event) => Promise<void>) | undefined;
      await onSelect?.(new Event('select'));
      expect(runs).toBe(0);
    } finally {
      w.unmount();
    }
  });

  test('requiresConnectivity false stays selectable while unreachable', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().markUnreachable();

    let runs = 0;
    const w = mountApp(ChoyActionMenuItem as any, {
      props: {
        requiresConnectivity: false,
        'data-testid': 'ami-local',
      },
      plugins: [pinia],
      on: {
        onExecute: () => {
          runs += 1;
        },
      },
      slots: { default: () => 'Reset' },
    });
    try {
      await flushPromises();
      expect(w.setupState()?.offlineBlocked).toBe(false);
      expect(w.setupState()?.effectiveDisabled).toBe(false);
      const onSelect = w.setupState()?.onSelect as ((e: Event) => Promise<void>) | undefined;
      await onSelect?.(new Event('select'));
      expect(runs).toBe(1);
    } finally {
      w.unmount();
    }
  });

  test('without Pinia treats session as reachable', async () => {
    setActivePinia(undefined as any);
    let runs = 0;
    const w = mountApp(ChoyActionMenuItem as any, {
      props: { 'data-testid': 'ami-nopinia' },
      on: {
        onExecute: () => {
          runs += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      expect(w.setupState()?.offlineBlocked).toBe(false);
      const onSelect = w.setupState()?.onSelect as ((e: Event) => Promise<void>) | undefined;
      await onSelect?.(new Event('select'));
      expect(runs).toBe(1);
    } finally {
      w.unmount();
    }
  });

  test('confirm string gates execute until confirmed', async () => {
    let runs = 0;
    const w = mountApp(ChoyActionMenuItem as any, {
      props: { confirm: 'Sure?', 'data-testid': 'ami-conf' },
      on: {
        onExecute: () => {
          runs += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onSelect = w.setupState()?.onSelect as ((e: Event) => Promise<void>) | undefined;
      const pending = onSelect?.(new Event('select'));
      expect(useConfirmChoyStore().open).toBe(true);
      resolveConfirmChoy('dismiss');
      await pending;
      expect(runs).toBe(0);

      const pending2 = onSelect?.(new Event('select'));
      resolveConfirmChoy('confirm');
      await pending2;
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
    const w = mountApp(ChoyActionMenuItem as any, {
      props: {
        run: () => gate,
        'data-testid': 'ami-pend',
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const onSelect = w.setupState()?.onSelect as ((e: Event) => Promise<void>) | undefined;
      const pendingSelect = onSelect?.(new Event('select'));
      await nextTick();
      expect(w.setupState()?.pending).toBe(true);
      expect(w.setupState()?.effectiveDisabled).toBe(true);
      release();
      await pendingSelect;
      expect(w.setupState()?.pending).toBe(false);
    } finally {
      w.unmount();
    }
  });
});
