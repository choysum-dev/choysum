// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { baseRoutes } from './routes';
import type { ChoysumWebApp } from '@/core/web/application';

export function setupRouter(app: ChoysumWebApp): void {
  const router = app.router;
  for (const route of baseRoutes) {
    router.addRoute('AppLayout', route);
  }
}

export { baseRoutes };
