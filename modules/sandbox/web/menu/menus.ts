// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { FlaskConical } from 'lucide-vue-next';
import { sandboxRecordsMenuTitle, sandboxRootMenuTitle } from './titles';

import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('sandbox', { scope: 'web/menu/menus' });

/** Settings → sandbox → Translations / Property Definitions / Field Defaults when sandbox is installed. */
export const sandboxSettingsAppMenu: MenuItem = defineMenu('sandbox.menu.settings_app', {
  title: _lt('sandbox'),
  sequence: 110,
  parentMenu: 'web.menu.settings',
  defaultRoles: ['sys.admin'],
  children: [
    defineMenu('sandbox.menu.terminology', {
      title: _lt('Translations'),
      path: '/web/terminology/sandbox',
      sequence: 10,
      defaultRoles: ['sys.admin'],
    }),
    defineMenu('sandbox.menu.property_definitions', {
      title: _lt('Property Definitions'),
      path: '/web/property-definitions/sandbox',
      sequence: 20,
      defaultRoles: ['sys.admin'],
    }),
    defineMenu('sandbox.menu.field_defaults', {
      title: _lt('Field Defaults'),
      path: '/web/field-defaults/sandbox',
      sequence: 30,
      defaultRoles: ['sys.admin'],
    }),
  ],
});

export const sandboxMenus: MenuItem[] = [
  defineMenu('sandbox.menu.root', {
    title: sandboxRootMenuTitle,
    sequence: 90,
    // Lab flask: field playground / kitchen-sink experiments (distinct from meta's Boxes).
    icon: FlaskConical,
    children: [
      defineMenu('sandbox.menu.records', {
        title: sandboxRecordsMenuTitle,
        path: '/sandbox/records',
        sequence: 10,
      }),
    ],
  }),
];
