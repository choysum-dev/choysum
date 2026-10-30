// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope } from 'vue';
import { usePropertyDefinitionDetailPage } from './property_definition_detail_page';

test('usePropertyDefinitionDetailPage binds and clears store from application param', () => {
  const created: string[] = [];
  const scope = effectScope();
  const api = scope.run(() =>
    usePropertyDefinitionDetailPage({
      route: {
        fullPath: '/web/property-definitions/web/1',
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
  expect(created).toEqual(['web.PropertyDefinition']);
  expect(api.definitionStore.value).toEqual({ model: 'web.PropertyDefinition' });
  scope.stop();

  const emptyScope = effectScope();
  const empty = emptyScope.run(() =>
    usePropertyDefinitionDetailPage({
      route: { fullPath: '/x', params: { application: '  ' } } as any,
      scopeManager: {},
      createStore: (() => ({})) as any,
    })
  )!;
  expect(empty.application.value).toBe('');
  expect(empty.definitionStore.value).toBeNull();
  emptyScope.stop();
});
