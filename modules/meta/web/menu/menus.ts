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

/** Modules app root + Catalog leaf — breadcrumb Modules > Catalog. */
export const metaMenus: MenuItem[] = [
  defineMenu('meta.menu.root', {
    title: _lt('Modules'),
    sequence: 60,
    icon: Boxes,
    defaultRoles: ['sys.admin'],
    children: [
      defineMenu('meta.menu.modules', {
        title: _lt('Catalog'),
        path: '/meta/modules',
        sequence: 10,
        defaultRoles: ['sys.admin'],
        // Catalog browse only; install/upgrade/sync stay gated by action requires.
        requires: [{ model: 'meta.MetaModuleCatalog' }],
      }),
    ],
  }),
];
