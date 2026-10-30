// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
// Side-effect: register web.action.property_definition_* UI resources.
import '../views/property_definition_actions';

const { _lt } = createTranslate('web', { scope: 'web/route/propertyDefinitionRoutes' });

/**
 * Shared PropertyDefinition list page. Domain modules mount their own list
 * defineRoute with a literal path so UI_VAL_006 can match menu paths.
 */
export const propertyDefinitionListComponent = () => import('../pages/PropertyDefinitionList.vue');

const propertyDefinitionFormComponent = () => import('../pages/PropertyDefinition.vue');

/**
 * web hosts the web list route plus shared create/detail routes.
 * Other apps declare list routes in their own modules. core is excluded
 * (PropertyDefinition is not injected for application=core).
 *
 * actions must be string-literal arrays (UI_DECL_008); ids match
 * defineModelActions('web.PropertyDefinition').
 */
export const propertyDefinitionRoutes: RouteRecordRaw[] = [
  defineRoute('web.route.property_definition_list', {
    sequence: 50,
    title: _lt('Record Properties'),
    path: 'web/property-definitions/web',
    name: 'PropertyDefinitionList_web',
    component: propertyDefinitionListComponent,
    actions: [
      'web.action.property_definition_create',
      'web.action.property_definition_edit',
      'web.action.property_definition_delete',
      'web.action.property_definition_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, propertyDefinitionApplication: 'web' },
  }),
  // Register /new before /:id so "new" is not captured as an id.
  defineRoute('web.route.property_definition_create', {
    sequence: 51,
    title: _lt('New Property Definition'),
    path: 'web/property-definitions/:application/new',
    name: 'PropertyDefinitionCreate',
    component: propertyDefinitionFormComponent,
    actions: [
      'web.action.property_definition_create',
      'web.action.property_definition_edit',
      'web.action.property_definition_delete',
      'web.action.property_definition_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
  defineRoute('web.route.property_definition_detail', {
    sequence: 52,
    title: _lt('Property Definition'),
    path: 'web/property-definitions/:application/:id',
    name: 'PropertyDefinitionDetail',
    component: propertyDefinitionFormComponent,
    actions: [
      'web.action.property_definition_create',
      'web.action.property_definition_edit',
      'web.action.property_definition_delete',
      'web.action.property_definition_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
];
