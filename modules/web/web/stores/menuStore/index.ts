// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, computed, inject, getCurrentInstance } from 'vue';
import { useRoute } from 'vue-router';
import { defineStore } from 'pinia';
import { MenuSymbol, getInstalledMenu } from '@/core/web/menu';
import type { Menu, MenuItem } from '@/core/web/menu';

/**
 * Resolves the live Menu registry.
 * Store setup may run outside a component (router redirect, effectScope);
 * inject is empty then, so fall back to the installed plugin singleton.
 */
function resolveMenuManager(): Menu | null {
  if (getCurrentInstance()) {
    try {
      const injected = inject(MenuSymbol, null) as Menu | null;
      if (injected) return injected;
    } catch {
      // Inject still failed despite an instance.
    }
  }
  return getInstalledMenu();
}

export const useMenuStore = defineStore('menu', () => {
  const activeMenuId = ref<string | null>(null);

  // Capture the reactive route during store setup when inject is available.
  // Re-try on later reads so a store created in a router redirect can still
  // pick up the route once a component render provides injection.
  let capturedRoute: { path?: string } | undefined;
  try {
    capturedRoute = useRoute();
  } catch {
    capturedRoute = undefined;
  }

  /**
   * Reads the current route path when vue-router injection is available.
   */
  function currentRoutePath(): string {
    if (capturedRoute && typeof capturedRoute.path === 'string') {
      return capturedRoute.path;
    }
    try {
      const route = useRoute();
      if (route) capturedRoute = route;
      return String(route?.path || '');
    } catch {
      return String(capturedRoute?.path || '');
    }
  }

  /**
   * Shared lookup for the first navigable menu item.
   */
  function findFirstNavigableMenu(menuItem: MenuItem | null | undefined): MenuItem | null {
    if (!menuItem) return null;

    // Return leaf items with a path immediately.
    if (menuItem.path && (!menuItem.children || menuItem.children.length === 0)) return menuItem;

    // Traverse children recursively, skipping hidden or disabled entries.
    if (Array.isArray(menuItem.children)) {
      for (const child of menuItem.children) {
        if (child.hidden || child.disabled) continue;
        const hit = findFirstNavigableMenu(child);
        if (hit) return hit;
      }
    }

    // Fall back to the current item if it is navigable.
    return menuItem.path ? menuItem : null;
  }

  /**
   * Walk up the path to find a menu, also handling routes without the /web prefix.
   */
  function findMenuByPathWithFallback(rawPath: string): MenuItem | null {
    const menuManager = resolveMenuManager();
    if (!menuManager) return null;

    const normalize = (p: string) => {
      // Trim a trailing slash so path comparisons stay consistent.
      if (p.length > 1 && p.endsWith('/')) return p.slice(0, -1);
      return p;
    };
    const stripWeb = (p: string) => (p.startsWith('/web/') ? p.replace(/^\/web(?=\/)/, '') : p);

    let p = normalize(rawPath);
    while (p && p !== '/') {
      // 1) Match the raw path.
      let found = menuManager.getMenuByPath(p);
      // 2) Match again after removing the /web prefix.
      if (!found) {
        const stripped = stripWeb(p);
        if (stripped !== p) {
          found = menuManager.getMenuByPath(stripped);
        }
      }
      if (found) return found;

      // Move up one path segment.
      const idx = p.lastIndexOf('/');
      if (idx <= 0) break;
      p = p.slice(0, idx);
    }
    // Final attempt at the root path, which usually has no menu entry.
    return null;
  }

  /**
   * Active menu synchronized from routing with upward path fallback.
   * It prefers activeMenuId when present; otherwise it walks up the current route
   * path, selects the first navigable match, and syncs activeMenuId to that entry.
   */
  const activeMenu = computed(() => {
    const menuManager = resolveMenuManager();
    if (!menuManager) return null;

    // Prefer the manually assigned activeMenuId.
    if (activeMenuId.value) {
      return menuManager.getMenu(activeMenuId.value);
    }

    // Match by walking up the current route path.
    const fromRoute = findMenuByPathWithFallback(currentRoutePath());
    const navigable = findFirstNavigableMenu(fromRoute);

    if (navigable) {
      // Keep activeMenuId aligned for later reads.
      activeMenuId.value = navigable.id;
      return navigable;
    }

    return null;
  });

  const activeApp = computed(() => {
    if (!activeMenu.value) return null;
    return findAppRoot(activeMenu.value);
  });

  /**
   * Sets the active menu id, or clears it when null.
   */
  function setActiveMenu(menu: MenuItem | null) {
    activeMenuId.value = menu ? menu.id : null;
  }

  /**
   * Walks to the app-root ancestor of a menu item.
   */
  function findAppRoot(menu: MenuItem): MenuItem | null {
    let current: MenuItem | null = menu;
    while (current) {
      if (!current.__parent) {
        return current;
      }
      current = current.__parent;
    }
    return null;
  }

  return {
    activeMenu,
    activeMenuId,
    activeApp,

    // Methods.
    setActiveMenu,

    // Proxies for menuManager methods (no-op safe when Menu inject is missing).
    hasMenu: (id: string) => !!resolveMenuManager()?.hasMenu?.(id),
    getMenu: (id: string) => resolveMenuManager()?.getMenu?.(id),
    getMenuByPath: (path: string) => resolveMenuManager()?.getMenuByPath?.(path),
    getMenuChildren: (id: string) => resolveMenuManager()?.getMenuChildren?.(id) ?? [],
    getMenuParent: (id: string) => resolveMenuManager()?.getMenuParent?.(id) ?? null,
    getMenus: () => resolveMenuManager()?.getMenus?.() ?? [],

    // Helper utilities.
    findFirstNavigableMenu,
  };
});
