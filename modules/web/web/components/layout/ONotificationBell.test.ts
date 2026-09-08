// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick, ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import {
  ElBadge,
  ElButton,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElIcon,
} from 'element-plus';

import { useAuthStore } from '@/auth/web/stores/auth';
import { UseNotificationInboxKey } from '@/web/web/composables/chatter/useNotificationInbox';
import {
  flushPromises,
  fnRecorder,
  mountApp,
  restoreSfc,
  stubSfc,
} from '@/web/web/__tests__/mountApp';
import ONotificationBell from './ONotificationBell.vue';

describe('ONotificationBell', () => {
  const rows = ref<Array<{ Id?: string; Model?: string; ResId?: string; IsRead?: boolean; CreatedAt?: string }>>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const unreadCount = ref(0);
  const refresh = fnRecorder(async () => undefined);
  const markRead = fnRecorder(async () => undefined);
  const markAllRead = fnRecorder(async () => undefined);
  const activate = fnRecorder(async () => undefined);
  const deactivate = fnRecorder(() => undefined);
  let pinia: ReturnType<typeof createPinia>;

  function installEpStubs() {
    stubSfc(ElIcon as any, {
      setup(_: any, { slots }: any) {
        return () => h('span', { class: 'el-icon' }, slots.default?.());
      },
    });
    stubSfc(ElBadge as any, {
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'el-badge' }, slots.default?.());
      },
    });
    stubSfc(ElButton as any, {
      name: 'ElButton',
      inheritAttrs: false,
      props: { disabled: { type: Boolean, default: false } },
      emits: ['click'],
      setup(props: any, { slots, emit, attrs }: any) {
        // Forward attrs.onClick: @click.stop may land as fallthrough when emits
        // wiring differs across suites that also stub ElButton.
        return () => {
          const { onClick: attrOnClick, class: attrClass, ...rest } = attrs;
          return h(
            'button',
            {
              type: 'button',
              class: ['el-button', attrClass],
              disabled: props.disabled || undefined,
              ...rest,
              onClick: (event: any) => {
                const payload = event ?? { stopPropagation: () => undefined };
                payload?.stopPropagation?.();
                emit('click', payload);
                if (typeof attrOnClick === 'function') attrOnClick(payload);
              },
            },
            slots.default?.()
          );
        };
      },
    });
    stubSfc(ElDropdown as any, {
      name: 'ElDropdown',
      emits: ['visible-change'],
      setup(_: any, { slots, emit }: any) {
        return () =>
          h('div', { class: 'el-dropdown' }, [
            slots.default?.(),
            h(
              'button',
              {
                type: 'button',
                class: 'open-dropdown',
                onClick: () => emit('visible-change', true),
              },
              'open'
            ),
            h(
              'button',
              {
                type: 'button',
                class: 'close-dropdown',
                onClick: () => emit('visible-change', false),
              },
              'close'
            ),
            slots.dropdown?.(),
          ]);
      },
    });
    stubSfc(ElDropdownMenu as any, {
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'el-dropdown-menu' }, slots.default?.());
      },
    });
    stubSfc(ElDropdownItem as any, {
      emits: ['click'],
      setup(_: any, { slots, emit, attrs }: any) {
        return () =>
          h(
            'div',
            {
              class: 'el-dropdown-item',
              'data-test': 'notification-item',
              ...attrs,
              onClick: () => emit('click'),
            },
            slots.default?.()
          );
      },
    });
  }

  function authedTokens() {
    return {
      accessToken: 'tok',
      refreshToken: 'ref',
      expiresAt: Date.now() + 60_000,
      refreshExpiresAt: Date.now() + 120_000,
    };
  }

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    const auth = useAuthStore();
    auth.tokens = authedTokens() as any;
    auth.currentUser = { Id: 'u1', Name: 'Tester' } as any;

    rows.value = [];
    loading.value = false;
    error.value = null;
    unreadCount.value = 0;
    refresh.mockReset();
    markRead.mockReset();
    markAllRead.mockReset();
    activate.mockReset();
    deactivate.mockReset();
    refresh.mockImplementation(async () => undefined);
    markRead.mockImplementation(async () => undefined);
    markAllRead.mockImplementation(async () => undefined);
    activate.mockImplementation(async () => undefined);
    deactivate.mockImplementation(() => undefined);

    installEpStubs();
  });

  afterEach(() => {
    restoreSfc(ElIcon as any);
    restoreSfc(ElBadge as any);
    restoreSfc(ElButton as any);
    restoreSfc(ElDropdown as any);
    restoreSfc(ElDropdownMenu as any);
    restoreSfc(ElDropdownItem as any);
  });

  function mountBell() {
    return mountApp(ONotificationBell as any, {
      plugins: [pinia],
      provide: {
        [UseNotificationInboxKey]: (enabled: () => boolean) => {
          void enabled();
          return {
            rows,
            loading,
            error,
            unreadCount,
            refresh,
            markRead,
            markAllRead,
            activate,
            deactivate,
          };
        },
      },
    });
  }

  test('activates on mount and refreshes when the dropdown opens', async () => {
    const mounted = mountBell();
    await flushPromises();
    expect(activate.calls.length).toBeGreaterThan(0);

    mounted.click('.open-dropdown');
    await flushPromises();
    expect(refresh.calls.length).toBeGreaterThan(0);
    mounted.unmount();
  });

  test('renders loading, error, empty, and populated dropdown states', async () => {
    loading.value = true;
    let mounted = mountBell();
    await flushPromises();
    expect(mounted.text()).toContain('Loading...');
    mounted.unmount();

    loading.value = false;
    error.value = 'inbox failed';
    mounted = mountBell();
    await flushPromises();
    expect(mounted.text()).toContain('inbox failed');
    mounted.unmount();

    error.value = null;
    mounted = mountBell();
    await flushPromises();
    expect(mounted.text()).toContain('No notifications');
    mounted.unmount();

    rows.value = [
      { Id: 'n1', Model: 'partner.Partner', ResId: 'p1', IsRead: false, CreatedAt: '2024-01-01T12:00:00.000Z' },
      { Id: 'n2', IsRead: true, CreatedAt: '2024-01-02T12:00:00.000Z' },
    ];
    unreadCount.value = 1;
    mounted = mountBell();
    await flushPromises();
    expect(mounted.text()).toContain('Update on partner.Partner (p1)');
    expect(mounted.text()).toContain('New notification');
    expect(mounted.text()).toContain('Mark all read');
    mounted.unmount();
  });

  test('marks unread notifications and reacts to auth changes', async () => {
    rows.value = [{ Id: 'n1', IsRead: false }];
    unreadCount.value = 1;
    const mounted = mountBell();
    await flushPromises();

    mounted.click('[data-test="notification-item"]');
    expect(markRead.calls[0]?.[0]).toBe('n1');

    const auth = useAuthStore();
    auth.tokens = null;
    await flushPromises();
    expect(deactivate.calls.length).toBeGreaterThan(0);

    auth.tokens = authedTokens() as any;
    await flushPromises();
    expect(activate.calls.length).toBeGreaterThan(0);

    rows.value = [{ Id: 'n2', IsRead: true }];
    unreadCount.value = 0;
    await nextTick();
    markRead.mockClear();
    mounted.click('[data-test="notification-item"]');
    expect(markRead.calls.length).toBe(0);

    unreadCount.value = 2;
    rows.value = [{ Id: 'n3', IsRead: false }];
    await flushPromises();
    markAllRead.mockClear();
    mounted.click('[data-test="notification-mark-all-read"]');
    expect(markAllRead.calls.length).toBe(1);
    mounted.unmount();
  });

  test('skips auth setup when unmounted before initialization completes', async () => {
    const mounted = mountBell();
    mounted.unmount();
    await flushPromises();
    expect(deactivate.calls.length).toBeGreaterThan(0);
  });

  test('cleans up when unmounted during activate', async () => {
    let settleActivate: (() => void) | undefined;
    activate.mockImplementation(
      () =>
        new Promise<void>(resolve => {
          settleActivate = resolve;
        })
    );
    const mounted = mountBell();
    await flushPromises();
    expect(activate.calls.length).toBeGreaterThan(0);
    mounted.unmount();
    settleActivate?.();
    await flushPromises();
    expect(deactivate.calls.length).toBeGreaterThanOrEqual(2);
  });

  test('ignores auth subscription updates when the auth state is unchanged', async () => {
    const mounted = mountBell();
    await flushPromises();
    activate.mockClear();
    // Same tokens → isAuthenticated stays true; $subscribe still fires on assignment.
    useAuthStore().tokens = { ...authedTokens() } as any;
    await flushPromises();
    // Identical auth boolean should not re-activate.
    expect(activate.calls.length).toBe(0);
    mounted.unmount();
  });

  test('ignores auth listener updates after unmount', async () => {
    const mounted = mountBell();
    await flushPromises();
    mounted.unmount();
    deactivate.mockClear();
    useAuthStore().tokens = null;
    await flushPromises();
    expect(deactivate.calls.length).toBe(0);
  });

  test('ignores notification clicks without an id', async () => {
    rows.value = [{ IsRead: false }];
    const mounted = mountBell();
    await flushPromises();
    mounted.click('[data-test="notification-item"]');
    expect(markRead.calls.length).toBe(0);
    mounted.unmount();
  });

  test('does not refresh when the dropdown closes or the user is logged out', async () => {
    const mounted = mountBell();
    await flushPromises();
    refresh.mockClear();
    mounted.click('.close-dropdown');
    await flushPromises();
    expect(refresh.calls.length).toBe(0);

    useAuthStore().tokens = null;
    const loggedOut = mountBell();
    await flushPromises();
    expect(loggedOut.q('.el-dropdown')).toBeFalsy();
    loggedOut.unmount();
    mounted.unmount();
  });

  test('handles auth initialization failures', async () => {
    // Dynamic auth import always succeeds here; force failure by clearing pinia.
    setActivePinia(null as any);
    const mounted = mountApp(ONotificationBell as any, {
      provide: {
        [UseNotificationInboxKey]: () => ({
          rows,
          loading,
          error,
          unreadCount,
          refresh,
          markRead,
          markAllRead,
          activate,
          deactivate,
        }),
      },
    });
    await flushPromises();
    expect(mounted.q('.el-dropdown')).toBeFalsy();
    mounted.unmount();
    setActivePinia(pinia);
  });

  test('swallows auth failures after unmount during activate', async () => {
    let rejectActivate: ((err: unknown) => void) | undefined;
    activate.mockImplementation(
      () =>
        new Promise<void>((_resolve, reject) => {
          rejectActivate = reject;
        })
    );
    const mounted = mountBell();
    await flushPromises();
    expect(activate.calls.length).toBeGreaterThan(0);
    mounted.unmount();
    rejectActivate?.(new Error('activate failed'));
    await flushPromises();
    expect(deactivate.calls.length).toBeGreaterThan(0);
  });
});
