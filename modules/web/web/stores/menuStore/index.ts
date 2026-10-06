// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, computed, inject, getCurrentInstance, watch } from 'vue';
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
  const routePath = ref('');

  // Capture the reactive route during store setup when inject is available.
  // Re-try on later reads so a store created in a router redirect can still
  // pick up the route once a component render provides injection.
  let capturedRoute: { path?: string } | undefined;
  try {
    capturedRoute = useRoute();
    routePath.value = String(capturedRoute?.path || '');
  } catch {
    capturedRoute = undefined;
  }

  /**
   * Binds vue-router injection when a component finally has it.
   * Store setup may have run in a guard with no route; this invalidates
   * the cached activeMenu computed once a path is available.
   */
  function bindRouteFromInjection(): void {
    try {
      const route = capturedRoute ?? useRoute();
      if (!route) {
        return;
      }
      capturedRoute = route;
      const next = String(route.path || '');
      if (routePath.value !== next) {
        routePath.value = next;
      }
    } catch {
      // Route inject is still missing.
    }
  }

  /**
   * Reads the current route path when vue-router injection is available.
   */
  function currentRoutePath(): string {
    if (capturedRoute && typeof capturedRoute.path === 'string') {
      return capturedRoute.path;
    }
    return routePath.value;
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
   * Prefers activeMenuId when present; otherwise walks up the current route path.
   */
  const activeMenu = computed(() => {
    void routePath.value;
    const menuManager = resolveMenuManager();
    if (!menuManager) return null;

    if (activeMenuId.value) {
      return menuManager.getMenu(activeMenuId.value) ?? null;
    }

    const fromRoute = findMenuByPathWithFallback(currentRoutePath());
    return findFirstNavigableMenu(fromRoute);
  });

  const activeApp = computed(() => {
    if (!activeMenu.value) return null;
    return findAppRoot(activeMenu.value);
  });

  // Keep activeMenuId aligned outside the computed so evaluating activeMenu
  // during render cannot retrigger Vue's update cycle.
  watch(
    () => activeMenu.value?.id ?? null,
    (nextId) => {
      if (activeMenuId.value !== nextId) {
        activeMenuId.value = nextId;
      }
    },
    { immediate: true, flush: 'sync' },
  );

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
    let hops = 0;
    while (current && hops < 64) {
      hops += 1;
      if (!current.__parent) {
        return current;
      }
      current = current.__parent;
    }
    // Hop cap means a cyclic __parent chain; do not treat a mid-chain node as the app root.
    return null;
  }

  return {
    activeMenu,
    activeMenuId,
    activeApp,

    // Methods.
    setActiveMenu,

    bindRouteFromInjection,

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
