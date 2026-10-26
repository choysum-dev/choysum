// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChoysumWebApp } from '@/core/web/application';
import { sandboxRoutes } from './routes';

/** Registers sandbox routes under the application layout. */
export function setupRouter(app: ChoysumWebApp): void {
  const router = app.router;
  for (const route of sandboxRoutes) {
    router.addRoute('AppLayout', route);
  }
}

export { sandboxRoutes };
