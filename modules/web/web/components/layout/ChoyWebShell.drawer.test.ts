// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { nextTick, h } from 'vue';
import ChoyWebShell from './ChoyWebShell.vue';
import { pinViewportWidth } from '../../stores/layoutStore/pinViewport';

describe('ChoyWebShell mobile drawer', () => {
  test('opens drawer, locks body scroll, closes on Escape and close control', async () => {
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

    const mounted = mountApp(ChoyWebShell as any, {
      plugins: [pinia, router, i18n],
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    await nextTick();

    expect(document.body.style.overflow || '').toBe('hidden');
    expect(mounted.q('[data-testid=choy-layout-aside]')?.getAttribute('role')).toBe('dialog');
    expect(mounted.q('[data-testid=choy-layout-aside]')?.getAttribute('aria-modal')).toBe('true');
    expect(mounted.q('[data-testid=choy-shell-drawer-close]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-layout-aside-backdrop]')).not.toBeNull();

    const state = mounted.setupState() as any;

    // Escape closes the drawer (covers onDocumentKeydown).
    const onKey = state?.onDocumentKeydown as
      | ((e: { key: string; defaultPrevented?: boolean }) => void)
      | undefined;
    expect(typeof onKey).toBe('function');
    onKey!({ key: 'Escape' });
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('hidden');
    expect(document.body.style.overflow || '').toBe('');

    // Backdrop → @aside-dismiss → closeMobileRail (native click; ChoyButton DOM click is unreliable).
    layout.setSidebarMode('expanded', { isUserAction: true });
    await flushPromises();
    await nextTick();
    expect(document.body.style.overflow || '').toBe('hidden');
    (mounted.q('[data-testid=choy-layout-aside-backdrop]') as HTMLElement).click();
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('hidden');
    expect(document.body.style.overflow || '').toBe('');

    // closeMobileRail is the same handler wired to the drawer X button.
    layout.setSidebarMode('expanded', { isUserAction: true });
    await flushPromises();
    await nextTick();
    const closer = state?.closeMobileRail as (() => void) | undefined;
    expect(typeof closer).toBe('function');
    closer!();
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('hidden');

    // Menu trigger opens the drawer from hidden.
    layout.setSidebarMode('hidden', { isUserAction: true });
    await flushPromises();
    await nextTick();
    const handle = state?.onMenuTriggerClick as (() => void) | undefined;
    expect(typeof handle).toBe('function');
    handle!();
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('expanded');

    // Hide before unmount so body scroll lock is cleared by the railIsDrawer watch;
    // onUnmounted also clears overflow as a safety net.
    layout.setSidebarMode('hidden', { isUserAction: true });
    await flushPromises();
    await nextTick();
    expect(document.body.style.overflow || '').toBe('');
    // Unmount while overflow is already clear; separately force-lock then clear via helper
    // is covered in choyWebShellChrome tests. Call onUnmounted path with drawer still open:
    layout.setSidebarMode('expanded', { isUserAction: true });
    await flushPromises();
    await nextTick();
    expect(document.body.style.overflow || '').toBe('hidden');
    mounted.unmount();
    // Safety-net clear in onUnmounted should unlock scroll.
    expect(document.body.style.overflow || '').toBe('');
  });
});

describe('ChoyWebShell without layout store', () => {
  test('tolerates missing layout store and still renders chrome', async () => {
    const { setActivePinia } = await import('pinia');
    // Clear any leftover pinia so useLayoutStore throws into the catch path.
    setActivePinia(undefined as any);
    const mounted = mountApp(ChoyWebShell as any, {
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-brand]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-menu-trigger]')).toBeNull();
    mounted.unmount();
  });
});

