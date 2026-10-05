// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { partnerListMenuTitle, partnerRootMenuTitle } from './titles';
import { Users } from 'lucide-vue-next';

/**
 * Menu tree registered by the partner module.
 */
export const partnerMenus: MenuItem[] = [
  defineMenu('partner.menu.root', {
    title: partnerRootMenuTitle,
    sequence: 40,
    icon: Users,
    children: [
      defineMenu('partner.menu.partner_list', {
        title: partnerListMenuTitle,
        path: '/partner/partners',
        sequence: 10,
      }),
    ],
  }),
];
