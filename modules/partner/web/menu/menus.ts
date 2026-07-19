// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';
import { UserFilled } from '@element-plus/icons-vue';
import { defineMenu } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('partner', { output: 'reference', scope: 'web/menu/menus' });

/**
 * Menu tree registered by the partner module.
 */
export const partnerMenus: MenuItem[] = [
  defineMenu('partner.menu.root', {
    title: _t('Partner Management'),
    icon: UserFilled,
    sequence: 40,
    children: [
      defineMenu('partner.menu.partner_list', {
        title: _t('Partner List'),
        path: '/partner/partners',
        sequence: 10,
      }),
    ],
  }),
];
