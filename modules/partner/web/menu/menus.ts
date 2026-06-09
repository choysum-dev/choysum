// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';
import { UserFilled } from '@element-plus/icons-vue';
import { defineMenu } from '@/core/web/resource';

/**
 * Menu tree registered by the partner module.
 */
export const partnerMenus: MenuItem[] = [
  defineMenu('partner.menu.root', {
    title: '伙伴管理',
    icon: UserFilled,
    sequence: 40,
    children: [
      defineMenu('partner.menu.partner_list', {
        title: '伙伴列表',
        path: '/partner/partners',
        sequence: 10,
      }),
    ],
  }),
];
