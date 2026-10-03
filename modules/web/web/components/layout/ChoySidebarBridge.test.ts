// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { pinViewportWidth } from '../../stores/layoutStore/pinViewport';
import { SidebarProvider, useSidebar } from '../vendor/ui/sidebar/index';
import ChoySidebarBridge from './ChoySidebarBridge.vue';

type SidebarApi = ReturnType<typeof useSidebar>;

async function mountBridge(opts?: { mobile?: boolean }) {
  pinViewportWidth(opts?.mobile ? 500 : 1280);
  const pinia = createPinia();
  setActivePinia(pinia);
  const { useLayoutStore } = await import('../../stores/layoutStore');
  const layout = useLayoutStore();

  let sidebarApi: SidebarApi | null = null;
  const Host = defineComponent({
    setup() {
      return () =>
        h(SidebarProvider, null, {
          default: () =>
            h(
              defineComponent({
                setup() {
                  sidebarApi = useSidebar();
                  return () => h(ChoySidebarBridge);
                },
              }),
            ),
        });
    },
  });
  const mounted = mountApp(Host as any, { plugins: [pinia] });
  await flushPromises();
  await nextTick();
  if (!sidebarApi) throw new Error('expected sidebar api');
  return { mounted, layout, sidebar: sidebarApi };
}

describe('ChoySidebarBridge', () => {
  test('desktop store↔sidebar sync covers open and no-op branches', async () => {
    const { mounted, layout, sidebar } = await mountBridge({ mobile: false });
    expect(layout.isMobile).toBe(false);

    layout.setSidebarMode('expanded', { isUserAction: true });
    await flushPromises();
    await nextTick();
    // Same mode again → open already matches (no-op setOpen).
    layout.setSidebarMode('expanded', { isUserAction: true });
    await flushPromises();
    await nextTick();
    expect(sidebar.open.value).toBe(true);

    sidebar.setOpen(false);
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('collapsed');

    // Identical open state → storeOpen === isOpen early return.
    sidebar.setOpen(false);
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('collapsed');

    sidebar.setOpen(true);
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('expanded');

    expect(mounted.q('[data-testid=choy-sidebar-bridge]')).not.toBeNull();
    mounted.unmount();
  });

  test('mobile store↔sidebar sync covers openMobile paths', async () => {
    const { mounted, layout, sidebar } = await mountBridge({ mobile: true });
    expect(layout.isMobile).toBe(true);

    layout.setSidebarMode('expanded', { isUserAction: true });
    await flushPromises();
    await nextTick();
    layout.setSidebarMode('expanded', { isUserAction: true });
    await flushPromises();
    await nextTick();
    expect(sidebar.openMobile.value).toBe(true);

    sidebar.setOpenMobile(false);
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('hidden');

    sidebar.setOpenMobile(true);
    await flushPromises();
    await nextTick();
    expect(layout.sidebarMode).toBe('expanded');

    mounted.unmount();
  });

  test('tolerates missing layout store', async () => {
    pinViewportWidth(1280);
    setActivePinia(undefined as any);
    const Host = defineComponent({
      setup() {
        return () =>
          h(SidebarProvider, null, {
            default: () => h(ChoySidebarBridge),
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.q('[data-testid=choy-sidebar-bridge]')).not.toBeNull();
    mounted.unmount();
  });
});
