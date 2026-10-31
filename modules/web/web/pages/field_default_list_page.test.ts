// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope } from 'vue';
import { useFieldDefaultListPage } from './field_default_list_page';

test('useFieldDefaultListPage binds store from meta application and clears when empty', () => {
  const created: Array<{ model: string; storeId: string }> = [];
  const route: any = {
    fullPath: '/web/field-defaults/auth',
    path: '/web/field-defaults/auth',
    params: {},
    meta: { fieldDefaultApplication: 'auth' },
  };
  const scope = effectScope();
  const api = scope.run(() =>
    useFieldDefaultListPage({
      route,
      scopeManager: { id: 'menu' },
      createStore: ((model: string, opts?: any) => {
        created.push({ model, storeId: String(opts?.storeId || '') });
        return { model } as any;
      }) as any,
      translate: (msgid, arg) => `${msgid}:${arg}`,
    })
  )!;

  expect(api.application.value).toBe('auth');
  expect(api.pageTitle.value).toContain('auth');
  expect(created).toEqual([{ model: 'auth.FieldDefault', storeId: 'FieldDefault_auth' }]);
  expect(api.fieldDefaultStore.value).toEqual({ model: 'auth.FieldDefault' });

  scope.stop();
  const emptyScope = effectScope();
  const empty = emptyScope.run(() =>
    useFieldDefaultListPage({
      route: { fullPath: '/settings', path: '/settings', params: {}, meta: {} } as any,
      scopeManager: {},
      createStore: (() => ({})) as any,
      translate: (msgid, arg) => `${msgid}:${arg}`,
    })
  )!;
  expect(empty.application.value).toBe('');
  expect(empty.fieldDefaultStore.value).toBeNull();
  emptyScope.stop();
});
