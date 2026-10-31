// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope } from 'vue';
import { useFieldDefaultDetailPage } from './field_default_detail_page';

test('useFieldDefaultDetailPage binds and clears store from application param', () => {
  const created: string[] = [];
  const scope = effectScope();
  const api = scope.run(() =>
    useFieldDefaultDetailPage({
      route: {
        fullPath: '/web/field-defaults/web/1',
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
  expect(created).toEqual(['web.FieldDefault']);
  expect(api.fieldDefaultStore.value).toEqual({ model: 'web.FieldDefault' });
  scope.stop();

  const emptyScope = effectScope();
  const empty = emptyScope.run(() =>
    useFieldDefaultDetailPage({
      route: { fullPath: '/x', params: { application: '  ' } } as any,
      scopeManager: {},
      createStore: (() => ({})) as any,
    })
  )!;
  expect(empty.application.value).toBe('');
  expect(empty.fieldDefaultStore.value).toBeNull();
  emptyScope.stop();
});
