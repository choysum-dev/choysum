// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, reactive } from 'vue';
import { useAppSettingListPage } from './app_setting_list_page';

test('useAppSettingListPage binds store and title; empty app clears store', async () => {
  const created: string[] = [];
  const route = reactive({
    meta: { appSettingApplication: 'auth' } as Record<string, unknown>,
    params: {} as Record<string, unknown>,
    path: '/web/app-settings/auth',
  });
  const api = useAppSettingListPage({
    route,
    scopeManager: {},
    createStore: ((model: string) => {
      created.push(model);
      return { id: model } as any;
    }) as any,
    translate: (msgid, app) => `${msgid}:${app}`,
  });
  await nextTick();
  expect(api.application.value).toBe('auth');
  expect(api.pageTitle.value).toBe('Parameters: %s:auth');
  expect(api.appSettingStore.value).toEqual({ id: 'auth.AppSetting' });
  expect(created).toEqual(['auth.AppSetting']);

  route.meta = {};
  route.params = {};
  route.path = '/web/app-settings/';
  await nextTick();
  expect(api.application.value).toBe('');
  expect(api.pageTitle.value).toBe('Parameters: %s:—');
  expect(api.appSettingStore.value).toBeNull();
});

test('useAppSettingListPage falls back to defaultTranslate when translate omitted', async () => {
  const route: any = {
    meta: { appSettingApplication: 'web' },
    params: {},
    path: '/web/app-settings/web',
  };
  const api = useAppSettingListPage({
    route,
    scopeManager: {},
    createStore: (() => ({ id: 's' })) as any,
  });
  await nextTick();
  expect(api.pageTitle.value).toContain('web');
});
