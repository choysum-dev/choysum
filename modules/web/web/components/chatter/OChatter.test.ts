// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { defineComponent, h, ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const refresh = vi.fn(async () => undefined);
const entries = ref<any[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const authState = { currentUser: { Id: 'usr_test', Name: 'Tester' } as { Id?: string; Name?: string } | null };

vi.mock('@/web/web/composables/chatter/useChatterTimeline', () => ({
  useChatterTimeline: () => ({ entries, loading, error, refresh }),
}));

vi.mock('@/web/web/composables/chatter/useChatterThreadTips', () => ({
  useChatterThreadTips: vi.fn(),
}));

vi.mock('@/auth/web/stores/auth', () => ({
  useAuthStore: () => authState,
}));

vi.mock('@/web/web/i18n', () => ({
  createTranslate: () => ({ _t: (msg: string) => msg }),
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: () => ({
    Post: vi.fn(async () => ({ Id: 'msg1' })),
    SearchByRecord: vi.fn(async () => []),
    Follow: vi.fn(async () => ({ Id: 'f1' })),
    Unfollow: vi.fn(async () => 1),
  }),
}));

import OChatter from './OChatter.vue';

describe('OChatter', () => {
  beforeEach(() => {
    refresh.mockClear();
    entries.value = [];
    loading.value = false;
    error.value = null;
    authState.currentUser = { Id: 'usr_test', Name: 'Tester' };
  });

  function mountChatter(props?: Record<string, unknown>) {
    return mount(OChatter, {
      props: {
        model: 'partner.Partner',
        resId: 'res_partner_1',
        ...props,
      },
      global: {
        stubs: {
          ElCard: defineComponent({
            setup(_, { slots }) {
              return () => h('div', [slots.header?.(), slots.default?.()]);
            },
          }),
          OChatterFollowerBar: defineComponent({
            props: ['model', 'resId', 'disabled'],
            setup: () => () => h('div', { class: 'follower-bar' }),
          }),
          OChatterComposer: defineComponent({
            emits: ['posted'],
            setup(_, { emit }) {
              return () =>
                h('button', {
                  class: 'composer-post',
                  onClick: () => emit('posted'),
                });
            },
          }),
          OChatterTimeline: defineComponent({
            props: ['resolveAuthorLabel'],
            setup(props) {
              return () =>
                h('div', { class: 'timeline' }, [
                  h('span', { class: 'system-label' }, props.resolveAuthorLabel(null)),
                  h('span', { class: 'you-label' }, props.resolveAuthorLabel('usr_test')),
                  h('span', { class: 'other-label' }, props.resolveAuthorLabel('usr_other')),
                ]);
            },
          }),
        },
      },
    });
  }

  it('renders empty timeline state and refreshes after post', async () => {
    const wrapper = mountChatter();
    expect(wrapper.text()).toContain('Activity');
    await wrapper.find('.composer-post').trigger('click');
    await flushPromises();
    expect(refresh).toHaveBeenCalled();
  });

  it('hides the composer when disabled or missing resId', () => {
    expect(mountChatter({ resId: '' }).find('.composer-post').exists()).toBe(false);
    expect(mountChatter({ disabled: true }).find('.composer-post').exists()).toBe(false);
    expect(mountChatter({ showComposer: false }).find('.composer-post').exists()).toBe(false);
  });

  it('resolves author labels for system, current user, and other users', () => {
    const wrapper = mountChatter();
    expect(wrapper.find('.system-label').text()).toBe('System');
    expect(wrapper.find('.you-label').text()).toBe('Tester');
    expect(wrapper.find('.other-label').text()).toBe('usr_other');

    authState.currentUser = { Id: 'usr_test', Name: '  ' };
    const unnamed = mountChatter();
    expect(unnamed.find('.you-label').text()).toBe('You');

    authState.currentUser = { Id: '  ', Name: 'Tester' };
    const noCurrentId = mountChatter();
    expect(noCurrentId.find('.you-label').text()).toBe('usr_test');
  });
});
