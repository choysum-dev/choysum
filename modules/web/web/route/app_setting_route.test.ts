// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { appSettingListComponent, appSettingRoutes } from './appSettingRoutes';

test('appSettingListComponent is exported for domain list routes', () => {
  expect(typeof appSettingListComponent).toBe('function');
});

test('appSettingRoutes host web list plus create/detail; no core', () => {
  const names = appSettingRoutes.map(r => String(r.name));
  expect(names).toEqual(['AppSettingList_web', 'AppSettingCreate', 'AppSettingDetail']);

  const webList = appSettingRoutes.find(r => r.name === 'AppSettingList_web') as any;
  expect(webList.path).toBe('web/app-settings/web');
  expect(webList.meta?.appSettingApplication).toBe('web');
  expect(webList.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(webList.component).toBe(appSettingListComponent);

  const create = appSettingRoutes.find(r => r.name === 'AppSettingCreate') as any;
  expect(create.path).toBe('web/app-settings/:application/new');

  const detail = appSettingRoutes.find(r => r.name === 'AppSettingDetail') as any;
  expect(detail.path).toBe('web/app-settings/:application/:id');

  expect(names.some(n => n.includes('core'))).toBe(false);
});
