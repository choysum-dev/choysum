// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
import { Boxes } from 'lucide-vue-next';

const { _lt } = createTranslate('meta', { scope: 'web/menu/menus' });

/**
 * Settings → meta: Parameters then platform catalogs. External IDs is the
 * Owner=meta MetaModelData admin ledger (read-only List/Form; not per-app).
 */
export const metaSettingsAppMenu: MenuItem = defineMenu('meta.menu.settings_app', {
  title: _lt('meta'),
  sequence: 50,
  parentMenu: 'web.menu.settings',
  defaultRoles: ['sys.admin'],
  children: [
    defineMenu('meta.menu.app_settings_parameters', {
      title: _lt('Parameters'),
      path: '/web/app-settings/meta',
      sequence: 20,
      defaultRoles: ['sys.admin'],
    }),
    defineMenu('meta.menu.terminology', {
      title: _lt('Translations'),
      path: '/web/terminology/meta',
      sequence: 30,
      defaultRoles: ['sys.admin'],
    }),
    defineMenu('meta.menu.property_definitions', {
      title: _lt('Property Definitions'),
      path: '/web/property-definitions/meta',
      sequence: 40,
      defaultRoles: ['sys.admin'],
    }),
    defineMenu('meta.menu.field_defaults', {
      title: _lt('Field Defaults'),
      path: '/web/field-defaults/meta',
      sequence: 50,
      defaultRoles: ['sys.admin'],
    }),
    defineMenu('meta.menu.external_ids', {
      title: _lt('External IDs'),
      path: '/web/external-ids',
      sequence: 60,
      defaultRoles: ['sys.admin'],
    }),
  ],
});

export const metaMenus: MenuItem[] = [
  defineMenu('meta.menu.root', {
    title: _lt('Module Management'),
    sequence: 60,
    icon: Boxes,
    defaultRoles: ['sys.admin'],
    children: [
      defineMenu('meta.menu.module_board', {
        title: _lt('Module Board'),
        path: '/meta/modules',
        sequence: 10,
        defaultRoles: ['sys.admin'],
        requires: [{ model: 'meta.MetaModuleCatalog' }, { model: 'meta.MetaModuleOperation' }],
      }),
      defineMenu('meta.menu.module_list', {
        title: _lt('Module List'),
        path: '/meta/modules/list',
        sequence: 20,
        defaultRoles: ['sys.admin'],
        requires: [{ model: 'meta.MetaModuleCatalog' }, { model: 'meta.MetaModule' }],
      }),
      defineMenu('meta.menu.module_history', {
        title: _lt('Operation History'),
        path: '/meta/modules/history',
        sequence: 30,
        defaultRoles: ['sys.admin'],
        requires: [{ model: 'meta.MetaModuleOperation' }, { model: 'meta.MetaModule' }, { model: 'meta.MetaModuleCatalog' }],
      }),

    ],
  }),
];
