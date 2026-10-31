// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
import { Boxes } from 'lucide-vue-next';

const { _lt } = createTranslate('meta', { scope: 'web/menu/menus' });

/** Settings → meta → Translations when meta is installed. */
export const metaSettingsAppMenu: MenuItem = defineMenu('meta.menu.settings_app', {
  title: _lt('meta'),
  sequence: 50,
  parentMenu: 'web.menu.settings',
  defaultRoles: ['sys.admin'],
  children: [
    defineMenu('meta.menu.terminology', {
      title: _lt('Translations'),
      path: '/web/terminology/meta',
      sequence: 10,
      defaultRoles: ['sys.admin'],
    }),
    defineMenu('meta.menu.property_definitions', {
      title: _lt('Record Properties'),
      path: '/web/property-definitions/meta',
      sequence: 20,
      defaultRoles: ['sys.admin'],
    }),
    defineMenu('meta.menu.field_defaults', {
      title: _lt('Field Defaults'),
      path: '/web/field-defaults/meta',
      sequence: 30,
      defaultRoles: ['sys.admin'],
    }),
  ],
});

export const metaMenus: MenuItem[] = [
  defineMenu('meta.menu.root', {
    title: _lt('Module Management'),
    sequence: 60,
    icon: Boxes,
    children: [
      defineMenu('meta.menu.module_board', {
        title: _lt('Module Board'),
        path: '/meta/modules',
        sequence: 10,
        requires: [{ model: 'meta.MetaModuleIndex' }, { model: 'meta.MetaModule' }],
      }),
      defineMenu('meta.menu.module_list', {
        title: _lt('Module List'),
        path: '/meta/modules/list',
        sequence: 20,
        requires: [{ model: 'meta.MetaModuleIndex' }, { model: 'meta.MetaModule' }],
      }),
      defineMenu('meta.menu.module_history', {
        title: _lt('Operation History'),
        path: '/meta/modules/history',
        sequence: 30,
        requires: [{ model: 'meta.ModuleManagementLog' }, { model: 'meta.MetaModule' }, { model: 'meta.MetaModuleIndex' }],
      }),
    ],
  }),
];
