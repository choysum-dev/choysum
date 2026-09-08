// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { useRecordExportScope } from './useRecordExportScope';

function deps(overrides: {
  company?: Record<string, unknown>;
  filters?: unknown;
  fields?: string[];
} = {}) {
  return {
    getCurrentRequestContext: (() =>
      overrides.company ?? { activeCompanyId: 'cmp-1' }) as any,
    buildUnifiedQuery: (() => ({
      filters: overrides.filters ?? { And: [{ field: 'Name', op: 'eq', value: 'A' }] },
    })) as any,
    exportFieldSelection: (() => overrides.fields ?? ['Name', 'Id']) as any,
  };
}

test('useRecordExportScope: injects export scope from store and list selection', () => {
  const listRef = {
    selectedItems: { value: [{ Id: 'p1' }, { Id: 'p2' }] },
  };
  const store = {
    storeId: 'Partner_/partner/partners',
    state: { result: { total: 7 } },
  };
  const scope = useRecordExportScope({
    store,
    getListRef: () => listRef,
    ...deps(),
  });
  expect(scope.companyId.value).toBe('cmp-1');
  expect(scope.ids.value).toEqual(['p1', 'p2']);
  expect(scope.domain.value).toBe(
    JSON.stringify({ And: [{ field: 'Name', op: 'eq', value: 'A' }] }),
  );
  expect(scope.defaultFields.value).toEqual(['Name']);
  expect(scope.filteredCount.value).toBe(7);
});

test('useRecordExportScope: companyId from request context', () => {
  const scope = useRecordExportScope({
    store: { storeId: 's1' },
    getListRef: () => null,
    ...deps({ company: { activeCompanyId: 'cmp-scope' } }),
  });
  expect(scope.companyId.value).toBe('cmp-scope');
  expect(scope.ids.value).toEqual([]);
});

test('useRecordExportScope: empty company when context has no company ids', () => {
  const scope = useRecordExportScope({
    store: { storeId: 's1' },
    getListRef: () => null,
    ...deps({ company: {} }),
  });
  expect(scope.companyId.value).toBe('');
});

test('useRecordExportScope: reads selection from a reactive list ref', () => {
  const selected = ref([{ Id: 'live' }]);
  const scope = useRecordExportScope({
    store: { storeId: 's1' },
    getListRef: () => ({ selectedItems: selected }),
    ...deps(),
  });
  expect(scope.ids.value).toEqual(['live']);
  selected.value = [{ Id: 'next' }];
  expect(scope.ids.value).toEqual(['next']);
});

test('useRecordExportScope: accepts plain selectedItems array', () => {
  const scope = useRecordExportScope({
    store: { storeId: 's1', state: { result: { total: 1 } } },
    getListRef: () => ({ selectedItems: [{ Id: 'plain' }] }),
    ...deps({ fields: ['Code', 'Id'], filters: { And: [] } }),
  });
  expect(scope.ids.value).toEqual(['plain']);
  expect(scope.defaultFields.value).toEqual(['Code']);
  expect(scope.domain.value).toBe(JSON.stringify({ And: [] }));
  expect(scope.filteredCount.value).toBe(1);
});
