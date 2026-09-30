// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { useLayoutStore } from './index';
import { pinViewportWidth } from './pinViewport';

describe('useLayoutStore', () => {
  beforeEach(() => {
    pinViewportWidth(1280);
    setActivePinia(createPinia());
  });

  test('exposes device flags and default sidebar modes for each class', () => {
    const store = useLayoutStore();
    expect(store.deviceType).toBe('desktop');
    expect(store.isDesktop).toBe(true);
    expect(store.isMobile).toBe(false);
    expect(['expanded', 'collapsed', 'hidden', 'hover']).toContain(store.sidebarMode);
  });

  test('toggleSidebar cycles expanded → collapsed → expanded on desktop', () => {
    const store = useLayoutStore();
    store.setSidebarMode('expanded', { isUserAction: true });
    store.toggleSidebar();
    expect(store.sidebarMode).toBe('collapsed');
    store.toggleSidebar();
    expect(store.sidebarMode).toBe('expanded');
  });

  test('closeSidebar does not hide desktop expanded rail', () => {
    const store = useLayoutStore();
    store.setSidebarMode('expanded', { isUserAction: true });
    store.closeSidebar();
    expect(store.sidebarMode).toBe('expanded');
  });

  test('mobile defaults to hidden and closeSidebar hides expanded drawer', () => {
    pinViewportWidth(500);
    setActivePinia(createPinia());
    const store = useLayoutStore();
    expect(store.isMobile).toBe(true);
    expect(store.sidebarMode).toBe('hidden');
    store.setSidebarMode('expanded', { isUserAction: true });
    store.closeSidebar();
    expect(store.sidebarMode).toBe('hidden');
  });
});
