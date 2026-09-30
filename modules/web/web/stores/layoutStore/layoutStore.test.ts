// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { useLayoutStore } from './index';

describe('useLayoutStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  test('exposes device flags and default sidebar modes for each class', () => {
    const store = useLayoutStore();
    expect(['mobile', 'tablet', 'desktop']).toContain(store.deviceType);
    expect(['expanded', 'collapsed', 'hidden', 'hover']).toContain(store.sidebarMode);
    expect(typeof store.isMobile).toBe('boolean');
    expect(typeof store.isTablet).toBe('boolean');
    expect(typeof store.isDesktop).toBe('boolean');
  });

  test('toggleSidebar cycles expanded → collapsed → expanded on non-mobile', () => {
    const store = useLayoutStore();
    store.setSidebarMode('expanded', { isUserAction: true });
    store.toggleSidebar();
    expect(store.sidebarMode).toBe('collapsed');
    store.toggleSidebar();
    // On mobile, collapsed toggles to hidden; otherwise expanded.
    if (store.isMobile) {
      expect(store.sidebarMode).toBe('hidden');
    } else {
      expect(store.sidebarMode).toBe('expanded');
    }
  });

  test('closeSidebar hides only mobile expanded drawer', () => {
    const store = useLayoutStore();
    store.setSidebarMode('expanded', { isUserAction: true });
    store.closeSidebar();
    if (store.isMobile) {
      expect(store.sidebarMode).toBe('hidden');
    } else {
      expect(store.sidebarMode).toBe('expanded');
    }
  });
});
