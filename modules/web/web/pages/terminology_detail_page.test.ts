// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope } from 'vue';
import { useTerminologyDetailPage } from './terminology_detail_page';

test('useTerminologyDetailPage binds and clears store from application param', () => {
  const created: string[] = [];
  const scope = effectScope();
  const api = scope.run(() =>
    useTerminologyDetailPage({
      route: {
        fullPath: '/web/terminology/web/1',
        params: { application: 'web', id: '1' },
      } as any,
      scopeManager: { id: 'menu' },
      createStore: ((model: string) => {
        created.push(model);
        return { model } as any;
      }) as any,
    })
  )!;
  expect(api.application.value).toBe('web');
  expect(created).toEqual(['web.TranslationTerm']);
  expect(api.termStore.value).toEqual({ model: 'web.TranslationTerm' });
  scope.stop();

  const emptyScope = effectScope();
  const empty = emptyScope.run(() =>
    useTerminologyDetailPage({
      route: { fullPath: '/x', params: { application: '  ' } } as any,
      scopeManager: {},
      createStore: (() => ({}) ) as any,
    })
  )!;
  expect(empty.application.value).toBe('');
  expect(empty.termStore.value).toBeNull();
  emptyScope.stop();
});
