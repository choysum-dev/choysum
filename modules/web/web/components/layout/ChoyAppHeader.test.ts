// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { pinViewportWidth } from '../../stores/layoutStore/pinViewport';
import ChoyAppShell from './ChoyAppShell.vue';
import sourceMessages from '../../i18n/source';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

describe('ChoyAppHeader', () => {
  test('renders menu trigger inside the app shell header', async () => {
    pinViewportWidth(1280);
    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/', fullPath: '/', meta: {} },
    });
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages: { en: sourceMessages as any },
    });
    const mounted = mountApp(ChoyAppShell as any, {
      plugins: [pinia, router, i18n],
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    await nextTick();
    expect(mounted.q('[data-testid=choy-shell-menu-trigger]')).not.toBeNull();
    mounted.click('[data-testid=choy-shell-menu-trigger]');
    await flushPromises();
    mounted.unmount();
  });

  test('renders menu trigger when layout store is unavailable', async () => {
    pinViewportWidth(1280);
    setActivePinia(undefined as any);
    const { router } = createFeStubRouter({
      route: { path: '/', fullPath: '/', meta: {} },
    });
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages: { en: sourceMessages as any },
    });
    const mounted = mountApp(ChoyAppShell as any, {
      plugins: [router, i18n],
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    await nextTick();
    expect(mounted.q('[data-testid=choy-shell-menu-trigger]')).not.toBeNull();
    mounted.click('[data-testid=choy-shell-menu-trigger]');
    await flushPromises();
    mounted.unmount();
  });
});
