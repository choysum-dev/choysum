// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyChatterComposer from './ChoyChatterComposer.vue';

type ComposerExposed = {
  setDraft: (text: string) => void;
  submit: () => void;
  clear: () => void;
};

describe('ChoyChatterComposer', () => {
  test('posts trimmed body, clears draft, and shows posting label', async () => {
    const posts: string[] = [];
    const mounted = mountApp(ChoyChatterComposer as any, {
      props: { posting: false },
      reactiveProps: true,
      on: { onPost: (body: string) => posts.push(body) },
    });
    await flushPromises();

    const ta = mounted.q('textarea') as HTMLTextAreaElement | null;
    expect(ta).toBeTruthy();
    expect(String(ta!.getAttribute('placeholder') || '')).toContain('Write a comment');

    const api = mounted.root as ComposerExposed;
    api.setDraft('  hello  ');
    await nextTick();
    api.submit();
    await flushPromises();
    expect(posts).toEqual(['hello']);

    api.clear();
    await nextTick();
    // After clear, empty submit is a no-op.
    api.submit();
    await flushPromises();
    expect(posts).toEqual(['hello']);

    mounted.props.posting = true;
    await nextTick();
    expect(mounted.text()).toContain('Posting');
    // Guard: submit must be a no-op while a post is already in flight.
    api.setDraft('second');
    api.submit();
    await flushPromises();
    expect(posts).toEqual(['hello']);
    mounted.unmount();
  });

  test('respects placeholder/postLabel props and skips empty submit', async () => {
    const posts: string[] = [];
    const mounted = mountApp(ChoyChatterComposer as any, {
      props: {
        placeholder: 'Say something',
        postLabel: 'Send',
        disabled: false,
      },
      on: { onPost: (body: string) => posts.push(body) },
    });
    await flushPromises();
    expect(mounted.q('textarea')?.getAttribute('placeholder')).toBe('Say something');
    expect(mounted.text()).toContain('Send');

    (mounted.root as ComposerExposed).submit();
    await flushPromises();
    expect(posts).toEqual([]);
    mounted.unmount();
  });

  test('blank placeholder/postLabel fall back to defaults', async () => {
    const mounted = mountApp(ChoyChatterComposer as any, {
      props: {
        placeholder: '   ',
        postLabel: '',
      },
    });
    await flushPromises();
    expect(String(mounted.q('textarea')?.getAttribute('placeholder') || '')).toContain('Write a comment');
    expect(mounted.text()).toContain('Post');
    mounted.unmount();
  });
});
