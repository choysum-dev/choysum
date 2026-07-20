// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { Setting } from '@element-plus/icons-vue';
import { defineMenu } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('meta', { scope: 'web/menu/menus' });

export const metaMenus: MenuItem[] = [
  defineMenu('meta.menu.root', {
    title: _lt('Module Management'),
    icon: Setting,
    sequence: 60,
    children: [
      defineMenu('meta.menu.module_board', {
        title: _lt('Module Board'),
        path: '/meta/modules',
        sequence: 10,
        requires: [{ model: 'meta.IrModuleIndex' }, { model: 'meta.IrModule' }],
      }),
      defineMenu('meta.menu.module_list', {
        title: _lt('Module List'),
        path: '/meta/modules/list',
        sequence: 20,
        requires: [{ model: 'meta.IrModuleIndex' }, { model: 'meta.IrModule' }],
      }),
      defineMenu('meta.menu.module_history', {
        title: _lt('Operation History'),
        path: '/meta/modules/history',
        sequence: 30,
        requires: [{ model: 'meta.ModuleManagementLog' }, { model: 'meta.IrModule' }, { model: 'meta.IrModuleIndex' }],
      }),
    ],
  }),
];
