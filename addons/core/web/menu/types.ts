// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Component } from 'vue';
import type { ObjectRecord } from '../../utils/types';

export interface MenuItem {
  id: string;
  title: string;
  order?: number;
  icon?: Component;
  path?: string;
  children?: MenuItem[];
  hidden?: boolean;
  disabled?: boolean;
  externalLink?: boolean;
  openMode?: 'current' | 'window' | 'parent' | 'top';
  meta?: ObjectRecord;
  readonly __parent?: MenuItem;
}

export interface Menu {
  addMenu(menu: MenuItem): Menu;
  addMenu(parentId: string | null, menu: MenuItem): Menu;
  removeMenu(id: string): boolean;
  replaceMenu(id: string, menu: MenuItem): boolean;
  hasMenu(id: string): boolean;
  getMenu(id: string): MenuItem | undefined;
  getMenus(): MenuItem[];
  clearMenus(): Menu;
  getMenuByPath(path: string): MenuItem | undefined;
  getMenuChildren(id: string): MenuItem[];
  getMenuParent(id: string): MenuItem | null;
}
