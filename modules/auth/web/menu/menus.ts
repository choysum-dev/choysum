// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { UserFilled } from '@element-plus/icons-vue';
import { defineMenu } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('auth', { scope: 'web/menu/menus' });

/**
 * Auth module menu definitions.
 *
 * Keep menu visibility concerns separate from route registration.
 */
export const authMenus: MenuItem[] = [
  defineMenu('auth.menu.root', {
    title: _lt('Access Control'),
    icon: UserFilled,
    sequence: 100,
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
      // Pathless group: cross-role / everyone rule catalog (PR-C-5). Flat leaf URLs.
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
