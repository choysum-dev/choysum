// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { pinViewportWidth } from '../../stores/layoutStore/pinViewport';
import ChoyAppHeader from './ChoyAppHeader.vue';
import ChoyAppShell from './ChoyAppShell.vue';
import ChoyCommandPalette from './ChoyCommandPalette.vue';
import ChoyShellBreadcrumb from './ChoyShellBreadcrumb.vue';
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

  test('menu toggle is a no-op when SidebarProvider is missing', async () => {
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
    stubSfc(ChoyShellBreadcrumb as any, {
      setup: () => () => h('div', { 'data-stub': 'breadcrumb' }),
    });
    stubSfc(ChoyCommandPalette as any, {
      setup: () => () => h('div', { 'data-stub': 'palette' }),
    });
    try {
      const mounted = mountApp(ChoyAppHeader as any, {
        props: { showSidebarChrome: true, menuTriggerLabel: 'Menu' },
        plugins: [pinia, router, i18n],
      });
      await flushPromises();
      await nextTick();
      expect(mounted.q('[data-testid=choy-shell-menu-trigger]')).not.toBeNull();
      mounted.click('[data-testid=choy-shell-menu-trigger]');
      await flushPromises();
      mounted.unmount();
    } finally {
      restoreSfc(ChoyShellBreadcrumb as any);
      restoreSfc(ChoyCommandPalette as any);
    }
  });
});
