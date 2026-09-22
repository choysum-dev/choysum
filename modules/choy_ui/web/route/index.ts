// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChoysumWebApp } from '@/core/web/application';
import { choyUiRoutes } from './routes';

/**
 * Registers the Choy UI gallery under the authenticated layout without a menu entry.
 */
export function setupRouter(app: ChoysumWebApp): void {
  const router = app.router;
  for (const route of choyUiRoutes) {
    router.addRoute('AppLayout', route);
  }
}

/**
 * Public entry used by the web module bootstrap when choy_ui is installed.
 */
export function registerChoyGalleryRoute(app: ChoysumWebApp): void {
  setupRouter(app);
}

export { choyUiRoutes };
