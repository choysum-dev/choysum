// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('web', { scope: 'web/route/terminologyRoutes' });

/**
 * Shared TranslationTerm list page. Domain modules mount their own list
 * defineRoute with a literal path so UI_VAL_006 can match menu paths.
 */
export const terminologyListComponent = () => import('../pages/TranslationTermList.vue');

const terminologyDetailComponent = () => import('../pages/TranslationTerm.vue');

/**
 * web hosts core + web list routes (core has no business web entry) and the
 * shared detail route. Other apps declare list routes in their own modules.
 */
export const terminologyRoutes: RouteRecordRaw[] = [
  defineRoute('core.route.terminology_list', {
    sequence: 40,
    title: _lt('Translations'),
    path: 'web/terminology/core',
    name: 'TranslationTermList_core',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'core' },
  }),
  defineRoute('web.route.terminology_list', {
    sequence: 40,
    title: _lt('Translations'),
    path: 'web/terminology/web',
    name: 'TranslationTermList_web',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'web' },
  }),
  defineRoute('web.route.terminology_detail', {
    sequence: 60,
    title: _lt('Translation Details'),
    path: 'web/terminology/:application/:id',
    name: 'TranslationTermDetail',
    component: terminologyDetailComponent,
    actions: ['web.action.translation_term_edit'],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
];
