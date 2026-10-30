// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope } from 'vue';
import { useTerminologyListPage } from './terminology_list_page';

test('useTerminologyListPage binds store from meta application and clears when empty', () => {
  const created: Array<{ model: string; storeId: string }> = [];
  const route: any = {
    fullPath: '/web/terminology/auth',
    path: '/web/terminology/auth',
    params: {},
    meta: { terminologyApplication: 'auth' },
  };
  const scope = effectScope();
  const api = scope.run(() =>
    useTerminologyListPage({
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
  expect(created).toEqual([
    { model: 'auth.TranslationTerm', storeId: 'TranslationTerm_auth_/web/terminology/auth' },
  ]);
  expect(api.termStore.value).toEqual({ model: 'auth.TranslationTerm' });

  route.meta = {};
  route.path = '/settings';
  route.fullPath = '/settings';
  // Re-create with empty application
  scope.stop();
  const emptyScope = effectScope();
  const empty = emptyScope.run(() =>
    useTerminologyListPage({
      route: { fullPath: '/settings', path: '/settings', params: {}, meta: {} } as any,
      scopeManager: {},
      createStore: (() => ({}) ) as any,
      translate: (msgid, arg) => `${msgid}:${arg}`,
    })
  )!;
  expect(empty.application.value).toBe('');
  expect(empty.termStore.value).toBeNull();
  emptyScope.stop();
});
