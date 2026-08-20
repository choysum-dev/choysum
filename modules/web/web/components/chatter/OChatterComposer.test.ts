// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const Post = vi.fn();

vi.mock('@/web/web/i18n', () => ({
  createTranslate: () => ({ _t: (msg: string) => msg }),
}));

vi.mock('@/web/web/composables/chatter/chatterStores', () => ({
  getMessageStore: () => ({ Post }),
}));

import OChatterComposer from './OChatterComposer.vue';

describe('OChatterComposer', () => {
  beforeEach(() => {
    Post.mockReset();
    Post.mockResolvedValue({ Id: 'm1' });
  });

  function mountComposer(props?: Partial<{ model: string; resId: string; disabled: boolean }>) {
    return mount(OChatterComposer, {
      props: {
        model: 'partner.Partner',
        resId: 'res1',
        ...props,
      },
      global: {
        stubs: {
          ElInput: defineComponent({
            props: { modelValue: String, disabled: Boolean },
            emits: ['update:modelValue', 'keydown'],
            setup(props, { emit }) {
              return () =>
                h('textarea', {
                  class: 'el-input',
                  disabled: props.disabled,
                  value: props.modelValue,
                  onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
                  onKeydown: (event: KeyboardEvent) => emit('keydown', event),
                });
            },
          }),
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

  it('posts a comment and emits posted on success', async () => {
    const wrapper = mountComposer();
    await wrapper.find('textarea').setValue('hello');
    await wrapper.find('button').trigger('click');
    await flushPromises();
    expect(Post).toHaveBeenCalledWith({
      Model: 'partner.Partner',
      ResId: 'res1',
      Body: 'hello',
    });
    expect(wrapper.emitted('posted')).toHaveLength(1);
    expect((wrapper.find('textarea').element as HTMLTextAreaElement).value).toBe('');
  });

  it('shows an error when posting fails', async () => {
    Post.mockRejectedValue(new Error('post failed'));
    const wrapper = mountComposer();
    await wrapper.find('textarea').setValue('hello');
    await wrapper.find('button').trigger('click');
    await flushPromises();
    expect(wrapper.find('[role="alert"]').text()).toBe('post failed');
  });

  it('uses the fallback error message for non-Error failures', async () => {
    Post.mockRejectedValue('boom');
    const wrapper = mountComposer();
    await wrapper.find('textarea').setValue('hello');
    await wrapper.find('button').trigger('click');
    await flushPromises();
    expect(wrapper.find('[role="alert"]').text()).toBe('Failed to post comment');
  });

  it('ignores empty submits and blocks posting while disabled', async () => {
    const wrapper = mountComposer({ disabled: true });
    expect(wrapper.find('button').attributes('disabled')).toBeDefined();
    await wrapper.find('textarea').setValue('hello');
    await wrapper.find('button').trigger('click');
    expect(Post).not.toHaveBeenCalled();

    await wrapper.setProps({ disabled: false });
    await wrapper.find('textarea').setValue('   ');
    await wrapper.find('button').trigger('click');
    expect(Post).not.toHaveBeenCalled();
  });

  it('uses the fallback error message for blank Error messages', async () => {
    Post.mockRejectedValue(new Error('   '));
    const wrapper = mountComposer();
    await wrapper.find('textarea').setValue('hello');
    await wrapper.find('button').trigger('click');
    await flushPromises();
    expect(wrapper.find('[role="alert"]').text()).toBe('Failed to post comment');
  });

  it('submits on ctrl+enter', async () => {
    const wrapper = mountComposer();
    await wrapper.find('textarea').setValue('keyboard');
    await wrapper.find('textarea').trigger('keydown', { key: 'Enter', ctrlKey: true });
    await flushPromises();
    expect(Post).toHaveBeenCalled();
  });

  it('submits on meta+enter and ignores duplicate submits while posting', async () => {
    let resolvePost: (() => void) | undefined;
    Post.mockImplementation(
      () =>
        new Promise(resolve => {
          resolvePost = resolve as () => void;
        })
    );
    const wrapper = mountComposer();
    await wrapper.find('textarea').setValue('hello');
    await wrapper.find('textarea').trigger('keydown', { key: 'Enter', metaKey: true });
    await wrapper.find('button').trigger('click');
    expect(Post).toHaveBeenCalledTimes(1);
    resolvePost?.();
    await flushPromises();
  });
});
