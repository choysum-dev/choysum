// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { userFilterListComponent, userFilterRoutes } from './userFilterRoutes';

test('userFilterListComponent is exported', () => {
  expect(typeof userFilterListComponent).toBe('function');
});

test('userFilterRoutes host web list plus create/detail with full CRUD actions', () => {
  const names = userFilterRoutes.map(r => String(r.name));
  expect(names).toEqual(['UserFilterList_web', 'UserFilterCreate', 'UserFilterDetail']);

  const list = userFilterRoutes.find(r => r.name === 'UserFilterList_web') as any;
  expect(list.path).toBe('web/user-filters');
  expect(list.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(list.component).toBe(userFilterListComponent);
  expect(list.meta?.resource?.actions).toEqual([
    'web.action.user_filter_create',
    'web.action.user_filter_edit',
    'web.action.user_filter_delete',
    'web.action.user_filter_copy',
  ]);

  const create = userFilterRoutes.find(r => r.name === 'UserFilterCreate') as any;
  expect(create.path).toBe('web/user-filters/new');

  const detail = userFilterRoutes.find(r => r.name === 'UserFilterDetail') as any;
  expect(detail.path).toBe('web/user-filters/:id');

  const createIdx = userFilterRoutes.findIndex(r => r.name === 'UserFilterCreate');
  const detailIdx = userFilterRoutes.findIndex(r => r.name === 'UserFilterDetail');
  expect(createIdx).toBeGreaterThanOrEqual(0);
  expect(createIdx).toBeLessThan(detailIdx);
});
