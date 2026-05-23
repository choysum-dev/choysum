// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { Setting } from '@element-plus/icons-vue';
import { defineMenu } from '@/core/web/resource';

export const metaMenus: MenuItem[] = [
  defineMenu('meta.menu.root', {
    title: '模块管理',
    icon: Setting,
    sequence: 60,
    children: [
      defineMenu('meta.menu.module_board', {
        title: '模块看板',
        path: '/meta/modules',
        sequence: 10,
        requires: [{ model: 'meta.IrModuleIndex' }, { model: 'meta.IrModule' }],
      }),
      defineMenu('meta.menu.module_list', {
        title: '模块列表',
        path: '/meta/modules/list',
        sequence: 20,
        requires: [{ model: 'meta.IrModuleIndex' }, { model: 'meta.IrModule' }],
      }),
      defineMenu('meta.menu.module_history', {
        title: '操作历史',
        path: '/meta/modules/history',
        sequence: 30,
        requires: [{ model: 'meta.ModuleManagementLog' }, { model: 'meta.IrModule' }, { model: 'meta.IrModuleIndex' }],
      }),
    ],
  }),
];
