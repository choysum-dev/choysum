// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Router } from 'vue-router';
import { terminologyListComponent, terminologyRoutes } from './terminologyRoutes';
import {
  propertyDefinitionListComponent,
  propertyDefinitionRoutes,
} from './propertyDefinitionRoutes';
import { fieldDefaultListComponent, fieldDefaultRoutes } from './fieldDefaultRoutes';

/** Register Settings → {app} → Translations pages under the authenticated shell layout. */
export function registerTerminologyRoutes(router: Router): void {
  for (const route of terminologyRoutes) {
    router.addRoute('AppLayout', route);
  }
}

/** Register Settings → {app} → Record Properties pages under the authenticated shell layout. */
export function registerPropertyDefinitionRoutes(router: Router): void {
  for (const route of propertyDefinitionRoutes) {
    router.addRoute('AppLayout', route);
  }
}

/** Register Settings → {app} → Field Defaults pages under the authenticated shell layout. */
export function registerFieldDefaultRoutes(router: Router): void {
  for (const route of fieldDefaultRoutes) {
    router.addRoute('AppLayout', route);
  }
}

export {
  terminologyListComponent,
  terminologyRoutes,
  propertyDefinitionListComponent,
  propertyDefinitionRoutes,
  fieldDefaultListComponent,
  fieldDefaultRoutes,
};
