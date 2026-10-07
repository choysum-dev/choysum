// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { nextTick, h } from 'vue';
import ChoyAppShell from './ChoyAppShell.vue';
import { pinViewportWidth } from '../../stores/layoutStore/pinViewport';

describe('ChoyAppShell mobile sidebar sheet', () => {
  test('shows mobile sheet when layout store expands and hides when store collapses', async () => {
    pinViewportWidth(500);
    const { createPinia, setActivePinia } = await import('pinia');
    const { createI18n } = await import('vue-i18n');
    const createFeStubRouter = (await import('vue-router') as any).createFeStubRouter;
    const sourceMessages = (await import('../../i18n/source')).default;

    const pinia = createPinia();
    setActivePinia(pinia);
    const { useLayoutStore } = await import('../../stores/layoutStore');
    const layout = useLayoutStore();
    expect(layout.isMobile).toBe(true);

    layout.setSidebarMode('expanded', { isUserAction: true });

    const { router } = createFeStubRouter({
      route: { path: '/app', fullPath: '/app', meta: {} },
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

    expect(mounted.q('[data-mobile=true]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-menu-trigger]')).not.toBeNull();
    expect(layout.sidebarMode).toBe('expanded');
    expect(mounted.q('[data-slot=sheet]')?.getAttribute('data-state')).toBe('open');

    layout.setSidebarMode('hidden', { isUserAction: true });
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('hidden');
    // Store→bridge closes the mobile Sheet (stub exposes data-state on DialogRoot).
    expect(mounted.q('[data-slot=sheet]')?.getAttribute('data-state')).toBe('closed');

    layout.setSidebarMode('expanded', { isUserAction: true });
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('expanded');
    expect(mounted.q('[data-slot=sheet]')?.getAttribute('data-state')).toBe('open');
    expect(mounted.q('[data-mobile=true]')).not.toBeNull();

    mounted.unmount();
  });
});

describe('ChoyAppShell without layout store', () => {
  test('tolerates missing layout store and still renders chrome', async () => {
    const { setActivePinia } = await import('pinia');
    setActivePinia(undefined as any);
    const mounted = mountApp(ChoyAppShell as any, {
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(
      mounted.q('[data-testid=choy-shell-brand]'),
    ).not.toBeNull();
    mounted.unmount();
  });
});
