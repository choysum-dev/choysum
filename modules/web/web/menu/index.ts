// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createMenuPlugin, type Menu } from '@/core/web/menu';
import { menus } from './menus';

export function createAppMenu() {
  const menuPlugin = createMenuPlugin();
  for (const menu of menus) {
    menuPlugin.addMenu(menu);
  }
  return menuPlugin;
}
