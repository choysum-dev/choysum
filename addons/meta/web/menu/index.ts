// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { metaMenus } from './menus';
import type { ChoysumWebApp } from '@/core/web/application';

export function setupAppMenu(app: ChoysumWebApp) {
  const menu = app.menu;
  for (const item of metaMenus) {
    menu.addMenu(item);
  }
}
