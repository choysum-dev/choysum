// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyPage from './ChoyPage.vue';
import { blurFocusedDescendant } from './choy_page_loading_focus';

describe('ChoyPage loading focus', () => {
  test('uses inert without aria-hidden so focused submit is not under aria-hidden', async () => {
    const wrapper = mountApp(ChoyPage, {
      reactiveProps: true,
      props: { loading: false, title: 'Login' },
      slots: {
        default: () => h('button', { type: 'button', class: 'submit-button' }, 'Log In'),
      },
    });

    const root = wrapper.q('[data-anchor="choy.page"]');
    const btn = wrapper.q('.submit-button') as HTMLButtonElement | null;
    expect(root).not.toBeNull();
    expect(btn).not.toBeNull();
    btn!.focus();
    expect(document.activeElement).toBe(btn);

    // Prove the page root is a valid blur target for the focused submit (the
    // watcher path). Instrument blur because choysum's minimal DOM may leave
    // activeElement unchanged after native blur().
    let blurred = false;
    btn!.blur = () => {
      blurred = true;
    };
    blurFocusedDescendant(root, { activeElement: btn });
    expect(blurred).toBe(true);

    wrapper.props.loading = true;
    await flushPromises();

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
  });

  test('mounts with loading true using inert and aria-busy', async () => {
    const wrapper = mountApp(ChoyPage, {
      props: { loading: true, title: 'Busy' },
      slots: {
        default: () => h('button', { type: 'button', class: 'submit-button' }, 'Go'),
      },
    });
    await flushPromises();

    const body = wrapper.q('.choy-page__body');
    expect(body?.hasAttribute('inert')).toBe(true);
    expect(body?.getAttribute('aria-hidden')).toBeNull();
    expect(wrapper.q('[aria-busy="true"]')).not.toBeNull();

    wrapper.unmount();
  });
});
