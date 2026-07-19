// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('partner', { output: 'reference', scope: 'web/route/routes' });

/**
 * Route table for the partner management pages.
 */
export const partnerRoutes: RouteRecordRaw[] = [
  defineRoute('partner.route.partner_list', {
    sequence: 10,
    title: _t('Partner List'),
    path: 'partner/partners',
    name: 'PartnerList',
    component: () => import('../pages/PartnerList.vue'),
    actions: [
      'partner.action.partner_create',
      'partner.action.partner_edit',
      'partner.action.partner_delete',
      'partner.action.partner_copy',
      'partner.action.partner_open_detail',
    ],
    meta: { requiresAuth: true },
  }),
  defineRoute('partner.route.partner_detail', {
    sequence: 20,
    title: _t('Partner Detail'),
    path: 'partner/partners/:id',
    name: 'PartnerDetail',
    component: () => import('../pages/Partner.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['partner.action.partner_create', 'partner.action.partner_edit', 'partner.action.partner_delete', 'partner.action.partner_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('partner.route.partner_create', {
    sequence: 30,
    title: _t('New Partner'),
    path: 'partner/partners/new',
    name: 'PartnerCreate',
    component: () => import('../pages/Partner.vue'),
    props: { viewMode: 'create' },
    actions: ['partner.action.partner_create', 'partner.action.partner_edit', 'partner.action.partner_delete', 'partner.action.partner_copy'],
    meta: { requiresAuth: true },
  }),
];
