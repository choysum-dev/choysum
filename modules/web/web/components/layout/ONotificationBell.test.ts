// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { defineComponent, h, nextTick, ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const rows = ref<Array<{ Id?: string; Model?: string; ResId?: string; IsRead?: boolean; CreatedAt?: string }>>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const unreadCount = ref(0);
const refresh = vi.fn(async () => undefined);
const markRead = vi.fn(async () => undefined);
const markAllRead = vi.fn(async () => undefined);
const activate = vi.fn(async () => undefined);
const deactivate = vi.fn(() => undefined);

const authStore = {
  isAuthenticated: true,
  $subscribe: vi.fn((listener: () => void) => {
    authStore._listener = listener;
    return () => undefined;
  }),
  _listener: null as null | (() => void),
};

vi.mock('@/web/web/i18n', () => ({
  createTranslate: () => ({ _t: (msg: string, ...args: unknown[]) => (args.length ? `${msg}:${args.join(':')}` : msg) }),
}));

vi.mock('@/web/web/composables/chatter/useNotificationInbox', () => ({
  useNotificationInbox: () => ({
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
}));

vi.mock('@/auth/web/stores/auth', () => ({
  useAuthStore: () => authStore,
}));

import ONotificationBell from './ONotificationBell.vue';

describe('ONotificationBell', () => {
  beforeEach(() => {
    rows.value = [];
    loading.value = false;
    error.value = null;
    unreadCount.value = 0;
    refresh.mockReset();
    markRead.mockReset();
    markAllRead.mockReset();
    activate.mockReset();
    deactivate.mockReset();
    refresh.mockResolvedValue(undefined);
    markRead.mockResolvedValue(undefined);
    markAllRead.mockResolvedValue(undefined);
    activate.mockResolvedValue(undefined);
    deactivate.mockReturnValue(undefined);
    authStore.isAuthenticated = true;
    authStore.$subscribe.mockClear();
    authStore._listener = null;
  });

  function mountBell() {
    return mount(ONotificationBell, {
      global: {
        stubs: {
          Bell: defineComponent({ setup: () => () => h('span', { class: 'bell-icon' }) }),
          ElIcon: defineComponent({
            setup(_, { slots }) {
              return () => h('span', { class: 'el-icon' }, slots.default?.());
            },
          }),
          ElBadge: defineComponent({
            setup(_, { slots }) {
              return () => h('div', { class: 'el-badge' }, slots.default?.());
            },
          }),
          ElButton: defineComponent({
            props: { disabled: Boolean },
            emits: ['click'],
            setup(props, { slots, emit }) {
              return () =>
                h(
                  'button',
                  {
                    class: 'el-button',
                    disabled: props.disabled,
                    onClick: () => {
                      emit('click', { stopPropagation: () => undefined });
                    },
                  },
                  slots.default?.()
                );
            },
          }),
          ElDropdown: defineComponent({
            name: 'ElDropdown',
            emits: ['visible-change'],
            setup(_, { slots, emit }) {
              return () =>
                h('div', { class: 'el-dropdown' }, [
                  slots.default?.(),
                  h(
                    'button',
                    {
                      class: 'open-dropdown',
                      onClick: () => emit('visible-change', true),
                    },
                    'open'
                  ),
                  h(
                    'button',
                    {
                      class: 'close-dropdown',
                      onClick: () => emit('visible-change', false),
                    },
                    'close'
                  ),
                  slots.dropdown?.(),
                ]);
            },
          }),
          ElDropdownMenu: defineComponent({
            setup(_, { slots }) {
              return () => h('div', { class: 'el-dropdown-menu' }, slots.default?.());
            },
          }),
          ElDropdownItem: defineComponent({
            emits: ['click'],
            setup(_, { slots, emit }) {
              return () =>
                h(
                  'div',
                  {
                    class: 'el-dropdown-item',
                    onClick: () => emit('click'),
                  },
                  slots.default?.()
                );
            },
          }),
        },
      },
    });
  }

  it('activates on mount and refreshes when the dropdown opens', async () => {
    const wrapper = mountBell();
    await flushPromises();
    expect(activate).toHaveBeenCalled();

    await wrapper.find('.open-dropdown').trigger('click');
    await flushPromises();
    expect(refresh).toHaveBeenCalled();
  });

  it('renders loading, error, empty, and populated dropdown states', async () => {
    loading.value = true;
    let wrapper = mountBell();
    await flushPromises();
    expect(wrapper.text()).toContain('Loading...');

    loading.value = false;
    error.value = 'inbox failed';
    wrapper = mountBell();
    await flushPromises();
    expect(wrapper.text()).toContain('inbox failed');

    error.value = null;
    wrapper = mountBell();
    await flushPromises();
    expect(wrapper.text()).toContain('No notifications');

    rows.value = [
      { Id: 'n1', Model: 'partner.Partner', ResId: 'p1', IsRead: false, CreatedAt: '2024-01-01T12:00:00.000Z' },
      { Id: 'n2', IsRead: true, CreatedAt: '2024-01-02T12:00:00.000Z' },
    ];
    unreadCount.value = 1;
    wrapper = mountBell();
    await flushPromises();
    expect(wrapper.text()).toContain('Update on %s (%s):partner.Partner:p1');
    expect(wrapper.text()).toContain('New notification');
    expect(wrapper.text()).toContain('Mark all read');
  });

  it('marks unread notifications and reacts to auth changes', async () => {
    rows.value = [{ Id: 'n1', IsRead: false }];
    unreadCount.value = 1;
    const wrapper = mountBell();
    await flushPromises();

    authStore.isAuthenticated = false;
    authStore._listener?.();
    expect(deactivate).toHaveBeenCalled();

    authStore.isAuthenticated = true;
    authStore._listener?.();
    expect(activate).toHaveBeenCalled();

    await wrapper.find('.el-dropdown-item').trigger('click');
    expect(markRead).toHaveBeenCalledWith('n1');

    rows.value = [{ Id: 'n2', IsRead: true }];
    unreadCount.value = 0;
    await nextTick();
    markRead.mockClear();
    await wrapper.find('.el-dropdown-item').trigger('click');
    expect(markRead).not.toHaveBeenCalled();

    unreadCount.value = 2;
    rows.value = [{ Id: 'n3', IsRead: false }];
    await nextTick();
    await wrapper.findAll('.el-button').at(-1)?.trigger('click');
    expect(markAllRead).toHaveBeenCalled();
  });

  it('skips auth setup when unmounted before initialization completes', async () => {
    const wrapper = mountBell();
    wrapper.unmount();
    await flushPromises();
    expect(deactivate).toHaveBeenCalled();
  });

  it('cleans up when unmounted during activate', async () => {
    let settleActivate: (() => void) | undefined;
    activate.mockReturnValue(
      new Promise<void>(resolve => {
        settleActivate = resolve;
      })
    );
    const wrapper = mountBell();
    await vi.waitFor(() => expect(activate).toHaveBeenCalled());
    wrapper.unmount();
    settleActivate?.();
    await flushPromises();
    expect(deactivate.mock.calls.length).toBeGreaterThanOrEqual(2);
  });

  it('ignores auth subscription updates when the auth state is unchanged', async () => {
    const wrapper = mountBell();
    await flushPromises();
    activate.mockClear();
    authStore._listener?.();
    expect(activate).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('ignores auth listener updates after unmount', async () => {
    const wrapper = mountBell();
    await flushPromises();
    wrapper.unmount();
    deactivate.mockClear();
    authStore._listener?.();
    expect(deactivate).not.toHaveBeenCalled();
  });

  it('ignores notification clicks without an id', async () => {
    rows.value = [{ IsRead: false }];
    const wrapper = mountBell();
    await flushPromises();
    await wrapper.find('.el-dropdown-item').trigger('click');
    expect(markRead).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('does not refresh when the dropdown closes or the user is logged out', async () => {
    const wrapper = mountBell();
    await flushPromises();
    refresh.mockClear();
    await wrapper.find('.close-dropdown').trigger('click');
    await flushPromises();
    expect(refresh).not.toHaveBeenCalled();

    authStore.isAuthenticated = false;
    const loggedOut = mountBell();
    await flushPromises();
    expect(loggedOut.find('.el-dropdown').exists()).toBe(false);
    loggedOut.unmount();
    wrapper.unmount();
  });

  it('handles auth initialization failures', async () => {
    authStore.$subscribe.mockImplementationOnce(() => {
      throw new Error('auth boom');
    });
    const wrapper = mountBell();
    await flushPromises();
    expect(wrapper.find('.el-dropdown').exists()).toBe(false);
  });
});
