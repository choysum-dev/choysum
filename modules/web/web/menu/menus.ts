// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { House } from '@element-plus/icons-vue';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('web', { scope: 'web/menu/menus' });

/**
 * Menu configuration with the full menu structure and display metadata.
 * Kept separate from routing so it can focus on menu presentation.
 */
export const menus: MenuItem[] = [
  // Home menu granted to base.user by default after sign-in.
  defineMenu('web.menu.home', {
    title: _lt('Home'),
    icon: House,
    path: '/home',
    sequence: 1,
    defaultRoles: ['base.user'],
  }),
];
