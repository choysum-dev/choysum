// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useUserFilterListPage } from './user_filter_list_page';

test('useUserFilterListPage binds fixed web.UserFilter store and title', () => {
  const created: string[] = [];
  const api = useUserFilterListPage({
    scopeManager: {},
    createStore: ((name: string) => {
      created.push(name);
      return { id: name } as any;
    }) as any,
    translate: m => m,
  });
  expect(api.pageTitle.value).toBe('Saved Filters');
  expect(api.userFilterStore.value).toEqual({ id: 'web.UserFilter' });
  expect(created).toEqual(['web.UserFilter']);
});
