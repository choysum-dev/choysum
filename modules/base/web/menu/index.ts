// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { baseMenus, baseSettingsAppMenu } from './menus';
import type { ChoysumWebApp } from '@/core/web/application';

export function setupAppMenu(app: ChoysumWebApp) {
  const menu = app.menu;
  for (const item of baseMenus) {
    menu.addMenu(item);
  }
  // Parent Settings is registered by web (may load after base).
  menu.addMenu('web.menu.settings', baseSettingsAppMenu);
}

export * from './menus';
