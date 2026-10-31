// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { fieldDefaultListComponent, fieldDefaultRoutes } from './fieldDefaultRoutes';

test('fieldDefaultListComponent is exported for domain list routes', () => {
  expect(typeof fieldDefaultListComponent).toBe('function');
});

test('fieldDefaultRoutes host web list plus create/detail; no core', () => {
  const names = fieldDefaultRoutes.map(r => String(r.name));
  expect(names).toEqual(['FieldDefaultList_web', 'FieldDefaultCreate', 'FieldDefaultDetail']);

  const webList = fieldDefaultRoutes.find(r => r.name === 'FieldDefaultList_web') as any;
  expect(webList.path).toBe('web/field-defaults/web');
  expect(webList.meta?.fieldDefaultApplication).toBe('web');
  expect(webList.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(webList.component).toBe(fieldDefaultListComponent);

  const create = fieldDefaultRoutes.find(r => r.name === 'FieldDefaultCreate') as any;
  expect(create.path).toBe('web/field-defaults/:application/new');

  const detail = fieldDefaultRoutes.find(r => r.name === 'FieldDefaultDetail') as any;
  expect(detail.path).toBe('web/field-defaults/:application/:id');

  expect(names.some(n => n.includes('core'))).toBe(false);
});
