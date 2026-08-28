// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { ref } from 'vue';
import { useRecordIoMenu } from './useRecordIoMenu';
import { useRecordExportScope } from './useRecordExportScope';

vi.mock('@/core/rpc/context', () => ({
  getCurrentRequestContext: vi.fn(() => ({ activeCompanyId: 'cmp-1' })),
}));

vi.mock('@/web/web/query/context', () => ({
  buildUnifiedQuery: vi.fn(() => ({ filters: { And: [] } })),
}));

vi.mock('@/web/web/query/utils/registry/field', () => ({
  exportFieldSelection: vi.fn(() => ['Name', 'Id']),
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
        model: 'partner.Partner',
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
      config: { model: 'partner.Partner' },
    });
    expect(visible.value).toBe(false);
    expect(items.value).toEqual([]);
  });
});

describe('useRecordExportScope', () => {
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
});
