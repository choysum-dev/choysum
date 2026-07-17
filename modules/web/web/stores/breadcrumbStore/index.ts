// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, readonly } from 'vue';
import { defineStore } from 'pinia';
import { nextTick } from 'vue';
import { useRouter } from 'vue-router';
import { isTextDescriptor, type TextDescriptor } from '@/core/service/i18n';
import type { TextSource } from '../../i18n';

export interface BreadcrumbItem {
  title: string;
  titleText?: TextDescriptor;
  path: string;
  clickable: boolean;
  timestamp: number;
}

/**
 * Breadcrumb store that tracks route-driven navigation history.
 */
export const useBreadcrumbStore = defineStore('breadcrumb', () => {
  // Core state.
  const breadcrumbStack = ref<BreadcrumbItem[]>([]);
  const currentMenuId = ref<string | null>(null);
  const routerGuardInstalled = ref(false);

  // Router guard initialization.

  /**
   * Installs the router guard that maintains breadcrumbs.
   */
  function installRouterGuard() {
    if (routerGuardInstalled.value) return;

    try {
      const router = useRouter();

      router.afterEach(async (to, from) => {
        try {
          await nextTick();
          await handleRouteChange(to, from);
        } catch (error) {
          console.error('Breadcrumb router guard failed:', error);
        }
      });

      routerGuardInstalled.value = true;

      // Process the current route immediately after installing the guard.
      nextTick(async () => {
        try {
          const currentRoute = router.currentRoute.value;
          if (currentRoute.path !== '/' && breadcrumbStack.value.length === 0) {
            await handleInitialRoute(currentRoute);
          }
        } catch (error) {
          console.error('Failed to process the initial route:', error);
        }
      });
    } catch (error) {
      console.error('Failed to install breadcrumb router guard:', error);
      setTimeout(() => {
        if (!routerGuardInstalled.value) {
          installRouterGuard();
        }
      }, 1000);
    }
  }

  /**
   * Handles the initial route when the page is opened directly.
   */
  async function handleInitialRoute(route: any) {
    try {
      await new Promise(resolve => setTimeout(resolve, 100));

      const { useMenuStore } = await import('../menuStore');
      const menuStore = useMenuStore();
      const activeMenu = menuStore.activeMenu;

      // Only create breadcrumbs when an active menu is available.
      if (activeMenu) {
        const menuId = getMenuId(activeMenu);
        const menuSource = activeMenu.titleText || activeMenu.title || '页面';
        const menuPath = getMenuBasePath(route.path, activeMenu);

        resetBreadcrumb(menuId, menuSource, menuPath);

        if (route.path !== menuPath) {
          const pageTitle = getPageTitle(route);
          if (pageTitle) {
            pushBreadcrumb(pageTitle, route.path);
          }
        }
      }
    } catch (error) {
      console.error('Failed to process the initial route:', error);
    }
  }

  /**
   * Handles a route change after navigation.
   */
  async function handleRouteChange(to: any, from: any) {
    try {
      const { useMenuStore } = await import('../menuStore');
      const menuStore = useMenuStore();
      const currentActiveMenu = menuStore.activeMenu;

      // Skip breadcrumb updates when no active menu is available.
      if (!currentActiveMenu) {
        return;
      }

      const activeMenuId = getMenuId(currentActiveMenu);
      const isMenuChanged = currentMenuId.value !== activeMenuId;
      const isFirstVisit = currentMenuId.value === null;

      if (isMenuChanged || isFirstVisit) {
        const menuSource = currentActiveMenu.titleText || currentActiveMenu.title || '页面';
        const menuPath = getMenuBasePath(to.path, currentActiveMenu);

        resetBreadcrumb(activeMenuId, menuSource, menuPath);

        if (to.path !== menuPath) {
          const pageTitle = getPageTitle(to);
          if (pageTitle) {
            pushBreadcrumb(pageTitle, to.path);
          }
        }
      } else {
        const pageTitle = getPageTitle(to);
        if (pageTitle) {
          pushBreadcrumb(pageTitle, to.path);
        }
      }
    } catch (error) {
      console.error('Failed to process the route change:', error);
    }
  }

  /**
   * Resolves a stable menu identifier.
   */
  function getMenuId(menu: any): string {
    if (!menu) return '';
    if (menu.id) return String(menu.id);
    if (menu.path) return menu.path.replace(/\//g, '-');
    if (menu.name) return String(menu.name);
    return 'unknown-menu';
  }

  /**
   * Resolves the base path for a menu entry.
   */
  function getMenuBasePath(currentPath: string, menu: any): string {
    if (menu?.path) return menu.path;
    return currentPath;
  }

  /**
   * Resolves the page title for a route.
   */
  function getPageTitle(route: any): TextSource {
    if (route.meta?.pageTitleText) {
      return route.meta.pageTitleText;
    }
    // 1. Prefer meta.pageTitle, which defineRoute.title maps to by default.
    if (route.meta?.pageTitle) {
      return typeof route.meta.pageTitle === 'function' ? route.meta.pageTitle(route) : String(route.meta.pageTitle);
    }

    // 2. Fall back to the legacy meta.title field.
    if (route.meta?.title) {
      return String(route.meta.title);
    }

    // 3. Fall back to the route name.
    if (route.name) {
      return String(route.name);
    }

    // 4. Infer detail pages from id-like path segments.
    const pathSegments = route.path.split('/').filter(Boolean);
    if (pathSegments.length > 0) {
      const lastSegment = pathSegments[pathSegments.length - 1];

      if (/^[a-z0-9]{16,}$/.test(lastSegment) || /^\d+$/.test(lastSegment)) {
        return '详情';
      }
    }

    return '页面';
  }

  // Core breadcrumb operations.

  /**
   * Resets the breadcrumb trail to the active menu root.
   */
  function normalizeTitle(source: TextSource): Pick<BreadcrumbItem, 'title' | 'titleText'> {
    return isTextDescriptor(source)
      ? { title: source.src, titleText: { ...source } }
      : { title: String(source || ''), titleText: undefined };
  }

  function resetBreadcrumb(menuId: string, source: TextSource, menuPath: string) {
    currentMenuId.value = menuId;
    breadcrumbStack.value = [
      {
        ...normalizeTitle(source),
        path: menuPath,
        clickable: false,
        timestamp: Date.now(),
      },
    ];
  }

  /**
   * Pushes a new breadcrumb item or truncates to an existing path.
   */
  function pushBreadcrumb(title: TextSource, path: string) {
    const existingIndex = breadcrumbStack.value.findIndex(item => item.path === path);

    if (existingIndex > -1) {
      // Truncate to the existing item and refresh its title.
      breadcrumbStack.value = breadcrumbStack.value.slice(0, existingIndex + 1);
      Object.assign(breadcrumbStack.value[existingIndex], normalizeTitle(title));
      breadcrumbStack.value[existingIndex].clickable = false;
    } else {
      // Earlier items become clickable once a deeper item is pushed.
      breadcrumbStack.value.forEach(item => {
        item.clickable = true;
      });

      breadcrumbStack.value.push({
        ...normalizeTitle(title),
        path,
        clickable: false,
        timestamp: Date.now(),
      });
    }

    // Limit the breadcrumb trail length.
    if (breadcrumbStack.value.length > 10) {
      breadcrumbStack.value = breadcrumbStack.value.slice(-10);
    }
  }

  /**
   * Truncates the breadcrumb trail and returns the target path.
   */
  function navigateToBreadcrumb(index: number): string | null {
    if (index < 0 || index >= breadcrumbStack.value.length) {
      return null;
    }

    breadcrumbStack.value = breadcrumbStack.value.slice(0, index + 1);
    const targetItem = breadcrumbStack.value[index];
    targetItem.clickable = false;

    return targetItem.path;
  }

  /**
   * Updates the title of the current breadcrumb item.
   */
  function updateCurrentTitle(title: TextSource) {
    if (breadcrumbStack.value.length > 0) {
      const currentItem = breadcrumbStack.value[breadcrumbStack.value.length - 1];
      Object.assign(currentItem, normalizeTitle(title));
    }
  }

  /**
   * Clears the breadcrumb trail.
   */
  function clearBreadcrumb() {
    breadcrumbStack.value = [];
    currentMenuId.value = null;
  }

  // Install the router guard as soon as the store is created.
  installRouterGuard();

  return {
    breadcrumbStack,
    currentMenuId: readonly(currentMenuId),
    routerGuardInstalled: readonly(routerGuardInstalled),

    // Public methods.
    resetBreadcrumb,
    pushBreadcrumb,
    navigateToBreadcrumb,
    updateCurrentTitle,
    clearBreadcrumb,
  };
});
