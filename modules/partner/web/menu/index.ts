// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { partnerMenus, partnerSettingsAppMenu } from './menus';
import type { ChoysumWebApp } from '@/core/web/application';

/**
 * Registers the partner menu tree on the application menu registry.
 */
export function setupAppMenu(app: ChoysumWebApp) {
  const menu = app.menu;
  for (const item of partnerMenus) {
    menu.addMenu(item);
  }
  menu.addMenu('web.menu.settings', partnerSettingsAppMenu);
}

/**
 * Partner menu exports.
 */
export * from './menus';
