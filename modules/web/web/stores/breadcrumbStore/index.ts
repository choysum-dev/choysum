// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, readonly, nextTick } from 'vue';
import { defineStore } from 'pinia';
import { useRouter } from 'vue-router';
import { isTermReference, type TermReference } from '@/core/service/i18n';
import { createTranslate, type TextSource } from '../../i18n';

const { _lt } = createTranslate('web', { scope: 'web/stores/breadcrumbStore' });
const pageFallback = _lt('Page');
const detailsFallback = _lt('Details');

/** Max trail length including the L1+L2 root prefix. */
const MAX_STACK = 10;

export type BreadcrumbKind = 'app' | 'menu' | 'page';

export interface BreadcrumbItem {
  title: string;
  titleText?: TermReference;
  path: string;
  clickable: boolean;
  timestamp: number;
  /** app = L1; menu = L2 leaf; page = deeper route segment. */
  kind: BreadcrumbKind;
  /** Menu id for app / leaf segments (used by app-home navigation). */
  menuId?: string;
}

/**
 * Breadcrumb store that tracks route-driven navigation history.
 * Stack shape: [L1 app?, L2 leaf, ...pages].
 */
export const useBreadcrumbStore = defineStore('breadcrumb', () => {
  const breadcrumbStack = ref<BreadcrumbItem[]>([]);
  const currentMenuId = ref<string | null>(null);
  const currentAppId = ref<string | null>(null);
  const routerGuardInstalled = ref(false);

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
   * Waits briefly for menuStore.activeMenu (route sync may lag cold start).
   */
  async function waitForActiveMenu(maxMs = 500, stepMs = 50): Promise<any | null> {
    const { useMenuStore } = await import('../menuStore');
    const menuStore = useMenuStore();
    const deadline = Date.now() + maxMs;
    while (Date.now() <= deadline) {
      if (menuStore.activeMenu) return menuStore.activeMenu;
      await new Promise((resolve) => setTimeout(resolve, stepMs));
    }
    return menuStore.activeMenu;
  }

  /**
   * Handles the initial route when the page is opened directly.
   */
  async function handleInitialRoute(route: any) {
    try {
      const { useMenuStore } = await import('../menuStore');
      const menuStore = useMenuStore();
      const activeMenu = (await waitForActiveMenu()) || menuStore.activeMenu;
      if (!activeMenu) {
        clearBreadcrumb();
        return;
      }
      syncStackToActiveMenu(activeMenu, route, menuStore.activeApp);
    } catch (error) {
      console.error('Failed to process the initial route:', error);
    }
  }

  /**
   * Handles a route change after navigation.
   */
  async function handleRouteChange(to: any, _from: any) {
    try {
      const { useMenuStore } = await import('../menuStore');
      const menuStore = useMenuStore();
      const currentActiveMenu = menuStore.activeMenu;

      if (!currentActiveMenu) {
        clearBreadcrumb();
        return;
      }

      syncStackToActiveMenu(currentActiveMenu, to, menuStore.activeApp);
    } catch (error) {
      console.error('Failed to process the route change:', error);
    }
  }

  /**
   * Rebuilds or extends the stack for the active L2 leaf and current route.
   */
  function syncStackToActiveMenu(activeMenu: any, route: any, activeApp: any | null) {
    const leafId = getMenuId(activeMenu);
    const app = activeApp || resolveAppRoot(activeMenu);
    const appId = app ? getMenuId(app) : null;
    const leafPath = getMenuBasePath(route.path, activeMenu);
    const leafSource = activeMenu.titleText || activeMenu.title || pageFallback;
    const sameLeaf = currentMenuId.value === leafId && breadcrumbStack.value.length > 0;
    const sameApp = !!appId && currentAppId.value === appId;

    if (sameLeaf) {
      if (route.path !== leafPath) {
        const pageTitle = getPageTitle(route);
        if (pageTitle) pushBreadcrumb(pageTitle, route.path);
      } else {
        truncateToMenuRoot();
      }
      return;
    }

    if (sameApp && app && appId !== leafId) {
      replaceLeafUnderApp(app, activeMenu, leafPath, leafSource);
    } else {
      resetBreadcrumbToAppAndLeaf(app, activeMenu, leafPath, leafSource);
    }

    if (route.path !== leafPath) {
      const pageTitle = getPageTitle(route);
      if (pageTitle) pushBreadcrumb(pageTitle, route.path);
    }
  }

  function resolveAppRoot(activeMenu: any): any | null {
    let current = activeMenu;
    let hops = 0;
    while (current && hops < 64) {
      hops += 1;
      if (!current.__parent) return current;
      current = current.__parent;
    }
    return activeMenu || null;
  }

  function getMenuId(menu: any): string {
    if (!menu) return '';
    if (menu.id) return String(menu.id);
    if (menu.path) return String(menu.path).replace(/\//g, '-');
    if (menu.name) return String(menu.name);
    return 'unknown-menu';
  }

  function getMenuBasePath(currentPath: string, menu: any): string {
    if (menu?.path) return menu.path;
    return currentPath;
  }

  function getPageTitle(route: any): TextSource {
    if (route.meta?.pageTitleText) {
      return route.meta.pageTitleText;
    }
    if (route.meta?.pageTitle) {
      return typeof route.meta.pageTitle === 'function'
        ? route.meta.pageTitle(route)
        : String(route.meta.pageTitle);
    }
    if (route.meta?.title) {
      return String(route.meta.title);
    }
    if (route.name) {
      return String(route.name);
    }
    const pathSegments = String(route.path || '')
      .split('/')
      .filter(Boolean);
    if (pathSegments.length > 0) {
      const lastSegment = pathSegments[pathSegments.length - 1];
      if (/^[a-z0-9]{16,}$/.test(lastSegment) || /^\d+$/.test(lastSegment)) {
        return detailsFallback;
      }
    }
    return pageFallback;
  }

  function normalizeTitle(source: TextSource): Pick<BreadcrumbItem, 'title' | 'titleText'> {
    return isTermReference(source)
      ? { title: source.src, titleText: { ...source } }
      : { title: String(source || ''), titleText: undefined };
  }

  function applyClickableFlags() {
    const n = breadcrumbStack.value.length;
    breadcrumbStack.value.forEach((item, i) => {
      item.clickable = i < n - 1;
    });
  }

  function rootPrefixLength(): number {
    if (!breadcrumbStack.value.length) return 0;
    if (breadcrumbStack.value[0]?.kind === 'app') {
      return Math.min(2, breadcrumbStack.value.length);
    }
    return 1;
  }

  function trimStackPreservingRoot() {
    if (breadcrumbStack.value.length <= MAX_STACK) return;
    const prefix = rootPrefixLength();
    const keepTail = MAX_STACK - prefix;
    if (keepTail <= 0) {
      breadcrumbStack.value = breadcrumbStack.value.slice(0, MAX_STACK);
      applyClickableFlags();
      return;
    }
    const head = breadcrumbStack.value.slice(0, prefix);
    const tail = breadcrumbStack.value.slice(-keepTail);
    breadcrumbStack.value = [...head, ...tail];
    applyClickableFlags();
  }

  function truncateToMenuRoot() {
    const prefix = rootPrefixLength();
    if (prefix <= 0) return;
    breadcrumbStack.value = breadcrumbStack.value.slice(0, prefix);
    applyClickableFlags();
  }

  /**
   * Full reset: [app?, leaf].
   */
  function resetBreadcrumbToAppAndLeaf(
    app: any | null,
    leaf: any,
    leafPath: string,
    leafSource: TextSource,
  ) {
    const leafId = getMenuId(leaf);
    const appId = app ? getMenuId(app) : null;
    currentMenuId.value = leafId;
    currentAppId.value = appId;

    const now = Date.now();
    const leafItem: BreadcrumbItem = {
      ...normalizeTitle(leafSource),
      path: leafPath,
      clickable: false,
      timestamp: now,
      kind: 'menu',
      menuId: leafId,
    };

    if (app && appId && appId !== leafId) {
      const appSource = app.titleText || app.title || pageFallback;
      breadcrumbStack.value = [
        {
          ...normalizeTitle(appSource),
          path: '',
          clickable: false,
          timestamp: now,
          kind: 'app',
          menuId: appId,
        },
        leafItem,
      ];
    } else {
      breadcrumbStack.value = [leafItem];
    }
    applyClickableFlags();
  }

  /**
   * Same L1 app, different L2 leaf: keep the app segment, replace the rest.
   */
  function replaceLeafUnderApp(
    app: any,
    leaf: any,
    leafPath: string,
    leafSource: TextSource,
  ) {
    const leafId = getMenuId(leaf);
    const appId = getMenuId(app);
    currentMenuId.value = leafId;
    currentAppId.value = appId;

    const now = Date.now();
    const appSource = app.titleText || app.title || pageFallback;
    const prevApp = breadcrumbStack.value[0];
    const appItem: BreadcrumbItem =
      prevApp?.kind === 'app' && prevApp.menuId === appId
        ? {
            ...prevApp,
            ...normalizeTitle(appSource),
            menuId: appId,
            kind: 'app',
            path: '',
          }
        : {
            ...normalizeTitle(appSource),
            path: '',
            clickable: false,
            timestamp: now,
            kind: 'app',
            menuId: appId,
          };

    breadcrumbStack.value = [
      appItem,
      {
        ...normalizeTitle(leafSource),
        path: leafPath,
        clickable: false,
        timestamp: now,
        kind: 'menu',
        menuId: leafId,
      },
    ];
    applyClickableFlags();
  }

  /**
   * Public reset used by tests / callers: treats source as the L2 leaf only.
   */
  function resetBreadcrumb(menuId: string, source: TextSource, menuPath: string) {
    currentMenuId.value = menuId;
    currentAppId.value = null;
    breadcrumbStack.value = [
      {
        ...normalizeTitle(source),
        path: menuPath,
        clickable: false,
        timestamp: Date.now(),
        kind: 'menu',
        menuId,
      },
    ];
    applyClickableFlags();
  }

  /**
   * Pushes a new page segment or truncates to an existing path.
   */
  function pushBreadcrumb(title: TextSource, path: string) {
    const existingIndex = breadcrumbStack.value.findIndex((item) => item.path === path && item.path !== '');

    if (existingIndex > -1) {
      breadcrumbStack.value = breadcrumbStack.value.slice(0, existingIndex + 1);
      Object.assign(breadcrumbStack.value[existingIndex], normalizeTitle(title));
    } else {
      breadcrumbStack.value.push({
        ...normalizeTitle(title),
        path,
        clickable: false,
        timestamp: Date.now(),
        kind: 'page',
      });
    }

    applyClickableFlags();
    trimStackPreservingRoot();
  }

  /**
   * Truncates the trail to index and returns a navigable path (null for app crumbs).
   */
  function navigateToBreadcrumb(index: number): string | null {
    if (index < 0 || index >= breadcrumbStack.value.length) {
      return null;
    }

    breadcrumbStack.value = breadcrumbStack.value.slice(0, index + 1);
    applyClickableFlags();
    const targetItem = breadcrumbStack.value[index];
    if (targetItem.kind === 'app') return null;
    return targetItem.path || null;
  }

  function updateCurrentTitle(title: TextSource) {
    if (breadcrumbStack.value.length > 0) {
      const currentItem = breadcrumbStack.value[breadcrumbStack.value.length - 1];
      Object.assign(currentItem, normalizeTitle(title));
    }
  }

  function clearBreadcrumb() {
    breadcrumbStack.value = [];
    currentMenuId.value = null;
    currentAppId.value = null;
  }

  installRouterGuard();

  return {
    breadcrumbStack,
    currentMenuId: readonly(currentMenuId),
    currentAppId: readonly(currentAppId),
    routerGuardInstalled: readonly(routerGuardInstalled),

    resetBreadcrumb,
    pushBreadcrumb,
    navigateToBreadcrumb,
    updateCurrentTitle,
    clearBreadcrumb,
  };
});
