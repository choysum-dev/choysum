// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { nextTick, ref } from 'vue';

const { getCurrentRequestContext, buildUnifiedQuery, exportFieldSelection } = vi.hoisted(() => ({
  getCurrentRequestContext: vi.fn(() => ({ activeCompanyId: 'cmp-1' })),
  buildUnifiedQuery: vi.fn(() => ({ filters: { And: [{ field: 'Name', op: 'eq', value: 'A' }] } })),
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
  normalizeExportFieldPaths: (paths: string[]) => paths.filter(p => p !== 'Id'),
}));

vi.mock('./ExportPanel.vue', () => ({
  default: {
    name: 'ExportPanelStub',
    props: ['modelValue', 'model', 'companyId', 'ids', 'domain', 'defaultFields', 'filteredCount'],
    emits: ['update:modelValue'],
    template:
      '<div data-test="export-panel" :data-model="model" :data-company-id="companyId" :data-ids="JSON.stringify(ids || [])" :data-domain="domain" :data-fields="JSON.stringify(defaultFields || [])" :data-count="String(filteredCount ?? 0)" :data-open="String(modelValue)" />',
  },
}));

describe('RecordExportShell', () => {
  beforeEach(() => {
    getCurrentRequestContext.mockReturnValue({ activeCompanyId: 'cmp-1' });
    buildUnifiedQuery.mockReturnValue({ filters: { And: [{ field: 'Name', op: 'eq', value: 'A' }] } });
    exportFieldSelection.mockReturnValue(['Name', 'Id']);
    vi.resetModules();
  });

  it('injects export scope from store and list selection', async () => {
    const { default: RecordExportShell } = await import('./RecordExportShell.vue');
    const listRef = {
      selectedItems: { value: [{ Id: 'p1' }, { Id: 'p2' }] },
    };
    const store = {
      storeId: 'Partner_/partner/partners',
      state: { result: { total: 7 } },
    };
    const wrapper = mount(RecordExportShell, {
      props: {
        model: 'partner.Partner',
        store,
        listRef,
        open: true,
      },
    });
    const panel = wrapper.find('[data-test="export-panel"]');
    expect(panel.attributes('data-model')).toBe('partner.Partner');
    expect(panel.attributes('data-company-id')).toBe('cmp-1');
    expect(panel.attributes('data-ids')).toBe(JSON.stringify(['p1', 'p2']));
    expect(panel.attributes('data-domain')).toBe(
      JSON.stringify({ And: [{ field: 'Name', op: 'eq', value: 'A' }] }),
    );
    expect(panel.attributes('data-fields')).toBe(JSON.stringify(['Name']));
    expect(panel.attributes('data-count')).toBe('7');
    expect(panel.attributes('data-open')).toBe('true');
  });

  it('prefers an explicit companyId prop', async () => {
    const { default: RecordExportShell } = await import('./RecordExportShell.vue');
    const wrapper = mount(RecordExportShell, {
      props: {
        model: 'partner.Partner',
        store: { storeId: 's1' },
        companyId: 'cmp-override',
        open: false,
      },
    });
    expect(wrapper.find('[data-test="export-panel"]').attributes('data-company-id')).toBe('cmp-override');
  });

  it('updates open binding and tolerates a missing list ref', async () => {
    const { default: RecordExportShell } = await import('./RecordExportShell.vue');
    const wrapper = mount(RecordExportShell, {
      props: {
        model: 'partner.Partner',
        store: { storeId: 's1', state: { result: { total: 1 } } },
        open: false,
      },
    });
    expect(wrapper.find('[data-test="export-panel"]').attributes('data-ids')).toBe(JSON.stringify([]));
    await wrapper.setProps({ open: true });
    await nextTick();
    expect(wrapper.find('[data-test="export-panel"]').attributes('data-open')).toBe('true');
  });

  it('reads selection from a reactive list ref getter', async () => {
    const { default: RecordExportShell } = await import('./RecordExportShell.vue');
    const selected = ref([{ Id: 'live' }]);
    const wrapper = mount(RecordExportShell, {
      props: {
        model: 'partner.Partner',
        store: { storeId: 's1' },
        listRef: { selectedItems: selected },
        open: true,
      },
    });
    expect(wrapper.find('[data-test="export-panel"]').attributes('data-ids')).toBe(JSON.stringify(['live']));
    selected.value = [{ Id: 'next' }];
    await nextTick();
    expect(wrapper.find('[data-test="export-panel"]').attributes('data-ids')).toBe(JSON.stringify(['next']));
  });
});
