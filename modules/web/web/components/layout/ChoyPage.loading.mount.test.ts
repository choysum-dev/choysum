// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyPage from './ChoyPage.vue';
import {
  blurFocusedDescendantCore,
  setBlurFocusedDescendantImplForTest,
} from './choy_page_loading_focus';

describe('ChoyPage loading focus', () => {
  test('loading transition uses inert without aria-hidden and blurs via watcher', async () => {
    let blurCalls = 0;
    const restore = setBlurFocusedDescendantImplForTest((root, doc) => {
      blurCalls++;
      blurFocusedDescendantCore(root, doc);
    });

    const wrapper = mountApp(ChoyPage, {
      reactiveProps: true,
      props: { loading: false, title: 'Login' },
      slots: {
        default: () => h('button', { type: 'button', class: 'submit-button' }, 'Log In'),
      },
    });

    const btn = wrapper.q('.submit-button') as HTMLButtonElement | null;
    expect(btn).not.toBeNull();
    btn!.focus();
    expect(document.activeElement).toBe(btn);

    // immediate:true may call once while rootEl is still null during setup.
    blurCalls = 0;
    wrapper.props.loading = false;
    await flushPromises();
    // Ensure a rising edge: false → true must hit the watcher (not only mount).
    wrapper.props.loading = true;
    await flushPromises();
    expect(blurCalls).toBeGreaterThan(0);

    const body = wrapper.q('.choy-page__body');
    expect(body?.hasAttribute('inert')).toBe(true);
    // Prefer inert over aria-hidden during loading (Chromium blocks aria-hidden
    // when a descendant still holds focus, e.g. login submit → navigate home).
    expect(body?.getAttribute('aria-hidden')).toBeNull();
    const region = wrapper.q('[aria-busy="true"]');
    expect(region).not.toBeNull();

    wrapper.props.loading = false;
    await flushPromises();
    expect(body?.hasAttribute('inert')).toBe(false);

    wrapper.unmount();
    restore();
  });

  test('mounts with loading true using inert and aria-busy', async () => {
    let blurCalls = 0;
    const restore = setBlurFocusedDescendantImplForTest((root, doc) => {
      blurCalls++;
      blurFocusedDescendantCore(root, doc);
    });

    const wrapper = mountApp(ChoyPage, {
      props: { loading: true, title: 'Busy' },
      slots: {
        default: () => h('button', { type: 'button', class: 'submit-button' }, 'Go'),
      },
    });
    await flushPromises();

    // onMounted (+ immediate) should blur once root is available.
    expect(blurCalls).toBeGreaterThan(0);
    const body = wrapper.q('.choy-page__body');
    expect(body?.hasAttribute('inert')).toBe(true);
    expect(body?.getAttribute('aria-hidden')).toBeNull();
    expect(wrapper.q('[aria-busy="true"]')).not.toBeNull();

    wrapper.unmount();
    restore();
  });
});
