// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  propertyDefinitionListComponent,
  propertyDefinitionRoutes,
} from './propertyDefinitionRoutes';

test('propertyDefinitionListComponent is exported for domain list routes', () => {
  expect(typeof propertyDefinitionListComponent).toBe('function');
});

test('propertyDefinitionRoutes host web list plus create/detail; no core', () => {
  const names = propertyDefinitionRoutes.map(r => String(r.name));
  expect(names).toEqual([
    'PropertyDefinitionList_web',
    'PropertyDefinitionCreate',
    'PropertyDefinitionDetail',
  ]);

  const webList = propertyDefinitionRoutes.find(
    r => r.name === 'PropertyDefinitionList_web'
  ) as any;
  expect(webList.path).toBe('web/property-definitions/web');
  expect(webList.meta?.propertyDefinitionApplication).toBe('web');
  expect(webList.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(webList.component).toBe(propertyDefinitionListComponent);

  const create = propertyDefinitionRoutes.find(r => r.name === 'PropertyDefinitionCreate') as any;
  expect(create.path).toBe('web/property-definitions/:application/new');

  const detail = propertyDefinitionRoutes.find(r => r.name === 'PropertyDefinitionDetail') as any;
  expect(detail.path).toBe('web/property-definitions/:application/:id');

  expect(names.some(n => n.includes('core'))).toBe(false);
});
