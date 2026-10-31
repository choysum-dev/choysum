// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
import { Shield } from 'lucide-vue-next';

const { _lt } = createTranslate('auth', { scope: 'web/menu/menus' });

/**
 * Auth module menu definitions.
 *
 * Keep menu visibility concerns separate from route registration.
 */
/** Settings → auth → Translations when auth is installed. */
export const authSettingsAppMenu: MenuItem = defineMenu('auth.menu.settings_app', {
  title: _lt('auth'),
  sequence: 20,
  parentMenu: 'web.menu.settings',
  defaultRoles: ['sys.admin'],
  children: [
    defineMenu('auth.menu.terminology', {
      title: _lt('Translations'),
      path: '/web/terminology/auth',
      sequence: 10,
      defaultRoles: ['sys.admin'],
    }),
    defineMenu('auth.menu.property_definitions', {
      title: _lt('Record Properties'),
      path: '/web/property-definitions/auth',
      sequence: 20,
      defaultRoles: ['sys.admin'],
    }),
    defineMenu('auth.menu.field_defaults', {
      title: _lt('Field Defaults'),
      path: '/web/field-defaults/auth',
      sequence: 30,
      defaultRoles: ['sys.admin'],
    }),
  ],
});

export const authMenus: MenuItem[] = [
  defineMenu('auth.menu.root', {
    title: _lt('Access Control'),
    sequence: 100,
    icon: Shield,
    children: [
      defineMenu('auth.menu.user_list', {
        title: _lt('User List'),
        path: '/auth/users',
        sequence: 10,
      }),
      defineMenu('auth.menu.role_list', {
        title: _lt('Role List'),
        path: '/auth/roles',
        sequence: 30,
      }),
      // Pathless folder: cross-role / everyone rule catalog.
      defineMenu('auth.menu.access_rules', {
        title: _lt('Access Rules'),
        sequence: 35,
        children: [
          defineMenu('auth.menu.record_rule_list', {
            title: _lt('Record Rules'),
            path: '/auth/record-rules',
            sequence: 10,
          }),
          defineMenu('auth.menu.field_rule_list', {
            title: _lt('Field Rules'),
            path: '/auth/field-rules',
            sequence: 20,
          }),
          defineMenu('auth.menu.method_access_list', {
            title: _lt('Method Access'),
            path: '/auth/method-accesses',
            sequence: 30,
          }),
          defineMenu('auth.menu.ui_resource_grant_list', {
            title: _lt('UI Resource Grants'),
            path: '/auth/ui-resource-grants',
            sequence: 40,
          }),
        ],
      }),
      defineMenu('auth.menu.session_list', {
        title: _lt('Session List'),
        path: '/auth/sessions',
        sequence: 40,
      }),
      defineMenu('auth.menu.token_list', {
        title: _lt('Token List'),
        path: '/auth/tokens',
        sequence: 50,
      }),
    ],
  }),
];
