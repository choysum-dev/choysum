// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, computed, inject } from 'vue';
import { useRoute } from 'vue-router';
import { defineStore } from 'pinia';
import { MenuSymbol } from '@/core/web/menu';
import type { Menu, MenuItem } from '@/core/web/menu';

export const useMenuStore = defineStore('menu', () => {
  const route = useRoute();

  // Core state.
  const activeMenuId = ref<string | null>(null);

  // Inject directly during setup.
  const menuManager = inject(MenuSymbol) as Menu;
  // Shared lookup for the first navigable menu item.
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

  // Walk up the path to find a menu, also handling routes without the /web prefix.
  function findMenuByPathWithFallback(rawPath: string): MenuItem | null {
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
    if (!menuManager) return null;

    // Prefer the manually assigned activeMenuId.
    if (activeMenuId.value) {
      return menuManager.getMenu(activeMenuId.value);
    }

    // Match by walking up the current route path.
    const fromRoute = findMenuByPathWithFallback(route.path);
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

  // State management methods.
  function setActiveMenu(menu: MenuItem | null) {
    activeMenuId.value = menu ? menu.id : null;
  }

  // Helper functions.
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

    // Proxies for menuManager methods.
    hasMenu: menuManager.hasMenu.bind(menuManager),
    getMenu: menuManager.getMenu.bind(menuManager),
    getMenuByPath: menuManager.getMenuByPath.bind(menuManager),
    getMenuChildren: menuManager.getMenuChildren.bind(menuManager),
    getMenuParent: menuManager.getMenuParent.bind(menuManager),
    getMenus: menuManager.getMenus.bind(menuManager),

    // Helper utilities.
    findFirstNavigableMenu,
  };
});
