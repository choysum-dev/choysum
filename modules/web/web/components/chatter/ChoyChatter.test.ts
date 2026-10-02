// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick, ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';

import { useAuthStore } from '@/auth/web/stores/auth';
import {
  GetFollowerStoreKey,
  GetMessageStoreKey,
} from '@/web/web/composables/chatter/chatterStores';
import { UseChatterTimelineKey } from '@/web/web/composables/chatter/useChatterTimeline';
import { UseChatterThreadTipsKey } from '@/web/web/composables/chatter/useChatterThreadTips';
import {
  flushPromises,
  fnRecorder,
  mountApp,
  restoreSfc,
  stubSfc,
} from '@/web/web/__tests__/mountApp';
import ChoyCard from '../layout/ChoyCard.vue';
import ChoyChatter from './ChoyChatter.vue';
import ChoyChatterComposer from './ChoyChatterComposer.vue';
import ChoyChatterFollowerBar from './ChoyChatterFollowerBar.vue';
import ChoyChatterTimeline from './ChoyChatterTimeline.vue';

describe('ChoyChatter', () => {
  const refresh = fnRecorder(async () => undefined);
  const entries = ref<any[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const SearchByRecord = fnRecorder(async () => [] as any[]);
  const Follow = fnRecorder(async () => ({ UserId: 'usr_test' }));
  const Unfollow = fnRecorder(async () => 1);
  const Post = fnRecorder(async () => ({ Id: 'm1' }));
  const composerClear = fnRecorder();
  let pinia: ReturnType<typeof createPinia>;

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    refresh.mockClear();
    composerClear.mockClear();
    entries.value = [];
    loading.value = false;
    error.value = null;
    SearchByRecord.mockReset();
    Follow.mockReset();
    Unfollow.mockReset();
    Post.mockReset();
    SearchByRecord.mockImplementation(async () => []);
    Follow.mockImplementation(async () => ({ UserId: 'usr_test' }));
    Unfollow.mockImplementation(async () => 1);
    Post.mockImplementation(async () => ({ Id: 'm1' }));
    useAuthStore().currentUser = { Id: 'usr_test', Name: 'Tester' } as any;

    stubSfc(ChoyCard as any, {
      name: 'ChoyCard',
      setup(_: any, { slots }: any) {
        return () =>
          h('div', { class: 'choy-chatter-card' }, [slots.header?.(), slots.default?.()]);
      },
    });
    stubSfc(ChoyChatterFollowerBar as any, {
      props: ['following', 'followerCount', 'loading', 'disabled', 'canToggle'],
      emits: ['follow', 'unfollow'],
      setup(props: any, { emit }: any) {
        return () =>
          h('div', { class: 'follower-bar' }, [
            h('span', { class: 'follower-state' }, props.following ? 'following' : 'idle'),
            h('span', { class: 'follower-count' }, String(props.followerCount ?? 0)),
            h('button', {
              type: 'button',
              class: 'follow-btn',
              onClick: () => emit('follow'),
            }),
            h('button', {
              type: 'button',
              class: 'unfollow-btn',
              onClick: () => emit('unfollow'),
            }),
          ]);
      },
    });
    stubSfc(ChoyChatterComposer as any, {
      props: ['disabled', 'posting', 'error'],
      emits: ['post'],
      setup(props: any, { emit, expose }: any) {
        expose({ clear: () => composerClear() });
        return () =>
          h('div', { class: 'composer' }, [
            h('span', { class: 'composer-error' }, props.error || ''),
            h('span', { class: 'composer-posting' }, props.posting ? 'posting' : 'idle'),
            h('button', {
              type: 'button',
              class: 'composer-post',
              onClick: () => emit('post', 'hello world'),
            }),
            h('button', {
              type: 'button',
              class: 'composer-post-empty',
              onClick: () => emit('post', '   '),
            }),
          ]);
      },
    });
    stubSfc(ChoyChatterTimeline as any, {
      props: ['entries', 'loading', 'error', 'resolveAuthorLabel'],
      setup(props: any) {
        return () =>
          h('div', { class: 'timeline' }, [
            h('span', { class: 'system-label' }, props.resolveAuthorLabel?.(null)),
            h('span', { class: 'you-label' }, props.resolveAuthorLabel?.('usr_test')),
            h('span', { class: 'other-label' }, props.resolveAuthorLabel?.('usr_other')),
            h('span', { class: 'timeline-loading' }, props.loading ? 'loading' : 'ready'),
            h('span', { class: 'timeline-error' }, props.error || ''),
            h('span', { class: 'timeline-count' }, String((props.entries || []).length)),
          ]);
      },
    });
  });

  afterEach(() => {
    restoreSfc(ChoyCard as any);
    restoreSfc(ChoyChatterFollowerBar as any);
    restoreSfc(ChoyChatterComposer as any);
    restoreSfc(ChoyChatterTimeline as any);
  });

  function storeProvide() {
    return {
      [UseChatterTimelineKey]: () => ({ entries, loading, error, refresh }),
      [UseChatterThreadTipsKey]: () => undefined,
      [GetMessageStoreKey]: () => ({ Post, SearchByRecord: async () => [] }),
      [GetFollowerStoreKey]: () => ({ SearchByRecord, Follow, Unfollow }),
    };
  }

  function mountChrome(props?: Record<string, unknown>) {
    const onPost = fnRecorder();
    const onFollow = fnRecorder();
    const onUnfollow = fnRecorder();
    const mounted = mountApp(ChoyChatter as any, {
      props: {
        model: 'partner.Partner',
        resId: 'res_1',
        entries: [{ kind: 'message', id: 'm1' }],
        following: false,
        followerCount: 2,
        currentUserId: 'usr_test',
        currentUserName: 'Tester',
        ...props,
      },
      reactiveProps: true,
      plugins: [pinia],
      provide: storeProvide(),
      on: {
        onPost: (body: string) => onPost(body),
        onFollow: () => onFollow(),
        onUnfollow: () => onUnfollow(),
      },
    });
    return { mounted, onPost, onFollow, onUnfollow };
  }

  function mountBound(props?: Record<string, unknown>) {
    return mountApp(ChoyChatter as any, {
      props: {
        model: 'partner.Partner',
        resId: 'res_1',
        bindStore: true,
        ...props,
      },
      reactiveProps: true,
      plugins: [pinia],
      provide: storeProvide(),
    });
  }

  test('chrome mode: renders props, emits post/follow/unfollow, hides composer', async () => {
    const { mounted, onPost, onFollow, onUnfollow } = mountChrome();
    expect(mounted.text()).toContain('Activity');
    expect(mounted.q('.timeline-count')?.textContent).toBe('1');
    expect(mounted.q('.follower-count')?.textContent).toBe('2');
    expect(mounted.q('.system-label')?.textContent).toBe('System');
    expect(mounted.q('.you-label')?.textContent).toBe('Tester');
    expect(mounted.q('.other-label')?.textContent).toBe('usr_other');

    mounted.click('.composer-post');
    mounted.click('.follow-btn');
    mounted.click('.unfollow-btn');
    await flushPromises();
    expect(onPost.calls[0]?.[0]).toBe('hello world');
    expect(onFollow.calls.length).toBe(1);
    expect(onUnfollow.calls.length).toBe(1);
    mounted.unmount();

    const titled = mountChrome({ title: 'Partner activity' }).mounted;
    expect(titled.text()).toContain('Partner activity');
    titled.unmount();

    const emptyRes = mountChrome({ resId: '' }).mounted;
    expect(emptyRes.q('.composer-post')).toBeFalsy();
    emptyRes.unmount();

    const disabled = mountChrome({ disabled: true }).mounted;
    expect(disabled.q('.composer-post')).toBeFalsy();
    disabled.unmount();

    const hidden = mountChrome({ showComposer: false }).mounted;
    expect(hidden.q('.composer-post')).toBeFalsy();
    hidden.unmount();
  });

  test('chrome mode: clears composer when posting flips false without error', async () => {
    const { mounted } = mountChrome({ posting: true });
    composerClear.mockClear();
    mounted.props.posting = false;
    await nextTick();
    await flushPromises();
    await Promise.resolve();
    expect(composerClear.calls.length).toBeGreaterThanOrEqual(1);
    mounted.unmount();
  });

  test('chrome mode: expose clear and prop author override', () => {
    const { mounted } = mountChrome({
      currentUserId: 'usr_other',
      currentUserName: 'Other',
    });
    expect(mounted.q('.you-label')?.textContent).toBe('usr_test');
    composerClear.mockClear();
    mounted.root.clear();
    expect(composerClear.calls.length).toBe(1);
    mounted.unmount();
  });

  test('bindStore: loads followers, posts, follows, and unfollows', async () => {
    SearchByRecord.mockImplementation(async () => [{ UserId: 'usr_test' }, { UserId: 'usr_2' }]);
    const mounted = mountBound();
    await flushPromises();
    expect(SearchByRecord.calls[0]?.[0]).toBe('partner.Partner');
    expect(mounted.q('.follower-state')?.textContent).toBe('following');
    expect(mounted.q('.follower-count')?.textContent).toBe('2');

    composerClear.mockClear();
    mounted.click('.composer-post');
    await flushPromises();
    expect(Post.calls[0]?.[0]).toEqual({
      Model: 'partner.Partner',
      ResId: 'res_1',
      Body: 'hello world',
    });
    expect(refresh.calls.length).toBeGreaterThanOrEqual(1);
    expect(composerClear.calls.length).toBeGreaterThanOrEqual(1);

    SearchByRecord.mockImplementation(async () => []);
    mounted.click('.unfollow-btn');
    await flushPromises();
    expect(Unfollow.calls[0]?.[0]).toEqual({ Model: 'partner.Partner', ResId: 'res_1' });

    SearchByRecord.mockImplementation(async () => [{ UserId: 'usr_test' }]);
    mounted.click('.follow-btn');
    await flushPromises();
    expect(Follow.calls[0]?.[0]).toEqual({ Model: 'partner.Partner', ResId: 'res_1' });
    mounted.unmount();
  });

  test('bindStore: ignores empty post body and surfaces post errors', async () => {
    Post.mockImplementation(async () => {
      throw new Error('boom');
    });
    const mounted = mountBound();
    await flushPromises();
    mounted.click('.composer-post-empty');
    await flushPromises();
    expect(Post.calls.length).toBe(0);

    mounted.click('.composer-post');
    await flushPromises();
    expect(mounted.q('.composer-error')?.textContent).toBe('boom');
    mounted.unmount();

    Post.mockImplementation(async () => {
      throw new Error('   ');
    });
    const fallback = mountBound();
    await flushPromises();
    fallback.click('.composer-post');
    await flushPromises();
    expect(fallback.q('.composer-error')?.textContent).toBe('Failed to post comment');
    fallback.unmount();
  });

  test('bindStore: does not clear draft or set error after thread switch mid-post', async () => {
    let resolvePost!: (value: unknown) => void;
    Post.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolvePost = resolve;
        }),
    );
    const mounted = mountBound({ resId: 'res_a' });
    await flushPromises();
    composerClear.mockClear();
    refresh.mockClear();
    mounted.click('.composer-post');
    await nextTick();
    mounted.props.resId = 'res_b';
    await flushPromises();
    // Model/resId watcher clears the composer once on switch.
    const clearsOnSwitch = composerClear.calls.length;
    resolvePost({ Id: 'm1' });
    await flushPromises();
    // Stale completion must not clear again or assign an error on the new thread.
    expect(composerClear.calls.length).toBe(clearsOnSwitch);
    expect(mounted.q('.composer-error')?.textContent).toBe('');
    expect(refresh.calls.length).toBe(0);
    mounted.unmount();
  });

  test('bindStore: discards stale follower search results and handles search failure', async () => {
    const pending: Array<(rows: any[]) => void> = [];
    SearchByRecord.mockImplementation(
      () =>
        new Promise((resolve) => {
          pending.push(resolve);
        }),
    );
    const mounted = mountBound({ resId: 'res_a' });
    await nextTick();
    mounted.props.resId = 'res_b';
    await nextTick();
    expect(pending.length).toBeGreaterThanOrEqual(2);
    // Resolve the obsolete search last with a large count — must not win.
    pending[1]!([{ UserId: 'usr_test' }]);
    await flushPromises();
    expect(mounted.q('.follower-count')?.textContent).toBe('1');
    pending[0]!([{ UserId: 'usr_test' }, { UserId: 'usr_2' }, { UserId: 'usr_3' }]);
    await flushPromises();
    expect(mounted.q('.follower-count')?.textContent).toBe('1');
    mounted.unmount();

    SearchByRecord.mockImplementation(async () => {
      throw new Error('search failed');
    });
    const failed = mountBound();
    await flushPromises();
    expect(failed.q('.follower-state')?.textContent).toBe('idle');
    expect(failed.q('.follower-count')?.textContent).toBe('0');
    failed.unmount();
  });

  test('bindStore: follow/unfollow failures keep prior snapshot; empty model clears', async () => {
    SearchByRecord.mockImplementation(async () => [{ UserId: 'usr_test' }]);
    const mounted = mountBound();
    await flushPromises();
    expect(mounted.q('.follower-state')?.textContent).toBe('following');

    Unfollow.mockImplementation(async () => {
      throw new Error('unfollow failed');
    });
    mounted.click('.unfollow-btn');
    await flushPromises();
    expect(mounted.q('.follower-state')?.textContent).toBe('following');

    Follow.mockImplementation(async () => {
      throw new Error('follow failed');
    });
    mounted.click('.follow-btn');
    await flushPromises();

    mounted.props.resId = '';
    await flushPromises();
    expect(mounted.q('.follower-count')?.textContent).toBe('0');
    expect(mounted.q('.composer-post')).toBeFalsy();
    mounted.unmount();
  });

  test('bindStore: author labels fall back to auth store when props omitted', async () => {
    const mounted = mountBound();
    await flushPromises();
    expect(mounted.q('.you-label')?.textContent).toBe('Tester');
    mounted.unmount();

    useAuthStore().currentUser = { Id: 'usr_test', Name: '  ' } as any;
    const unnamed = mountBound();
    await flushPromises();
    expect(unnamed.q('.you-label')?.textContent).toBe('You');
    unnamed.unmount();
  });

  test('bindStore: skips toggle when disabled or missing user', async () => {
    SearchByRecord.mockImplementation(async () => []);
    useAuthStore().currentUser = null;
    const noUser = mountBound();
    await flushPromises();
    const followCallsBefore = Follow.calls.length;
    noUser.click('.follow-btn');
    await flushPromises();
    expect(Follow.calls.length).toBe(followCallsBefore);
    noUser.unmount();

    useAuthStore().currentUser = { Id: 'usr_test', Name: 'Tester' } as any;
    const disabled = mountBound({ disabled: true });
    await flushPromises();
    const before = Follow.calls.length;
    disabled.click('.follow-btn');
    await flushPromises();
    expect(Follow.calls.length).toBe(before);
    disabled.unmount();
  });

  test('bindStore: timeline error/loading and non-Error post failure', async () => {
    loading.value = true;
    error.value = 'tl failed';
    const mounted = mountBound();
    await flushPromises();
    expect(mounted.q('.timeline-loading')?.textContent).toBe('loading');
    expect(mounted.q('.timeline-error')?.textContent).toBe('tl failed');
    mounted.unmount();

    Post.mockImplementation(async () => {
      throw 'raw';
    });
    const raw = mountBound();
    await flushPromises();
    raw.click('.composer-post');
    await flushPromises();
    expect(raw.q('.composer-error')?.textContent).toBe('Failed to post comment');
    raw.unmount();
  });

  test('chrome mode: posting watcher skips clear when postError is set', async () => {
    const { mounted } = mountChrome({ posting: true, postError: 'nope' });
    composerClear.mockClear();
    mounted.props.posting = false;
    await nextTick();
    await flushPromises();
    await Promise.resolve();
    expect(composerClear.calls.length).toBe(0);
    mounted.unmount();
  });
});
