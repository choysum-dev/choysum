// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyChatterFollowerBar from './ChoyChatterFollowerBar.vue';

describe('ChoyChatterFollowerBar', () => {
  test('emits follow/unfollow and formats follower counts', async () => {
    const follows: string[] = [];
    const mounted = mountApp(ChoyChatterFollowerBar as any, {
      props: {
        following: false,
        followerCount: 1,
        loading: false,
        canToggle: true,
      },
      reactiveProps: true,
      on: {
        onFollow: () => follows.push('follow'),
        onUnfollow: () => follows.push('unfollow'),
      },
    });
    await flushPromises();
    expect(mounted.text()).toContain('Follow');
    expect(mounted.text()).toMatch(/1/);
    expect(mounted.text()).toContain('follower');

    // DOM click is unreliable for SFC @click in FE-QJS; call toggle directly.
    const state = mounted.setupState() as { toggle: () => void };
    state.toggle();
    await flushPromises();
    expect(follows).toEqual(['follow']);

    mounted.props.following = true;
    mounted.props.followerCount = 3;
    await nextTick();
    expect(mounted.text()).toContain('Unfollow');
    expect(mounted.text()).toContain('followers');

    state.toggle();
    await flushPromises();
    expect(follows).toEqual(['follow', 'unfollow']);
    mounted.unmount();
  });

  test('coerces string followerCount for plural label', async () => {
    const mounted = mountApp(ChoyChatterFollowerBar as any, {
      props: {
        following: false,
        followerCount: '2' as unknown as number,
        canToggle: true,
      },
    });
    await flushPromises();
    expect(mounted.text()).toContain('followers');
    mounted.unmount();
  });

  test('disables toggle while loading or when canToggle is false', async () => {
    const follows: string[] = [];
    const mounted = mountApp(ChoyChatterFollowerBar as any, {
      props: { following: false, loading: true, canToggle: true },
      on: { onFollow: () => follows.push('follow') },
    });
    await flushPromises();
    expect((mounted.q('button') as HTMLButtonElement).disabled).toBe(true);
    const state = mounted.setupState() as { toggle: () => void };
    state.toggle();
    expect(follows).toEqual([]);
    mounted.unmount();

    const locked = mountApp(ChoyChatterFollowerBar as any, {
      props: { following: false, canToggle: false },
      on: { onFollow: () => follows.push('follow') },
    });
    await flushPromises();
    expect((locked.q('button') as HTMLButtonElement).disabled).toBe(true);
    (locked.setupState() as { toggle: () => void }).toggle();
    expect(follows).toEqual([]);
    locked.unmount();
  });
});
