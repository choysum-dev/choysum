// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, readonly, nextTick } from 'vue';
import { defineStore } from 'pinia';
import { useRouter } from 'vue-router';
import { isTermReference, type TermReference } from '@/core/service/i18n';
import { createTranslate, type TextSource } from '../../i18n';
import {
  getIntermediateAncestors,
  getMenuBasePath,
  getMenuId,
  getPageTitle as resolvePageTitle,
  isMenuNavigatedCrumb,
  resolveAppRoot,
} from './breadcrumb_helpers';

const { _lt } = createTranslate('web', { scope: 'web/stores/breadcrumbStore' });
const pageFallback = _lt('Page');
const detailsFallback = _lt('Details');

/** Max trail length including the menu-root prefix (app + folders + leaf). */
const MAX_STACK = 10;

export type BreadcrumbKind = 'app' | 'folder' | 'menu' | 'page';

export interface BreadcrumbItem {
  title: string;
  titleText?: TermReference;
  path: string;
  clickable: boolean;
  timestamp: number;
  /** app = root; folder = pathless intermediate; menu = leaf; page = deeper segment. */
  kind: BreadcrumbKind;
  /** Menu id for app / folder / leaf segments (used by menu navigation). */
  menuId?: string;
}

/**
 * Breadcrumb store that tracks route-driven navigation history.
 * Stack shape: [app?, ...folders, leaf, ...pages].
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
        await nextTick();
        await handleRouteChange(to, from);
      });

      routerGuardInstalled.value = true;

      nextTick(async () => {
        const currentRoute = router.currentRoute.value;
        if (currentRoute.path !== '/' && breadcrumbStack.value.length === 0) {
          await handleInitialRoute(currentRoute);
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
    } catch (err) {
      console.warn('[breadcrumbStore] handleInitialRoute failed', err);
      clearBreadcrumb();
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
    } catch (err) {
      console.warn('[breadcrumbStore] handleRouteChange failed', err);
      clearBreadcrumb();
    }
  }

  /**
   * Rebuilds or extends the stack for the active leaf (and ancestor folders) and current route.
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

  function getPageTitle(route: any): TextSource {
    return resolvePageTitle(route, detailsFallback, pageFallback);
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

  /** Count of non-page crumbs (app + intermediate folders + leaf). */
  function rootPrefixLength(): number {
    let i = 0;
    while (i < breadcrumbStack.value.length && breadcrumbStack.value[i]?.kind !== 'page') {
      i += 1;
    }
    return i;
  }

  function trimStackPreservingRoot() {
    if (breadcrumbStack.value.length <= MAX_STACK) return;
    const stack = breadcrumbStack.value;
    const prefix = rootPrefixLength();
    // Deep menu roots can fill MAX_STACK; keep app + leaf + newest pages, drop folders.
    if (prefix >= MAX_STACK) {
      const root = stack.slice(0, prefix);
      const app = root.find((item) => item.kind === 'app');
      const leaf = [...root].reverse().find((item) => item.kind === 'menu');
      const kept: BreadcrumbItem[] = [];
      if (app) kept.push(app);
      if (leaf && leaf !== app) kept.push(leaf);
      const pages = stack.slice(prefix);
      const room = Math.max(0, MAX_STACK - kept.length);
      kept.push(...pages.slice(-room));
      breadcrumbStack.value = kept.slice(0, MAX_STACK);
      applyClickableFlags();
      return;
    }
    const keepTail = MAX_STACK - prefix;
    const head = stack.slice(0, prefix);
    const tail = stack.slice(-keepTail);
    breadcrumbStack.value = [...head, ...tail];
    applyClickableFlags();
  }

  function truncateToMenuRoot() {
    const prefix = rootPrefixLength();
    if (prefix <= 0) return;
    breadcrumbStack.value = breadcrumbStack.value.slice(0, prefix);
    applyClickableFlags();
  }

  function buildMenuRootItems(
    app: any | null,
    leaf: any,
    leafPath: string,
    leafSource: TextSource,
    now: number,
  ): BreadcrumbItem[] {
    const leafId = getMenuId(leaf);
    const leafItem: BreadcrumbItem = {
      ...normalizeTitle(leafSource),
      path: leafPath,
      clickable: false,
      timestamp: now,
      kind: 'menu',
      menuId: leafId,
    };

    const appId = app ? getMenuId(app) : null;
    if (!app || !appId || appId === leafId) {
      return [leafItem];
    }

    const appSource = app.titleText || app.title || pageFallback;
    const items: BreadcrumbItem[] = [
      {
        ...normalizeTitle(appSource),
        path: '',
        clickable: false,
        timestamp: now,
        kind: 'app',
        menuId: appId,
      },
    ];

    for (const folder of getIntermediateAncestors(leaf, app)) {
      const folderId = getMenuId(folder);
      if (!folderId || folderId === appId || folderId === leafId) continue;
      const folderSource = folder.titleText || folder.title || pageFallback;
      items.push({
        ...normalizeTitle(folderSource),
        path: folder.path || '',
        clickable: false,
        timestamp: now,
        kind: 'folder',
        menuId: folderId,
      });
    }

    items.push(leafItem);
    return items;
  }

  /**
   * Full reset: [app?, ...folders, leaf].
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
    breadcrumbStack.value = buildMenuRootItems(app, leaf, leafPath, leafSource, Date.now());
    applyClickableFlags();
  }

  /**
   * Same L1 app, different leaf: keep the app segment when possible, rebuild folders + leaf.
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
    const items = buildMenuRootItems(app, leaf, leafPath, leafSource, now);
    const prevApp = breadcrumbStack.value[0];
    if (prevApp?.kind === 'app' && prevApp.menuId === appId && items[0]?.kind === 'app') {
      items[0] = {
        ...prevApp,
        ...normalizeTitle(app.titleText || app.title || pageFallback),
        menuId: appId,
        kind: 'app',
        path: '',
      };
    }
    breadcrumbStack.value = items;
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
    // Skip pathless folders/app crumbs; folders with a path must not steal page truncate.
    const existingIndex = breadcrumbStack.value.findIndex(
      (item) => item.path === path && item.path !== '' && item.kind !== 'folder',
    );

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

    const targetItem = breadcrumbStack.value[index];
    // App / pathless folder crumbs navigate via menuApi; do not truncate here.
    if (isMenuNavigatedCrumb(targetItem)) {
      return null;
    }

    breadcrumbStack.value = breadcrumbStack.value.slice(0, index + 1);
    applyClickableFlags();
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
