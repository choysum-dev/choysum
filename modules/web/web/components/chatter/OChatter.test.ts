// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { ElCard } from 'element-plus';

import { useAuthStore } from '@/auth/web/stores/auth';
import { UseChatterTimelineKey } from '@/web/web/composables/chatter/useChatterTimeline';
import { UseChatterThreadTipsKey } from '@/web/web/composables/chatter/useChatterThreadTips';
import {
  flushPromises,
  fnRecorder,
  mountApp,
  restoreSfc,
  stubSfc,
} from '@/web/web/__tests__/mountApp';
import OChatter from './OChatter.vue';
import OChatterComposer from './OChatterComposer.vue';
import OChatterFollowerBar from './OChatterFollowerBar.vue';
import OChatterTimeline from './OChatterTimeline.vue';

describe('OChatter', () => {
  const refresh = fnRecorder(async () => undefined);
  const entries = ref<any[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  let pinia: ReturnType<typeof createPinia>;

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    refresh.mockClear();
    entries.value = [];
    loading.value = false;
    error.value = null;
    useAuthStore().currentUser = { Id: 'usr_test', Name: 'Tester' } as any;

    stubSfc(ElCard as any, {
      name: 'ElCard',
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'o-chatter-card' }, [slots.header?.(), slots.default?.()]);
      },
    });
    stubSfc(OChatterFollowerBar, {
      props: ['model', 'resId', 'disabled'],
      setup: () => () => h('div', { class: 'follower-bar' }),
    });
    stubSfc(OChatterComposer, {
      emits: ['posted'],
      setup(_: any, { emit }: any) {
        return () =>
          h('button', {
            type: 'button',
            class: 'composer-post',
            onClick: () => emit('posted'),
          });
      },
    });
    stubSfc(OChatterTimeline, {
      props: ['resolveAuthorLabel'],
      setup(props: any) {
        return () =>
          h('div', { class: 'timeline' }, [
            h('span', { class: 'system-label' }, props.resolveAuthorLabel(null)),
            h('span', { class: 'you-label' }, props.resolveAuthorLabel('usr_test')),
            h('span', { class: 'other-label' }, props.resolveAuthorLabel('usr_other')),
          ]);
      },
    });
  });

  afterEach(() => {
    restoreSfc(ElCard as any);
    restoreSfc(OChatterFollowerBar);
    restoreSfc(OChatterComposer);
    restoreSfc(OChatterTimeline);
  });

  function mountChatter(props?: Record<string, unknown>) {
    return mountApp(OChatter as any, {
      props: {
        model: 'partner.Partner',
        resId: 'res_partner_1',
        ...props,
      },
      plugins: [pinia],
      provide: {
        [UseChatterTimelineKey]: () => ({ entries, loading, error, refresh }),
        [UseChatterThreadTipsKey]: () => undefined,
      },
    });
  }

  test('renders empty timeline state and refreshes after post', async () => {
    const mounted = mountChatter();
    expect(mounted.text()).toContain('Activity');
    mounted.click('.composer-post');
    await flushPromises();
    expect(refresh.calls.length).toBe(1);
    mounted.unmount();
  });

  test('hides the composer when disabled or missing resId', () => {
    const emptyRes = mountChatter({ resId: '' });
    expect(emptyRes.q('.composer-post')).toBeFalsy();
    emptyRes.unmount();

    const disabled = mountChatter({ disabled: true });
    expect(disabled.q('.composer-post')).toBeFalsy();
    disabled.unmount();

    const hidden = mountChatter({ showComposer: false });
    expect(hidden.q('.composer-post')).toBeFalsy();
    hidden.unmount();
  });

  test('resolves author labels for system, current user, and other users', () => {
    const mounted = mountChatter();
    expect(mounted.q('.system-label')?.textContent).toBe('System');
    expect(mounted.q('.you-label')?.textContent).toBe('Tester');
    expect(mounted.q('.other-label')?.textContent).toBe('usr_other');
    mounted.unmount();

    useAuthStore().currentUser = { Id: 'usr_test', Name: '  ' } as any;
    const unnamed = mountChatter();
    expect(unnamed.q('.you-label')?.textContent).toBe('You');
    unnamed.unmount();

    useAuthStore().currentUser = { Id: 'usr_test' } as any;
    const missingName = mountChatter();
    expect(missingName.q('.you-label')?.textContent).toBe('You');
    missingName.unmount();

    useAuthStore().currentUser = { Id: '  ', Name: 'Tester' } as any;
    const noCurrentId = mountChatter();
    expect(noCurrentId.q('.you-label')?.textContent).toBe('usr_test');
    noCurrentId.unmount();

    useAuthStore().currentUser = null;
    const loggedOut = mountChatter();
    expect(loggedOut.q('.you-label')?.textContent).toBe('usr_test');
    expect(loggedOut.q('.system-label')?.textContent).toBe('System');
    loggedOut.unmount();
  });
});
