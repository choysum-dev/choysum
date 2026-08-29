// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ref } from 'vue';
import { useRecordIoMenu } from './useRecordIoMenu';
import { useRecordExportScope } from './useRecordExportScope';
import { useRecordImportScope } from './useRecordImportScope';

const { getCurrentRequestContext, buildUnifiedQuery, exportFieldSelection } = vi.hoisted(() => ({
  getCurrentRequestContext: vi.fn(() => ({ activeCompanyId: 'cmp-1' })),
  buildUnifiedQuery: vi.fn(() => ({ filters: { And: [] } })),
  exportFieldSelection: vi.fn(() => ['Name', 'Id']),
}));

vi.mock('@/core/rpc/context', () => ({
  getCurrentRequestContext,
}));

vi.mock('@/web/web/query/context', () => ({
  buildUnifiedQuery,
}));

vi.mock('@/web/web/query/utils/registry/field', () => ({
  exportFieldSelection,
}));

vi.mock('@/core/web/export/field_paths', () => ({
  normalizeExportFieldPaths: (paths: string[]) => paths.map(p => p.replace(/\./g, '/')),
}));

describe('useRecordIoMenu', () => {
  it('builds import and export items when enabled', () => {
    const openImport = vi.fn();
    const openExport = vi.fn();
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
    expect(openImport).toHaveBeenCalled();
    expect(openExport).toHaveBeenCalled();
  });

  it('hides menu when neither import nor export is enabled', () => {
    const { visible, items } = useRecordIoMenu({
      config: {},
    });
    expect(visible.value).toBe(false);
    expect(items.value).toEqual([]);
  });

  it('reads config from a ref and accepts custom labels', () => {
    const openImport = vi.fn();
    const openExport = vi.fn();
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
    };
    expect(menu.items.value.map(i => ({ key: i.key, label: i.label }))).toEqual([
      { key: 'export', label: 'Ship out' },
    ]);
    menu.items.value[0].onClick();
    expect(openExport).toHaveBeenCalled();
  });
  it('skips items when open callbacks are missing', () => {
    const { items, visible } = useRecordIoMenu({
      config: {
        import: { enabled: true },
        export: { enabled: true },
      },
    });
    expect(items.value).toEqual([]);
    expect(visible.value).toBe(false);
  });

  it('tolerates a nullish config value', () => {
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
    getCurrentRequestContext.mockReturnValue({ activeCompanyId: 'cmp-1' });
    buildUnifiedQuery.mockReturnValue({ filters: { And: [] } });
    exportFieldSelection.mockReturnValue(['Name', 'Id']);
  });

  it('collects ids domain default fields and count', () => {
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

  it('reads selected ids from a plain array and companyId fallback', () => {
    getCurrentRequestContext.mockReturnValue({ companyId: 'cmp-fallback' });
    const scope = useRecordExportScope({
      store: { storeId: 's2', state: { result: { total: 0 } } },
      getListRef: () => ({ selectedItems: [{ Id: 'x' }, { Id: 'y' }] }),
    });
    expect(scope.companyId.value).toBe('cmp-fallback');
    expect(scope.ids.value).toEqual(['x', 'y']);
  });

  it('returns empty ids when selectedItems value is not an array', () => {
    getCurrentRequestContext.mockReturnValue(null);
    const scope = useRecordExportScope({
      store: {},
      getListRef: () => ({ selectedItems: { value: { unexpected: true } as any } }),
    });
    expect(scope.companyId.value).toBe('');
    expect(scope.ids.value).toEqual([]);
    expect(scope.filteredCount.value).toBe(0);
  });

  it('filters blank ids from a plain selectedItems array', () => {
    const scope = useRecordExportScope({
      store: { storeId: 's2', state: { result: {} } },
      getListRef: () => ({
        selectedItems: [{ Id: 'keep' }, { Id: '' }, { Id: null as any }, null as any, {}],
      }),
    });
    expect(scope.ids.value).toEqual(['keep']);
    expect(scope.filteredCount.value).toBe(0);
  });

  it('filters nullish ids from selectedItems.value', () => {
    const scope = useRecordExportScope({
      store: { storeId: 's2b' },
      getListRef: () => ({
        selectedItems: { value: [{ Id: 'ok' }, { Id: null as any }, {}] },
      }),
    });
    expect(scope.ids.value).toEqual(['ok']);
  });

  it('returns empty ids when list ref has no selectedItems', () => {
    const scope = useRecordExportScope({
      store: { state: { result: { total: 2 } } },
      getListRef: () => ({}),
    });
    expect(scope.ids.value).toEqual([]);
    expect(scope.filteredCount.value).toBe(2);
  });

  it('returns empty default fields when selection is missing', () => {
    exportFieldSelection.mockReturnValue(null);
    const scope = useRecordExportScope({
      store: {},
      getListRef: () => null,
    });
    expect(scope.defaultFields.value).toEqual([]);
    expect(scope.ids.value).toEqual([]);
  });

  it('uses empty filters when buildUnifiedQuery omits filters', () => {
    buildUnifiedQuery.mockReturnValue({});
    const scope = useRecordExportScope({
      store: { storeId: 's3' },
      getListRef: () => null,
    });
    expect(scope.domain.value).toBe(JSON.stringify({ And: [] }));
  });
});

describe('useRecordImportScope', () => {
  beforeEach(() => {
    getCurrentRequestContext.mockReturnValue({ activeCompanyId: 'cmp-1' });
  });

  it('resolves model mapping hint and company', () => {
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

  it('reads config from a ref and defaults mapping', () => {
    getCurrentRequestContext.mockReturnValue({ companyId: 'from-company' });
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
    };
    expect(scope.model.value).toBe('other.Model');
    expect(scope.uploadHint.value).toBe('next');
  });

  it('tolerates a nullish config value', () => {
    const scope = useRecordImportScope({ model: 'partner.Partner', config: ref(null) as any });
    expect(scope.model.value).toBe('partner.Partner');
    expect(scope.columnMapping.value).toEqual({});
    expect(scope.uploadHint.value).toBeUndefined();
  });
});
