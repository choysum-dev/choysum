// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
import { Settings } from 'lucide-vue-next';

const { _lt } = createTranslate('web', { scope: 'web/menu/menus' });

/**
 * Platform shell menus. Domain modules attach their own roots; Settings is the
 * cross-cutting configuration mount point. Each installed application registers
 * Settings → {app} → Translations; web hosts core + web here.
 */
export const menus: MenuItem[] = [
  defineMenu('web.menu.settings', {
    title: _lt('Settings'),
    sequence: 110,
    icon: Settings,
    defaultRoles: ['sys.admin'],
    children: [
      // Pathless folder: core has no business web entry; web hosts its terminology.
      defineMenu('web.menu.settings_app_core', {
        title: _lt('core'),
        sequence: 10,
        defaultRoles: ['sys.admin'],
        children: [
          defineMenu('web.menu.terminology_core', {
            title: _lt('Translations'),
            path: '/web/terminology/core',
            sequence: 10,
            defaultRoles: ['sys.admin'],
          }),
        ],
      }),
      defineMenu('web.menu.settings_app_web', {
        title: _lt('web'),
        sequence: 40,
        defaultRoles: ['sys.admin'],
        children: [
          defineMenu('web.menu.terminology', {
            title: _lt('Translations'),
            path: '/web/terminology/web',
            sequence: 10,
            defaultRoles: ['sys.admin'],
          }),
        ],
      }),
    ],
  }),
];
