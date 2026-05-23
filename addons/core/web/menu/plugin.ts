// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { App } from 'vue';
import { MenuManager } from './manager';
import type { Menu, MenuItem } from './types';

export const MenuSymbol = Symbol('ChoysumMenu');

export interface MenuPlugin {
  install(app: App): void;
  readonly manager: Menu;
  addMenu(parentIdOrMenu: string | null | MenuItem, menu?: MenuItem): Menu;
  removeMenu(id: string): boolean;
  replaceMenu(id: string, menu: MenuItem): boolean;
  hasMenu(id: string): boolean;
  getMenu(id: string): MenuItem | undefined;
  getMenuByPath(path: string): MenuItem | undefined;
  getMenuChildren(id: string): MenuItem[];
  getMenuParent(id: string): MenuItem | null;
  getMenus(): MenuItem[];
  clearMenus(): Menu;
  loadMenusFromConfig(menus: MenuItem[]): Menu;
  exportMenuConfig(): MenuItem[];
}

export function createMenuPlugin(): MenuPlugin {
  const menuManager = new MenuManager();

  return {
    install(app: App) {
      app.config.globalProperties.$menu = menuManager;
      app.provide(MenuSymbol, menuManager);
    },
    manager: menuManager,
    addMenu: menuManager.addMenu.bind(menuManager),
    removeMenu: menuManager.removeMenu.bind(menuManager),
    replaceMenu: menuManager.replaceMenu.bind(menuManager),
    hasMenu: menuManager.hasMenu.bind(menuManager),
    getMenu: menuManager.getMenu.bind(menuManager),
    getMenuByPath: menuManager.getMenuByPath.bind(menuManager),
    getMenuChildren: menuManager.getMenuChildren.bind(menuManager),
    getMenuParent: menuManager.getMenuParent.bind(menuManager),
    getMenus: menuManager.getMenus.bind(menuManager),
    clearMenus: menuManager.clearMenus.bind(menuManager),
    loadMenusFromConfig: menuManager.loadMenusFromConfig.bind(menuManager),
    exportMenuConfig: menuManager.exportMenuConfig.bind(menuManager),
  };
}
