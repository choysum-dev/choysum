// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { authMenus, authSettingsAppMenu } from './menus';
import type { ChoysumWebApp } from '@/core/web/application';

/**
 * Register auth menus on the application menu registry.
 */
export function setupAppMenu(app: ChoysumWebApp) {
  const menu = app.menu;
  for (const item of authMenus) {
    menu.addMenu(item);
  }
  // Parent Settings is registered by web; only appears when auth web is installed.
  menu.addMenu('web.menu.settings', authSettingsAppMenu);
}
