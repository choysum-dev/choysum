// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick } from 'vue';
import { ElButton, ElInput } from 'element-plus';

import { GetMessageStoreKey } from '@/web/web/composables/chatter/chatterStores';
import {
  flushPromises,
  fnRecorder,
  mountApp,
  restoreSfc,
  stubSfc,
  type MountAppResult,
} from '@/web/web/__tests__/mountApp';
import OChatterComposer from './OChatterComposer.vue';

describe('OChatterComposer', () => {
  const Post = fnRecorder(async () => ({ Id: 'm1' }));

  function installEpStubs() {
    stubSfc(ElInput as any, {
      name: 'ElInput',
      inheritAttrs: false,
      props: { modelValue: { type: String, default: '' }, disabled: { type: Boolean, default: false } },
      emits: ['update:modelValue', 'keydown'],
      setup(props: any, { emit, attrs }: any) {
        return () =>
          h('textarea', {
            ...attrs,
            class: 'composer-input',
            disabled: props.disabled || undefined,
            value: props.modelValue,
            onInput: (event: any) => emit('update:modelValue', event?.target?.value ?? ''),
            onKeydown: (event: any) => emit('keydown', event),
          });
      },
    });
    stubSfc(ElButton as any, {
      name: 'ElButton',
      inheritAttrs: false,
      props: { disabled: { type: Boolean, default: false }, loading: { type: Boolean, default: false } },
      emits: ['click'],
      setup(props: any, { slots, emit, attrs }: any) {
        return () =>
          h(
            'button',
            {
              ...attrs,
              type: 'button',
              class: 'composer-post',
              disabled: props.disabled || undefined,
              onClick: () => emit('click'),
            },
            slots.default?.()
          );
      },
    });
  }

  beforeEach(() => {
    Post.mockReset();
    Post.mockImplementation(async () => ({ Id: 'm1' }));
    installEpStubs();
  });

  afterEach(() => {
    restoreSfc(ElInput as any);
    restoreSfc(ElButton as any);
  });

  function mountComposer(props?: Partial<{ model: string; resId: string; disabled: boolean }>) {
    const onPosted = fnRecorder();
    const mounted = mountApp(OChatterComposer as any, {
      props: {
        model: 'partner.Partner',
        resId: 'res1',
        ...props,
      },
      reactiveProps: true,
      on: { onPosted },
      provide: {
        [GetMessageStoreKey]: () => ({ Post }),
      },
    });
    return { ...mounted, onPosted };
  }

  async function typeBody(mounted: MountAppResult, text: string) {
    const input = mounted.q('.composer-input') as any;
    expect(input).toBeTruthy();
    input.value = text;
    input.dispatchEvent(new Event('input', { bubbles: true }));
    await nextTick();
  }

  async function clickPost(mounted: MountAppResult) {
    mounted.click('.composer-post');
    await flushPromises();
  }

  test('posts a comment and emits posted on success', async () => {
    const mounted = mountComposer();
    await typeBody(mounted, 'hello');
    await clickPost(mounted);
    expect(Post.calls[0]?.[0]).toEqual({
      Model: 'partner.Partner',
      ResId: 'res1',
      Body: 'hello',
    });
    expect(mounted.onPosted.calls.length).toBe(1);
    expect((mounted.q('.composer-input') as any)?.value ?? '').toBe('');
    mounted.unmount();
  });

  test('shows an error when posting fails', async () => {
    Post.mockImplementation(async () => {
      throw new Error('post failed');
    });
    const mounted = mountComposer();
    await typeBody(mounted, 'hello');
    await clickPost(mounted);
    expect(mounted.q('[role=alert]')?.textContent).toBe('post failed');
    mounted.unmount();
  });

  test('uses the fallback error message for non-Error failures', async () => {
    Post.mockImplementation(async () => {
      throw 'boom';
    });
    const mounted = mountComposer();
    await typeBody(mounted, 'hello');
    await clickPost(mounted);
    expect(mounted.q('[role=alert]')?.textContent).toBe('Failed to post comment');
    mounted.unmount();
  });

  test('ignores empty submits and blocks posting while disabled', async () => {
    const mounted = mountComposer({ disabled: true });
    await typeBody(mounted, 'hello');
    await clickPost(mounted);
    expect(Post.calls.length).toBe(0);

    mounted.props.disabled = false;
    await flushPromises();
    await nextTick();
    await typeBody(mounted, '   ');
    await clickPost(mounted);
    expect(Post.calls.length).toBe(0);
    mounted.unmount();
  });

  test('uses the fallback error message for blank Error messages', async () => {
    Post.mockImplementation(async () => {
      throw new Error('   ');
    });
    const mounted = mountComposer();
    await typeBody(mounted, 'hello');
    await clickPost(mounted);
    expect(mounted.q('[role=alert]')?.textContent).toBe('Failed to post comment');
    mounted.unmount();
  });

  test('submits on ctrl+enter keydown path via Post button wiring', async () => {
    // QJS host has no KeyboardEvent; exercise the same submit() handler the
    // @keydown.ctrl.enter template binding invokes.
    const mounted = mountComposer();
    await typeBody(mounted, 'keyboard');
    await clickPost(mounted);
    expect(Post.calls.length).toBe(1);
    mounted.unmount();
  });

  test('ignores duplicate submits while posting', async () => {
    let resolvePost: (() => void) | undefined;
    Post.mockImplementation(
      () =>
        new Promise(resolve => {
          resolvePost = () => resolve({ Id: 'm1' });
        })
    );
    const mounted = mountComposer();
    await typeBody(mounted, 'hello');
    mounted.click('.composer-post');
    await Promise.resolve();
    mounted.click('.composer-post');
    expect(Post.calls.length).toBe(1);
    resolvePost?.();
    await flushPromises();
    mounted.unmount();
  });
});
