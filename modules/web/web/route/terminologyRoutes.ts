// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('web', { scope: 'web/route/terminologyRoutes' });

/**
 * Standard TranslationTerm List/Form (Scheme B).
 * UI extract is static (no `.map` / template ids): each app needs a literal
 * defineRoute path so UI_VAL_006 can match menu `/web/terminology/{app}`.
 */
const terminologyListComponent = () => import('../pages/TranslationTermList.vue');
const terminologyDetailComponent = () => import('../pages/TranslationTerm.vue');

export const terminologyRoutes: RouteRecordRaw[] = [
  defineRoute('web.route.terminology_core_list', {
    sequence: 40,
    title: _lt('Translations'),
    path: 'web/terminology/core',
    name: 'TranslationTermList_core',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'core' },
  }),
  defineRoute('web.route.terminology_auth_list', {
    sequence: 41,
    title: _lt('Translations'),
    path: 'web/terminology/auth',
    name: 'TranslationTermList_auth',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'auth' },
  }),
  defineRoute('web.route.terminology_base_list', {
    sequence: 42,
    title: _lt('Translations'),
    path: 'web/terminology/base',
    name: 'TranslationTermList_base',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'base' },
  }),
  defineRoute('web.route.terminology_web_list', {
    sequence: 43,
    title: _lt('Translations'),
    path: 'web/terminology/web',
    name: 'TranslationTermList_web',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'web' },
  }),
  defineRoute('web.route.terminology_meta_list', {
    sequence: 44,
    title: _lt('Translations'),
    path: 'web/terminology/meta',
    name: 'TranslationTermList_meta',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'meta' },
  }),
  defineRoute('web.route.terminology_partner_list', {
    sequence: 45,
    title: _lt('Translations'),
    path: 'web/terminology/partner',
    name: 'TranslationTermList_partner',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'partner' },
  }),
  defineRoute('web.route.terminology_document_list', {
    sequence: 46,
    title: _lt('Translations'),
    path: 'web/terminology/document',
    name: 'TranslationTermList_document',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'document' },
  }),
  defineRoute('web.route.terminology_task_list', {
    sequence: 47,
    title: _lt('Translations'),
    path: 'web/terminology/task',
    name: 'TranslationTermList_task',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'task' },
  }),
  defineRoute('web.route.terminology_audit_list', {
    sequence: 48,
    title: _lt('Translations'),
    path: 'web/terminology/audit',
    name: 'TranslationTermList_audit',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'audit' },
  }),
  defineRoute('web.route.terminology_message_list', {
    sequence: 49,
    title: _lt('Translations'),
    path: 'web/terminology/message',
    name: 'TranslationTermList_message',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'message' },
  }),
  defineRoute('web.route.terminology_sandbox_list', {
    sequence: 50,
    title: _lt('Translations'),
    path: 'web/terminology/sandbox',
    name: 'TranslationTermList_sandbox',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'sandbox' },
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
