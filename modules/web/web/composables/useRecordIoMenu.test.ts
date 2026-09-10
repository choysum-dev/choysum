// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import {
  clearGlobalRequestContextProvider,
  setGlobalRequestContextProvider,
} from '@/core/rpc/context';
import { clearFieldsByStore, registerFieldPath } from '@/web/web/query/utils/registry/field';
import { useRecordIoMenu } from './useRecordIoMenu';
import { useRecordExportScope } from './useRecordExportScope';
import { useRecordImportScope } from './useRecordImportScope';
import { fnRecorder } from '@/web/web/__tests__/mountApp';

afterEach(() => {
  clearGlobalRequestContextProvider();
  clearFieldsByStore('s1');
  clearFieldsByStore('s2');
  clearFieldsByStore('s2b');
  clearFieldsByStore('s3');
});

describe('useRecordIoMenu', () => {
  test('builds import and export items when enabled', () => {
    const openImport = fnRecorder();
    const openExport = fnRecorder();
    const { items, visible } = useRecordIoMenu({
      config: {
        import: { enabled: true },
        export: { enabled: true },
      },
      openImport,
      openExport,
    });
    expect(visible.value).toBe(true);
    expect(items.value.map(i => i.key)).toEqual(['import', 'export']);
    items.value[0].onClick();
    items.value[1].onClick();
    expect(openImport.calls.length).toBe(1);
    expect(openExport.calls.length).toBe(1);
  });

  test('hides menu when neither import nor export is enabled', () => {
    const { visible, items } = useRecordIoMenu({
      config: {},
    });
    expect(visible.value).toBe(false);
    expect(items.value).toEqual([]);
  });

  test('reads config from a ref and accepts custom labels', () => {
    const openImport = fnRecorder();
    const openExport = fnRecorder();
    const config = ref({
      import: { enabled: true },
      export: { enabled: true },
    });
    const menu = useRecordIoMenu({
      config,
      openImport,
      openExport,
      importLabel: 'Bring in',
      exportLabel: 'Ship out',
    });
    expect(menu.visible.value).toBe(true);
    expect(menu.items.value.map(i => ({ key: i.key, label: i.label }))).toEqual([
      { key: 'import', label: 'Bring in' },
      { key: 'export', label: 'Ship out' },
    ]);
    config.value = {
      export: { enabled: true },
    } as any;
    expect(menu.items.value.map(i => ({ key: i.key, label: i.label }))).toEqual([
      { key: 'export', label: 'Ship out' },
    ]);
    menu.items.value[0].onClick();
    expect(openExport.calls.length).toBe(1);
  });

  test('skips items when open callbacks are missing', () => {
    const { items, visible } = useRecordIoMenu({
      config: {
        import: { enabled: true },
        export: { enabled: true },
      },
    });
    expect(items.value).toEqual([]);
    expect(visible.value).toBe(false);
  });

  test('tolerates a nullish config value', () => {
    const { items, visible } = useRecordIoMenu({
      config: ref(null) as any,
      openImport: () => undefined,
      openExport: () => undefined,
    });
    expect(items.value).toEqual([]);
    expect(visible.value).toBe(false);
  });
});

describe('useRecordExportScope', () => {
  beforeEach(() => {
    setGlobalRequestContextProvider({ activeCompanyId: 'cmp-1' });
  });

  test('collects ids domain default fields and count', () => {
    registerFieldPath('s1', 'Name');
    const listRef = ref({
      selectedItems: { value: [{ Id: 'a' }, { Id: '' }] },
    });
    const scope = useRecordExportScope({
      store: { storeId: 's1', state: { result: { total: 9 } } },
      getListRef: () => listRef.value,
    });
    expect(scope.companyId.value).toBe('cmp-1');
    expect(scope.ids.value).toEqual(['a']);
    expect(scope.domain.value).toBe(JSON.stringify({ And: [] }));
    expect(scope.defaultFields.value).toEqual(['Name']);
    expect(scope.filteredCount.value).toBe(9);
  });

  test('reads selected ids from a plain array and companyId fallback', () => {
    setGlobalRequestContextProvider({ companyId: 'cmp-fallback' });
    const scope = useRecordExportScope({
      store: { storeId: 's2', state: { result: { total: 0 } } },
      getListRef: () => ({ selectedItems: [{ Id: 'x' }, { Id: 'y' }] }),
    });
    expect(scope.companyId.value).toBe('cmp-fallback');
    expect(scope.ids.value).toEqual(['x', 'y']);
  });

  test('returns empty ids when selectedItems value is not an array', () => {
    clearGlobalRequestContextProvider();
    const scope = useRecordExportScope({
      store: { storeId: 's2' },
      getListRef: () => ({ selectedItems: { value: { unexpected: true } as any } }),
    });
    expect(scope.companyId.value).toBe('');
    expect(scope.ids.value).toEqual([]);
    expect(scope.filteredCount.value).toBe(0);
  });

  test('filters blank ids from a plain selectedItems array', () => {
    const scope = useRecordExportScope({
      store: { storeId: 's2', state: { result: {} } },
      getListRef: () => ({
        selectedItems: [{ Id: 'keep' }, { Id: '' }, { Id: null as any }, null as any, {}],
      }),
    });
    expect(scope.ids.value).toEqual(['keep']);
    expect(scope.filteredCount.value).toBe(0);
  });

  test('filters nullish ids from selectedItems.value', () => {
    const scope = useRecordExportScope({
      store: { storeId: 's2b' },
      getListRef: () => ({
        selectedItems: { value: [{ Id: 'ok' }, { Id: null as any }, {}] },
      }),
    });
    expect(scope.ids.value).toEqual(['ok']);
  });

  test('returns empty ids when list ref has no selectedItems', () => {
    const scope = useRecordExportScope({
      store: { storeId: 's2', state: { result: { total: 2 } } },
      getListRef: () => ({}),
    });
    expect(scope.ids.value).toEqual([]);
    expect(scope.filteredCount.value).toBe(2);
  });

  test('returns empty default fields when selection is missing', () => {
    const scope = useRecordExportScope({
      store: { storeId: 's3' },
      getListRef: () => null,
    });
    expect(scope.defaultFields.value).toEqual([]);
    expect(scope.ids.value).toEqual([]);
  });

  test('uses empty filters when buildUnifiedQuery omits filters', () => {
    const scope = useRecordExportScope({
      store: { storeId: 's3' },
      getListRef: () => null,
    });
    expect(scope.domain.value).toBe(JSON.stringify({ And: [] }));
  });
});

describe('useRecordImportScope', () => {
  beforeEach(() => {
    setGlobalRequestContextProvider({ activeCompanyId: 'cmp-1' });
  });

  test('resolves model mapping hint and company', () => {
    const scope = useRecordImportScope({
      model: 'partner.Partner',
      config: {
        import: {
          enabled: true,
          columnMapping: { Name: 'name' },
          uploadHint: 'hint',
        },
      },
    });
    expect(scope.model.value).toBe('partner.Partner');
    expect(scope.companyId.value).toBe('cmp-1');
    expect(scope.columnMapping.value).toEqual({ Name: 'name' });
    expect(scope.uploadHint.value).toBe('hint');
  });

  test('reads config from a ref and defaults mapping', () => {
    setGlobalRequestContextProvider({ companyId: 'from-company' });
    const model = ref('partner.Partner');
    const config = ref({
      import: { enabled: true },
    });
    const scope = useRecordImportScope({ model, config });
    expect(scope.companyId.value).toBe('from-company');
    expect(scope.columnMapping.value).toEqual({});
    expect(scope.uploadHint.value).toBeUndefined();
    model.value = 'other.Model';
    config.value = {
      import: { enabled: true, uploadHint: 'next' },
    } as any;
    expect(scope.model.value).toBe('other.Model');
    expect(scope.uploadHint.value).toBe('next');
  });

  test('tolerates a nullish config value', () => {
    const scope = useRecordImportScope({ model: 'partner.Partner', config: ref(null) as any });
    expect(scope.model.value).toBe('partner.Partner');
    expect(scope.columnMapping.value).toEqual({});
    expect(scope.uploadHint.value).toBeUndefined();
  });
});
