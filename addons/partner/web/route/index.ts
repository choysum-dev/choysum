// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChoysumWebApp } from '@/core/web/application';
import { partnerRoutes } from './routes';

/**
 * Registers the partner route table under the application layout.
 */
export function setupRouter(app: ChoysumWebApp): void {
  const router = app.router;
  for (const route of partnerRoutes) {
    router.addRoute('AppLayout', route);
  }
}

export { partnerRoutes };
