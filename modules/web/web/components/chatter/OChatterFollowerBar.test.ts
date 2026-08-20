// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const SearchByRecord = vi.fn();
const Follow = vi.fn();
const Unfollow = vi.fn();
const authState = { currentUser: { Id: 'usr_1', Name: 'Tester' } as { Id?: string; Name?: string } | null };

vi.mock('@/web/web/i18n', () => ({
  createTranslate: () => ({ _t: (msg: string, ...args: unknown[]) => (args.length ? `${msg}:${args.join(',')}` : msg) }),
}));

vi.mock('@/web/web/composables/chatter/chatterStores', () => ({
  getFollowerStore: () => ({ SearchByRecord, Follow, Unfollow }),
}));

vi.mock('@/auth/web/stores/auth', () => ({
  useAuthStore: () => authState,
}));

import OChatterFollowerBar from './OChatterFollowerBar.vue';

describe('OChatterFollowerBar', () => {
  beforeEach(() => {
    SearchByRecord.mockReset();
    Follow.mockReset();
    Unfollow.mockReset();
    authState.currentUser = { Id: 'usr_1', Name: 'Tester' };
    SearchByRecord.mockResolvedValue([{ UserId: 'usr_1' }, { UserId: 'usr_2' }]);
    Follow.mockResolvedValue({ Id: 'f1' });
    Unfollow.mockResolvedValue(1);
  });

  function mountBar(props?: Partial<{ model: string; resId: string; disabled: boolean }>) {
    return mount(OChatterFollowerBar, {
      props: {
        model: 'partner.Partner',
        resId: 'res1',
        ...props,
      },
      global: {
        stubs: {
          ElButton: defineComponent({
            props: { disabled: Boolean, loading: Boolean },
            emits: ['click'],
            setup(props, { slots, emit }) {
              return () =>
                h(
                  'button',
                  {
                    class: 'el-button',
                    disabled: props.disabled,
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

  it('loads follower state and shows the count', async () => {
    const wrapper = mountBar();
    await flushPromises();
    expect(SearchByRecord).toHaveBeenCalledWith('partner.Partner', 'res1', ['UserId']);
    expect(wrapper.text()).toContain('Unfollow');
    expect(wrapper.text()).toContain('%d followers:2');
  });

  it('follows and unfollows the current record', async () => {
    SearchByRecord.mockResolvedValueOnce([]).mockResolvedValueOnce([{ UserId: 'usr_1' }]);
    const wrapper = mountBar();
    await flushPromises();
    expect(wrapper.text()).toContain('Follow');

    await wrapper.find('button').trigger('click');
    await flushPromises();
    expect(Follow).toHaveBeenCalledWith({ Model: 'partner.Partner', ResId: 'res1' });
    expect(wrapper.text()).toContain('Unfollow');

    await wrapper.find('button').trigger('click');
    await flushPromises();
    expect(Unfollow).toHaveBeenCalledWith({ Model: 'partner.Partner', ResId: 'res1' });
  });

  it('clears state when the thread identity is empty', async () => {
    const wrapper = mountBar({ model: '  ', resId: '  ' });
    await flushPromises();
    expect(SearchByRecord).not.toHaveBeenCalled();
    expect(wrapper.find('button').attributes('disabled')).toBeDefined();
  });

  it('does not toggle when disabled or unauthenticated', async () => {
    authState.currentUser = null;
    const wrapper = mountBar();
    await flushPromises();
    await wrapper.find('button').trigger('click');
    expect(Follow).not.toHaveBeenCalled();

    authState.currentUser = { Id: 'usr_1' };
    await wrapper.setProps({ disabled: true });
    await flushPromises();
    await wrapper.find('button').trigger('click');
    expect(Follow).not.toHaveBeenCalled();
  });

  it('ignores toggle clicks while loading', async () => {
    let resolveSearch: ((rows: unknown[]) => void) | undefined;
    SearchByRecord.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveSearch = resolve;
        })
    );
    const wrapper = mountBar();
    await Promise.resolve();
    await wrapper.find('button').trigger('click');
    expect(Follow).not.toHaveBeenCalled();
    resolveSearch?.([]);
    await flushPromises();
  });
});
