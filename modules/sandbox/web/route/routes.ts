// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
import {
  fieldDefaultListComponent,
  propertyDefinitionListComponent,
  terminologyListComponent,
} from '@/web/web/route';

const { _lt } = createTranslate('sandbox', { scope: 'web/route/routes' });

/** Sibling paths for SandboxRecord list / kanban / chart ViewSwitcher. */
const sandboxRecordViewPaths = {
  list: '/sandbox/records',
  kanban: '/sandbox/records/kanban',
  chart: '/sandbox/records/chart',
} as const;

/** Route table for the sandbox kitchen-sink pages. */
export const sandboxRoutes: RouteRecordRaw[] = [
  defineRoute('sandbox.route.record_list', {
    sequence: 10,
    title: _lt('Sandbox Record List'),
    path: 'sandbox/records',
    name: 'SandboxRecordList',
    component: () => import('../pages/SandboxRecordList.vue'),
    actions: [
      'sandbox.action.sandbox_record_create',
      'sandbox.action.sandbox_record_edit',
      'sandbox.action.sandbox_record_delete',
      'sandbox.action.sandbox_record_copy',
      'sandbox.action.sandbox_record_open_detail',
    ],
    meta: {
      requiresAuth: true,
      viewKind: 'list',
      defaultGroups: [],
      viewPaths: { ...sandboxRecordViewPaths },
    },
  }),
  defineRoute('sandbox.route.record_kanban', {
    sequence: 20,
    title: _lt('Sandbox Record Kanban'),
    path: 'sandbox/records/kanban',
    name: 'SandboxRecordKanban',
    component: () => import('../pages/SandboxRecordKanban.vue'),
    actions: [
      'sandbox.action.sandbox_record_create',
      'sandbox.action.sandbox_record_edit',
      'sandbox.action.sandbox_record_delete',
      'sandbox.action.sandbox_record_copy',
      'sandbox.action.sandbox_record_open_detail',
    ],
    meta: {
      requiresAuth: true,
      viewKind: 'kanban',
      defaultGroups: ['State'],
      viewPaths: { ...sandboxRecordViewPaths },
    },
  }),
  defineRoute('sandbox.route.record_chart', {
    sequence: 30,
    title: _lt('Sandbox Record Chart'),
    path: 'sandbox/records/chart',
    name: 'SandboxRecordChart',
    component: () => import('../pages/SandboxRecordChart.vue'),
    actions: [
      'sandbox.action.sandbox_record_create',
      'sandbox.action.sandbox_record_edit',
      'sandbox.action.sandbox_record_delete',
      'sandbox.action.sandbox_record_copy',
      'sandbox.action.sandbox_record_open_detail',
    ],
    meta: {
      requiresAuth: true,
      viewKind: 'chart',
      defaultGroups: ['State', 'IsActive'],
      viewPaths: { ...sandboxRecordViewPaths },
    },
  }),
  // Static `/new` before `/:id` so create is never captured as a record id.
  defineRoute('sandbox.route.record_create', {
    sequence: 40,
    title: _lt('New Sandbox Record'),
    path: 'sandbox/records/new',
    name: 'SandboxRecordCreate',
    component: () => import('../pages/SandboxRecord.vue'),
    actions: [
      'sandbox.action.sandbox_record_create',
      'sandbox.action.sandbox_record_edit',
      'sandbox.action.sandbox_record_delete',
      'sandbox.action.sandbox_record_copy',
      'sandbox.action.sandbox_record_open_detail',
    ],
    meta: { requiresAuth: true },
  }),
  defineRoute('sandbox.route.record_detail', {
    sequence: 50,
    title: _lt('Sandbox Record Detail'),
    path: 'sandbox/records/:id',
    name: 'SandboxRecordDetail',
    component: () => import('../pages/SandboxRecord.vue'),
    actions: [
      'sandbox.action.sandbox_record_create',
      'sandbox.action.sandbox_record_edit',
      'sandbox.action.sandbox_record_delete',
      'sandbox.action.sandbox_record_copy',
      'sandbox.action.sandbox_record_open_detail',
    ],
    meta: { requiresAuth: true },
  }),
  defineRoute('sandbox.route.tag_create', {
    sequence: 60,
    title: _lt('New Sandbox Tag'),
    path: 'sandbox/tags/new',
    name: 'SandboxTagCreate',
    component: () => import('../pages/SandboxTag.vue'),
    actions: [
      'sandbox.action.sandbox_tag_create',
      'sandbox.action.sandbox_tag_edit',
      'sandbox.action.sandbox_tag_delete',
      'sandbox.action.sandbox_tag_copy',
    ],
    meta: { requiresAuth: true },
  }),
  defineRoute('sandbox.route.tag_detail', {
    sequence: 70,
    title: _lt('Sandbox Tag Detail'),
    path: 'sandbox/tags/:id',
    name: 'SandboxTagDetail',
    component: () => import('../pages/SandboxTag.vue'),
    actions: [
      'sandbox.action.sandbox_tag_create',
      'sandbox.action.sandbox_tag_edit',
      'sandbox.action.sandbox_tag_delete',
      'sandbox.action.sandbox_tag_copy',
    ],
    meta: { requiresAuth: true },
  }),
  defineRoute('sandbox.route.terminology_list', {
    sequence: 40,
    title: _lt('Translations'),
    path: 'web/terminology/sandbox',
    name: 'TranslationTermList_sandbox',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'sandbox' },
  }),
  defineRoute('sandbox.route.property_definition_list', {
    sequence: 50,
    title: _lt('Record Properties'),
    path: 'web/property-definitions/sandbox',
    name: 'PropertyDefinitionList_sandbox',
    component: propertyDefinitionListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, propertyDefinitionApplication: 'sandbox' },
  }),
  defineRoute('sandbox.route.field_default_list', {
    sequence: 60,
    title: _lt('Field Defaults'),
    path: 'web/field-defaults/sandbox',
    name: 'FieldDefaultList_sandbox',
    component: fieldDefaultListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, fieldDefaultApplication: 'sandbox' },
  }),
];
