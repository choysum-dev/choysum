// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { App } from 'vue';
import { MenuManager } from './manager';
import type { Menu, MenuItem } from './types';

export const MenuSymbol = Symbol('ChoysumMenu');

let installedMenu: Menu | null = null;

/**
 * Returns the Menu from the last `createMenuPlugin().install()`.
 * Used when Pinia/router callbacks have no inject context.
 */
export function getInstalledMenu(): Menu | null {
  return installedMenu;
}

/**
 * Drops the last installed manager. App unmount also does this; tests that share
 * one JS realm call it between cases so a leftover tree cannot leak.
 */
export function resetInstalledMenu(): void {
  installedMenu = null;
}

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
      const prevInstalled = installedMenu;
      installedMenu = menuManager;
      app.config.globalProperties.$menu = menuManager;
      app.provide(MenuSymbol, menuManager);
      const prevUnmount = typeof app.unmount === 'function' ? app.unmount.bind(app) : undefined;
      if (prevUnmount) {
        app.unmount = () => {
          if (installedMenu === menuManager) installedMenu = prevInstalled;
          prevUnmount();
        };
      }
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
