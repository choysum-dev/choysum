// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChoysumWebApp } from '@/core/web/application';
import { sandboxMenus } from './menus';

export function setupAppMenu(app: ChoysumWebApp) {
  for (const item of sandboxMenus) {
    app.menu.addMenu(item);
  }
}

export * from './menus';
