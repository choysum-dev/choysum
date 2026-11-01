// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, reactive } from 'vue';
import { useAppSettingDetailPage } from './app_setting_detail_page';

test('useAppSettingDetailPage creates store for application and clears when empty', async () => {
  const created: string[] = [];
  const route = reactive({ params: { application: 'sandbox' } as Record<string, unknown> });
  const api = useAppSettingDetailPage({
    route: route as any,
    scopeManager: {},
    createStore: ((model: string) => {
      created.push(model);
      return { id: model } as any;
    }) as any,
  });
  await nextTick();
  expect(api.application.value).toBe('sandbox');
  expect(api.appSettingStore.value).toEqual({ id: 'sandbox.AppSetting' });
  expect(created).toEqual(['sandbox.AppSetting']);

  route.params = {};
  await nextTick();
  expect(api.application.value).toBe('');
  expect(api.appSettingStore.value).toBeNull();
});
