// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { ElButton } from 'element-plus';

import { useAuthStore } from '@/auth/web/stores/auth';
import { GetFollowerStoreKey } from '@/web/web/composables/chatter/chatterStores';
import {
  flushPromises,
  fnRecorder,
  mountApp,
  restoreSfc,
  stubSfc,
} from '@/web/web/__tests__/mountApp';
import OChatterFollowerBar from './OChatterFollowerBar.vue';

describe('OChatterFollowerBar', () => {
  const SearchByRecord = fnRecorder(async () => [{ UserId: 'usr_1' }, { UserId: 'usr_2' }] as any[]);
  const Follow = fnRecorder(async () => ({ Id: 'f1' }));
  const Unfollow = fnRecorder(async () => 1);
  let pinia: ReturnType<typeof createPinia>;

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    useAuthStore().currentUser = { Id: 'usr_1', Name: 'Tester' } as any;

    SearchByRecord.mockReset();
    Follow.mockReset();
    Unfollow.mockReset();
    SearchByRecord.mockImplementation(async () => [{ UserId: 'usr_1' }, { UserId: 'usr_2' }]);
    Follow.mockImplementation(async () => ({ Id: 'f1' }));
    Unfollow.mockImplementation(async () => 1);

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
              class: 'follower-toggle',
              'data-disabled': props.disabled ? 'true' : 'false',
              disabled: props.disabled || undefined,
              onClick: () => emit('click'),
            },
            slots.default?.()
          );
      },
    });
  });

  afterEach(() => {
    restoreSfc(ElButton as any);
  });

  function mountBar(props?: Partial<{ model: string; resId: string; disabled: boolean }>) {
    return mountApp(OChatterFollowerBar as any, {
      props: {
        model: 'partner.Partner',
        resId: 'res1',
        ...props,
      },
      reactiveProps: true,
      plugins: [pinia],
      provide: {
        [GetFollowerStoreKey]: () => ({ SearchByRecord, Follow, Unfollow }),
      },
    });
  }

  test('loads follower state and shows the count', async () => {
    const mounted = mountBar();
    await flushPromises();
    expect(SearchByRecord.calls[0]).toEqual(['partner.Partner', 'res1', ['UserId']]);
    expect(mounted.text()).toContain('Unfollow');
    expect(mounted.text()).toContain('2 followers');
    mounted.unmount();
  });

  test('follows and unfollows the current record', async () => {
    let call = 0;
    SearchByRecord.mockImplementation(async () => {
      call += 1;
      return call === 1 ? [] : [{ UserId: 'usr_1' }];
    });
    const mounted = mountBar();
    await flushPromises();
    expect(mounted.text()).toContain('Follow');

    mounted.click('.follower-toggle');
    await flushPromises();
    expect(Follow.calls[0]?.[0]).toEqual({ Model: 'partner.Partner', ResId: 'res1' });
    expect(mounted.text()).toContain('Unfollow');

    mounted.click('.follower-toggle');
    await flushPromises();
    expect(Unfollow.calls[0]?.[0]).toEqual({ Model: 'partner.Partner', ResId: 'res1' });
    mounted.unmount();
  });

  test('clears state when the thread identity is empty', async () => {
    const mounted = mountBar({ model: '  ', resId: '  ' });
    await flushPromises();
    expect(SearchByRecord.calls.length).toBe(0);
    expect(mounted.q('.follower-toggle')?.getAttribute('data-disabled')).toBe('true');
    mounted.unmount();

    SearchByRecord.mockClear();
    const missingModel = mountBar({ model: '', resId: 'res1' });
    await flushPromises();
    expect(SearchByRecord.calls.length).toBe(0);
    expect(missingModel.q('.follower-toggle')?.getAttribute('data-disabled')).toBe('true');
    missingModel.unmount();

    const missingResId = mountBar({ model: 'partner.Partner', resId: '' });
    await flushPromises();
    expect(SearchByRecord.calls.length).toBe(0);
    expect(missingResId.q('.follower-toggle')?.getAttribute('data-disabled')).toBe('true');
    missingResId.unmount();
  });

  test('treats follower rows without UserId as not the current user', async () => {
    SearchByRecord.mockImplementation(async () => [{ UserId: null }, { UserId: '  ' }, {}]);
    const mounted = mountBar();
    await flushPromises();
    expect(mounted.text()).toContain('Follow');
    expect(mounted.text()).toContain('3 followers');
    mounted.unmount();
  });

  test('does not toggle when disabled or unauthenticated', async () => {
    useAuthStore().currentUser = null;
    const mounted = mountBar();
    await flushPromises();
    expect(mounted.q('.follower-toggle')?.getAttribute('data-disabled')).toBe('true');
    mounted.click('.follower-toggle');
    expect(Follow.calls.length).toBe(0);

    useAuthStore().currentUser = { Id: 'usr_1' } as any;
    mounted.props.disabled = true;
    await flushPromises();
    mounted.click('.follower-toggle');
    expect(Follow.calls.length).toBe(0);
    mounted.unmount();
  });

  test('ignores toggle clicks while loading', async () => {
    let resolveSearch: ((rows: unknown[]) => void) | undefined;
    SearchByRecord.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveSearch = resolve;
        })
    );
    const mounted = mountBar();
    await Promise.resolve();
    mounted.click('.follower-toggle');
    expect(Follow.calls.length).toBe(0);
    resolveSearch?.([]);
    await flushPromises();
    mounted.unmount();
  });
});
