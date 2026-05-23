// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, markRaw } from 'vue';
import type { MenuItem, Menu } from './types';

export class MenuManager implements Menu {
  private menuIdMap = new Map<string, MenuItem>();
  private menuPathMap = new Map<string, MenuItem>();
  private rootMenus = ref<MenuItem[]>([]);

  constructor() {
    this.rootMenus.value = [];
  }

  addMenu(parentIdOrMenu: string | null | MenuItem, menu?: MenuItem): Menu {
    if (typeof parentIdOrMenu === 'object' && parentIdOrMenu !== null && menu === undefined) {
      return this.addMenuInternal(null, parentIdOrMenu);
    }

    if ((typeof parentIdOrMenu === 'string' || parentIdOrMenu === null) && menu !== undefined) {
      return this.addMenuInternal(parentIdOrMenu, menu);
    }

    throw new Error('Invalid addMenu arguments');
  }

  private addMenuInternal(parentId: string | null, menu: MenuItem): Menu {
    if (this.menuIdMap.has(menu.id)) {
      console.warn(`Menu ID "${menu.id}" already exists and cannot be added`);
      return this;
    }

    const parentMenu = parentId ? this.menuIdMap.get(parentId) : null;
    const menuItem: MenuItem = this.processMenuItemForReactivity(menu, parentMenu);

    this.menuIdMap.set(menu.id, menuItem);

    if (menuItem.path) {
      this.menuPathMap.set(menuItem.path, menuItem);
    }

    if (parentId === null) {
      this.rootMenus.value.push(menuItem);
      this.sortMenus(this.rootMenus.value);
    } else {
      const parent = this.menuIdMap.get(parentId);
      if (parent) {
        if (!parent.children) {
          parent.children = [];
        }
        parent.children.push(menuItem);
        this.sortMenus(parent.children);
      } else {
        console.warn(`Parent menu "${parentId}" does not exist`);
        return this;
      }
    }

    return this;
  }

  private processMenuItemForReactivity(menu: MenuItem, parent: MenuItem | null = null): MenuItem {
    const processedMenu: MenuItem = {
      ...menu,
      icon: menu.icon ? markRaw(menu.icon) : undefined,
      __parent: parent || undefined,
      children: undefined,
    };

    if (menu.children && menu.children.length > 0) {
      processedMenu.children = menu.children.map(child => this.processMenuItemForReactivity(child, processedMenu));
      this.addChildrenToMaps(processedMenu.children);
    }

    return processedMenu;
  }

  private addChildrenToMaps(children: MenuItem[]): void {
    for (const child of children) {
      this.menuIdMap.set(child.id, child);

      if (child.path) {
        this.menuPathMap.set(child.path, child);
      }

      if (child.children) {
        this.addChildrenToMaps(child.children);
      }
    }
  }

  removeMenu(id: string): boolean {
    const menu = this.menuIdMap.get(id);
    if (!menu) {
      return false;
    }

    if (menu.children) {
      this.removeChildrenFromMaps(menu.children);
    }

    if (menu.path) {
      this.menuPathMap.delete(menu.path);
    }

    this.removeFromParent(id);
    this.menuIdMap.delete(id);

    return true;
  }

  private removeChildrenFromMaps(children: MenuItem[]): void {
    for (const child of children) {
      this.menuIdMap.delete(child.id);

      if (child.path) {
        this.menuPathMap.delete(child.path);
      }

      if (child.children) {
        this.removeChildrenFromMaps(child.children);
      }
    }
  }

  replaceMenu(id: string, menu: MenuItem): boolean {
    if (!this.menuIdMap.has(id)) {
      return false;
    }

    const oldMenu = this.menuIdMap.get(id);
    const parent = oldMenu?.__parent;

    if (oldMenu?.children) {
      this.removeChildrenFromMaps(oldMenu.children);
    }

    const newMenu: MenuItem = {
      ...this.processMenuItemForReactivity(menu, parent),
      id,
    };

    this.menuIdMap.set(id, newMenu);

    if (oldMenu?.path) {
      this.menuPathMap.delete(oldMenu.path);
    }
    if (newMenu.path) {
      this.menuPathMap.set(newMenu.path, newMenu);
    }

    this.updateMenuReferences(id, newMenu);

    return true;
  }

  hasMenu(id: string): boolean {
    return this.menuIdMap.has(id);
  }

  getMenu(id: string): MenuItem | undefined {
    return this.menuIdMap.get(id);
  }

  getMenuByPath(path: string): MenuItem | undefined {
    return this.menuPathMap.get(path);
  }

  getMenuChildren(id: string): MenuItem[] {
    const menu = this.menuIdMap.get(id);
    return menu?.children ? [...menu.children] : [];
  }

  getMenuParent(id: string): MenuItem | null {
    const menu = this.menuIdMap.get(id);
    return menu?.__parent || null;
  }

  getMenus(): MenuItem[] {
    return [...this.rootMenus.value];
  }

  clearMenus(): Menu {
    this.menuIdMap.clear();
    this.menuPathMap.clear();
    this.rootMenus.value = [];
    return this;
  }

  loadMenusFromConfig(menus: MenuItem[]): Menu {
    this.clearMenus();

    const loadMenu = (menu: MenuItem, parentId: string | null = null) => {
      this.addMenuInternal(parentId, {
        ...menu,
        children: [],
      });

      if (menu.children) {
        for (const child of menu.children) {
          loadMenu(child, menu.id);
        }
      }
    };

    for (const menu of menus) {
      loadMenu(menu);
    }

    return this;
  }

  exportMenuConfig(): MenuItem[] {
    const cleanMenu = (menu: MenuItem): MenuItem => {
      const { __parent, ...cleanedMenu } = menu;
      return {
        ...cleanedMenu,
        children: menu.children ? menu.children.map(cleanMenu) : undefined,
      };
    };

    return this.rootMenus.value.map(cleanMenu);
  }

  private sortMenus(menus: MenuItem[]): void {
    menus.sort((a, b) => (a.order || 0) - (b.order || 0));
  }

  private removeFromParent(id: string): void {
    const rootIndex = this.rootMenus.value.findIndex(menu => menu.id === id);
    if (rootIndex !== -1) {
      this.rootMenus.value.splice(rootIndex, 1);
      return;
    }

    for (const menu of this.menuIdMap.values()) {
      if (menu.children) {
        const childIndex = menu.children.findIndex(child => child.id === id);
        if (childIndex !== -1) {
          menu.children.splice(childIndex, 1);
          return;
        }
      }
    }
  }

  private updateMenuReferences(id: string, newMenu: MenuItem): void {
    const rootIndex = this.rootMenus.value.findIndex(menu => menu.id === id);
    if (rootIndex !== -1) {
      this.rootMenus.value[rootIndex] = newMenu;
      return;
    }

    for (const menu of this.menuIdMap.values()) {
      if (menu.children) {
        const childIndex = menu.children.findIndex(child => child.id === id);
        if (childIndex !== -1) {
          menu.children[childIndex] = newMenu;
          return;
        }
      }
    }
  }
}
