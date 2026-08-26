// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { config, mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import { createI18n } from 'vue-i18n';

config.global.renderStubDefaultSlot = true;

const { refresh, exportFieldSelection, buildUnifiedQuery, listSelectedItems } = vi.hoisted(() => ({
  refresh: vi.fn(),
  exportFieldSelection: vi.fn(() => ['Name', 'CompanyId.Code', 'Id']),
  buildUnifiedQuery: vi.fn(() => ({ filters: { And: [{ field: 'Name', op: 'contains', value: 'A' }] } })),
  listSelectedItems: { current: { value: [{ Id: 'p1' }, { Id: 'p2' }] } as { value?: { Id?: string }[] } | { Id?: string }[] },
}));

vi.mock('vue-router', () => ({
  useRoute: () => ({ fullPath: '/partner/partners' }),
}));

vi.mock('@/core/rpc/context', () => ({
  getCurrentRequestContext: vi.fn(() => ({ activeCompanyId: 'cmp-1' })),
}));

vi.mock('@/web/web/stores/storeScopeManager', () => ({
  useScopeManager: () => ({ menuScopeManager: {} }),
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: vi.fn(() => ({
    Search: vi.fn(),
    storeId: 'Partner_/partner/partners',
    state: { result: { total: 3 }, queryState: { appliedFilters: [] } },
  })),
}));

vi.mock('@/web/web/query/utils/registry/field', () => ({
  exportFieldSelection,
}));

vi.mock('@/web/web/query/context', () => ({
  buildUnifiedQuery,
}));

vi.mock('../views/PartnerListView.vue', () => ({
  default: {
    name: 'PartnerListViewStub',
    props: ['store', 'createAction'],
    setup(_: unknown, { expose }: { expose: (exposed: Record<string, unknown>) => void }) {
      expose({
        refresh,
        selectedItems: listSelectedItems.current,
      });
      return () => null;
    },
  },
}));

vi.mock('../components/PartnerImportWizard.vue', () => ({
  default: {
    name: 'PartnerImportWizardStub',
    props: ['companyId', 'modelValue'],
    emits: ['update:modelValue', 'imported'],
    template: '<button data-test="emit-imported" @click="$emit(\'imported\')">import</button>',
  },
}));

vi.mock('@/core/web/export', () => ({
  ExportPanel: {
    name: 'ExportPanelStub',
    props: ['model', 'companyId', 'ids', 'domain', 'defaultFields', 'filteredCount', 'modelValue'],
    template: '<div data-test="export-panel-stub" />',
  },
}));

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

describe('PartnerList page', () => {
  beforeEach(async () => {
    listSelectedItems.current = { value: [{ Id: 'p1' }, { Id: 'p2' }] };
    exportFieldSelection.mockReturnValue(['Name', 'CompanyId.Code', 'Id']);
    const ctx = await import('@/core/rpc/context');
    (ctx.getCurrentRequestContext as any).mockReset();
    (ctx.getCurrentRequestContext as any).mockReturnValue({ activeCompanyId: 'cmp-1' });
  });

  it('refreshes list after import completes', async () => {
    const PartnerList = (await import('./PartnerList.vue')).default;
    const wrapper = mount(PartnerList, {
      global: {
        plugins: [i18n],
        stubs: { OPage: { template: '<div><slot /></div>' } },
      },
    });
    expect(wrapper.text()).toContain('Import CSV');
    await wrapper.find('[data-test="emit-imported"]').trigger('click');
    expect(refresh).toHaveBeenCalled();
  });

  it('renders export panel with partner model and filtered count', async () => {
    const PartnerList = (await import('./PartnerList.vue')).default;
    const wrapper = mount(PartnerList, {
      global: {
        plugins: [i18n],
        stubs: { OPage: { template: '<div><slot /></div>' } },
      },
    });
    expect(wrapper.text()).toContain('Export CSV');
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('model')).toBe('partner.Partner');
    expect(panel.props('filteredCount')).toBe(3);
    expect(panel.props('companyId')).toBe('cmp-1');
  });

  it('passes company id from request context fallback', async () => {
    const ctx = await import('@/core/rpc/context');
    (ctx.getCurrentRequestContext as any).mockReturnValue({ companyId: 'cmp-fallback' });
    const PartnerList = (await import('./PartnerList.vue')).default;
    const wrapper = mount(PartnerList, {
      global: {
        plugins: [i18n],
        stubs: { OPage: { template: '<div><slot /></div>' } },
      },
    });
    const wizard = wrapper.findComponent({ name: 'PartnerImportWizardStub' });
    expect(wizard.props('companyId')).toBe('cmp-fallback');
  });

  it('uses empty company id when request context has no company', async () => {
    const ctx = await import('@/core/rpc/context');
    (ctx.getCurrentRequestContext as any).mockReturnValue({});
    const PartnerList = (await import('./PartnerList.vue')).default;
    const wrapper = mount(PartnerList, {
      global: {
        plugins: [i18n],
        stubs: { OPage: { template: '<div><slot /></div>' } },
      },
    });
    const wizard = wrapper.findComponent({ name: 'PartnerImportWizardStub' });
    expect(wizard.props('companyId')).toBe('');
  });

  it('passes selected ids, domain, and normalized default fields to export panel', async () => {
    const PartnerList = (await import('./PartnerList.vue')).default;
    const wrapper = mount(PartnerList, {
      global: {
        plugins: [i18n],
        stubs: { OPage: { template: '<div><slot /></div>' } },
      },
    });
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('ids')).toEqual(['p1', 'p2']);
    expect(panel.props('domain')).toBe(JSON.stringify({ And: [{ field: 'Name', op: 'contains', value: 'A' }] }));
    expect(panel.props('defaultFields')).toEqual(['Name', 'CompanyId/Code']);
    expect(buildUnifiedQuery).toHaveBeenCalled();
    expect(exportFieldSelection).toHaveBeenCalledWith('Partner_/partner/partners');
  });

  it('reads selected ids from a plain selectedItems array', async () => {
    listSelectedItems.current = [{ Id: 'p9' }];
    const PartnerList = (await import('./PartnerList.vue')).default;
    const wrapper = mount(PartnerList, {
      global: {
        plugins: [i18n],
        stubs: { OPage: { template: '<div><slot /></div>' } },
      },
    });
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('ids')).toEqual(['p9']);
  });

  it('uses empty default fields when registry selection is missing', async () => {
    exportFieldSelection.mockReturnValue(null as unknown as string[]);
    const PartnerList = (await import('./PartnerList.vue')).default;
    const wrapper = mount(PartnerList, {
      global: {
        plugins: [i18n],
        stubs: { OPage: { template: '<div><slot /></div>' } },
      },
    });
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('defaultFields')).toEqual([]);
  });

  it('filters blank partner ids from export scope', async () => {
    listSelectedItems.current = [{ Id: 'p9' }, { Id: '' }, { Id: '  ' }];
    const PartnerList = (await import('./PartnerList.vue')).default;
    const wrapper = mount(PartnerList, {
      global: {
        plugins: [i18n],
        stubs: { OPage: { template: '<div><slot /></div>' } },
      },
    });
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('ids')).toEqual(['p9']);
  });
});
