// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { TextSource } from '../../i18n';

/** Walk to the top-most ancestor; hop-capped against cyclic __parent chains. */
export function resolveAppRoot(activeMenu: any): any | null {
  let current = activeMenu;
  let hops = 0;
  while (current && hops < 64) {
    hops += 1;
    if (!current.__parent) return current;
    current = current.__parent;
  }
  return activeMenu || null;
}

/** Stable id for a menu node when building breadcrumb stack keys. */
export function getMenuId(menu: any): string {
  if (!menu) return '';
  if (menu.id) return String(menu.id);
  if (menu.path) return String(menu.path).replace(/\//g, '-');
  if (menu.name) return String(menu.name);
  return 'unknown-menu';
}

/** Prefer the menu's own path; otherwise keep the current route path. */
export function getMenuBasePath(currentPath: string, menu: any): string {
  if (menu?.path) return menu.path;
  return currentPath;
}

/**
 * Resolve a page crumb title from route meta / name / path heuristics.
 */
export function getPageTitle(
  route: any,
  detailsFallback: TextSource,
  pageFallback: TextSource,
): TextSource {
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
