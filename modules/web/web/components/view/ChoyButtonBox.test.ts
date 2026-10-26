// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Comment, Text, defineComponent, h, nextTick, ref } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyButtonBox from './ChoyButtonBox.vue';

describe('ChoyButtonBox', () => {
  test('does not render the flex shell when the default slot is empty', async () => {
    const empty = mountApp(ChoyButtonBox as any);
    await flushPromises();
    expect(empty.q('.choy-button-box')).toBeNull();
    empty.unmount();

    const commentsOnly = mountApp(ChoyButtonBox as any, {
      slots: {
        default: () => [h(Comment, 'x'), h(Text, '   ')],
      },
    });
    await flushPromises();
    expect(commentsOnly.q('.choy-button-box')).toBeNull();
    commentsOnly.unmount();
  });

  test('renders children and remounts when slot content appears later', async () => {
    const show = ref(false);
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyButtonBox as any, null, {
            default: () =>
              show.value ? h('button', { 'data-testid': 'action' }, 'Go') : null,
          });
      },
    });
    const mounted = mountApp(Host);
    await flushPromises();
    expect(mounted.q('.choy-button-box')).toBeNull();

    show.value = true;
    await nextTick();
    await flushPromises();
    expect(mounted.q('.choy-button-box')).not.toBeNull();
    expect(mounted.q('[data-testid=action]')?.textContent).toBe('Go');

    // Content → empty must tear down the flex shell again (v-if remount).
    show.value = false;
    await nextTick();
    await flushPromises();
    expect(mounted.q('.choy-button-box')).toBeNull();
    mounted.unmount();
  });
});
