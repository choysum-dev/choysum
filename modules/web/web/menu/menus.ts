// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
import { Settings } from 'lucide-vue-next';

const { _lt } = createTranslate('web', { scope: 'web/menu/menus' });

/**
 * Platform shell menus. Domain modules attach their own roots; Settings is the
 * cross-cutting configuration mount point. Child order under Settings → {app}:
 * Settings (when present) → Parameters → Translations → Property Definitions →
 * Field Defaults. Under Settings → web only: Saved Filters + Export Templates
 * (Owner B/web admin ledgers; not mounted per business app). core has no
 * PropertyDefinition / FieldDefault / AppSetting store.
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
            sequence: 30,
            defaultRoles: ['sys.admin'],
          }),
        ],
      }),
      defineMenu('web.menu.settings_app_web', {
        title: _lt('web'),
        sequence: 40,
        defaultRoles: ['sys.admin'],
        children: [
          defineMenu('web.menu.app_settings_parameters', {
            title: _lt('Parameters'),
            path: '/web/app-settings/web',
            sequence: 20,
            defaultRoles: ['sys.admin'],
          }),
          defineMenu('web.menu.terminology', {
            title: _lt('Translations'),
            path: '/web/terminology/web',
            sequence: 30,
            defaultRoles: ['sys.admin'],
          }),
          defineMenu('web.menu.property_definitions', {
            title: _lt('Property Definitions'),
            path: '/web/property-definitions/web',
            sequence: 40,
            defaultRoles: ['sys.admin'],
          }),
          defineMenu('web.menu.field_defaults', {
            title: _lt('Field Defaults'),
            path: '/web/field-defaults/web',
            sequence: 50,
            defaultRoles: ['sys.admin'],
          }),
          defineMenu('web.menu.user_filters', {
            title: _lt('Saved Filters'),
            path: '/web/user-filters',
            sequence: 60,
            defaultRoles: ['sys.admin'],
          }),
          defineMenu('web.menu.export_templates', {
            title: _lt('Export Templates'),
            path: '/web/export-templates',
            sequence: 70,
            defaultRoles: ['sys.admin'],
          }),
        ],
      }),
    ],
  }),
];
