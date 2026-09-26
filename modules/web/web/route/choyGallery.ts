// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChoysumWebApp } from '@/core/web/application';
import { choyUiRoutes } from './choyGalleryRoutes';

/**
 * Registers the Choy UI gallery under the authenticated layout without a menu entry.
 */
export function setupRouter(app: ChoysumWebApp): void {
  const router = app.router;
  if (!router.hasRoute('AppLayout')) {
    console.warn('[web] AppLayout route is not registered; Choy gallery routes were skipped');
    return;
  }
  for (const route of choyUiRoutes) {
    const name = route.name != null ? String(route.name) : '';
    if (name && router.hasRoute(name)) {
      continue;
    }
    router.addRoute('AppLayout', route);
  }
}

/**
 * Public entry used when the kit host boots (product web and/or choy_ui shell).
 */
export function registerChoyGalleryRoute(app: ChoysumWebApp): void {
  setupRouter(app);
}

export { choyUiRoutes };
