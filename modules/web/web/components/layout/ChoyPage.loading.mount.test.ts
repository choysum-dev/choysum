// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyPage from './ChoyPage.vue';

describe('ChoyPage loading focus', () => {
  test('uses inert without aria-hidden so focused submit is not under aria-hidden', async () => {
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

    wrapper.props.loading = true;
    await flushPromises();

    const body = wrapper.q('.choy-page__body');
    expect(body?.hasAttribute('inert')).toBe(true);
    // Prefer inert over aria-hidden during loading (Chromium blocks aria-hidden
    // when a descendant still holds focus, e.g. login submit → navigate home).
    expect(body?.getAttribute('aria-hidden')).toBeNull();
    const region = wrapper.q('[aria-busy="true"]');
    expect(region).not.toBeNull();

    wrapper.unmount();
  });
});
