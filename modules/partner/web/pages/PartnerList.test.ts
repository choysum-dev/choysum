// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { config, mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import { createI18n } from 'vue-i18n';

config.global.renderStubDefaultSlot = true;

const { refresh, exportFieldSelection, buildUnifiedQuery, listSelectedItems, createStoreByModelMock } = vi.hoisted(() => ({
  refresh: vi.fn(),
  exportFieldSelection: vi.fn(() => ['Name', 'CompanyId.Code', 'Id']),
  buildUnifiedQuery: vi.fn(() => ({ filters: { And: [{ field: 'Name', op: 'contains', value: 'A' }] } })),
  listSelectedItems: { current: { value: [{ Id: 'p1' }, { Id: 'p2' }] } as { value?: { Id?: string }[] } | { Id?: string }[] },
  createStoreByModelMock: vi.fn(() => ({
    Search: vi.fn(),
    storeId: 'Partner_/partner/partners',
    state: { result: { total: 3 }, queryState: { appliedFilters: [] } },
  })),
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
  createStoreByModel: createStoreByModelMock,
}));

vi.mock('@/web/web/export', () => ({
  ExportPanel: {
    name: 'ExportPanelStub',
    props: ['model', 'companyId', 'ids', 'domain', 'defaultFields', 'filteredCount', 'modelValue'],
    emits: ['update:modelValue'],
    template: '<div data-test="export-panel-stub" />',
  },
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
        get selectedItems() {
          return listSelectedItems.current;
        },
      });
      return () => null;
    },
  },
}));

const partnerImportWizardStub = {
  name: 'PartnerImportWizardStub',
  props: ['companyId', 'modelValue'],
  emits: ['update:modelValue', 'imported'],
  template: '<div :data-company-id="companyId" :data-open="String(modelValue)"><button data-test="wizard-close" @click="$emit(\'update:modelValue\', false)">close</button><button data-test="emit-imported" @click="$emit(\'imported\')">import</button></div>',
};

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

async function mountPartnerList(extraStubs: Record<string, unknown> = {}) {
  const PartnerList = (await import('./PartnerList.vue')).default;
  return mount(PartnerList, {
    global: {
      plugins: [i18n],
      stubs: {
        OPage: { template: '<div><slot /></div>' },
        PartnerImportWizard: partnerImportWizardStub,
        ...extraStubs,
      },
    },
  });
}

describe('PartnerList page', () => {
  beforeEach(async () => {
    listSelectedItems.current = { value: [{ Id: 'p1' }, { Id: 'p2' }] };
    exportFieldSelection.mockReturnValue(['Name', 'CompanyId.Code', 'Id']);
    createStoreByModelMock.mockReturnValue({
      Search: vi.fn(),
      storeId: 'Partner_/partner/partners',
      state: { result: { total: 3 }, queryState: { appliedFilters: [] } },
    });
    buildUnifiedQuery.mockReturnValue({ filters: { And: [{ field: 'Name', op: 'contains', value: 'A' }] } });
    const ctx = await import('@/core/rpc/context');
    (ctx.getCurrentRequestContext as any).mockReset();
    (ctx.getCurrentRequestContext as any).mockReturnValue({ activeCompanyId: 'cmp-1' });
  });

  it('refreshes list after import completes', async () => {
    const wrapper = await mountPartnerList();
    expect(wrapper.text()).toContain('Import CSV');
    await wrapper.find('[data-test="emit-imported"]').trigger('click');
    expect(refresh).toHaveBeenCalled();
  });

  it('renders export panel with partner model and filtered count', async () => {
    const wrapper = await mountPartnerList();
    expect(wrapper.text()).toContain('Export CSV');
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('model')).toBe('partner.Partner');
    expect(panel.props('filteredCount')).toBe(3);
    expect(panel.props('companyId')).toBe('cmp-1');
  });

  it('passes company id from request context fallback', async () => {
    const ctx = await import('@/core/rpc/context');
    (ctx.getCurrentRequestContext as any).mockReturnValue({ companyId: 'cmp-fallback' });
    const wrapper = await mountPartnerList();
    const wizard = wrapper.findComponent({ name: 'PartnerImportWizardStub' });
    expect(wizard.props('companyId')).toBe('cmp-fallback');
  });

  it('uses empty company id when request context has no company', async () => {
    const ctx = await import('@/core/rpc/context');
    (ctx.getCurrentRequestContext as any).mockReturnValue({});
    const wrapper = await mountPartnerList();
    const wizard = wrapper.findComponent({ name: 'PartnerImportWizardStub' });
    expect(wizard.props('companyId')).toBe('');
  });

  it('passes selected ids, domain, and normalized default fields to export panel', async () => {
    const wrapper = await mountPartnerList();
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
    const wrapper = await mountPartnerList();
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('ids')).toEqual(['p9']);
  });

  it('uses empty default fields when registry selection is missing', async () => {
    exportFieldSelection.mockReturnValue(null as unknown as string[]);
    const wrapper = await mountPartnerList();
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('defaultFields')).toEqual([]);
  });

  it('filters blank partner ids from export scope', async () => {
    listSelectedItems.current = [{ Id: 'p9' }, {}, { Id: '' }, { Id: '  ' }];
    const wrapper = await mountPartnerList();
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('ids')).toEqual(['p9']);
  });

  it('filters blank partner ids from ref-backed selected items', async () => {
    listSelectedItems.current = { value: [{ Id: 'p9' }, { Id: undefined }, {}] };
    const wrapper = await mountPartnerList();
    await nextTick();
    expect((wrapper.vm as any).exportIds).toEqual(['p9']);
  });

  it('opens export panel from toolbar button', async () => {
    const wrapper = await mountPartnerList({
      ElButton: {
        emits: ['click'],
        template: '<button @click="$emit(\'click\')"><slot /></button>',
      },
    });
    const exportButton = wrapper.findAll('button').find(button => button.text().includes('Export CSV'));
    expect(exportButton).toBeTruthy();
    await exportButton!.trigger('click');
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('modelValue')).toBe(true);
  });

  it('uses default domain when unified query has no filters', async () => {
    buildUnifiedQuery.mockReturnValue({});
    const wrapper = await mountPartnerList();
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('domain')).toBe(JSON.stringify({ And: [] }));
  });

  it('uses zero filtered count when store result is missing', async () => {
    createStoreByModelMock.mockReturnValue({
      Search: vi.fn(),
      storeId: 'Partner_/partner/partners',
      state: { queryState: { appliedFilters: [] } },
    });
    const wrapper = await mountPartnerList();
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('filteredCount')).toBe(0);
  });

  it('opens import wizard from toolbar button', async () => {
    const wrapper = await mountPartnerList({
      ElButton: {
        emits: ['click'],
        template: '<button @click="$emit(\'click\')"><slot /></button>',
      },
    });
    const importButton = wrapper.findAll('button').find(button => button.text().includes('Import CSV'));
    await importButton!.trigger('click');
    const wizard = wrapper.findComponent({ name: 'PartnerImportWizardStub' });
    expect(wizard.props('modelValue')).toBe(true);
    expect(wizard.attributes('data-company-id')).toBe('cmp-1');
    expect(wizard.attributes('data-open')).toBe('true');
  });

  it('syncs import wizard open state from child updates', async () => {
    const wrapper = await mountPartnerList();
    const wizard = wrapper.findComponent({ name: 'PartnerImportWizardStub' });
    await wizard.find('[data-test="wizard-close"]').trigger('click');
    expect(wizard.props('modelValue')).toBe(false);
    expect(wizard.attributes('data-open')).toBe('false');
  });

  it('uses empty selected ids when selectedItems is null', async () => {
    listSelectedItems.current = null;
    const wrapper = await mountPartnerList();
    await nextTick();
    expect((wrapper.vm as any).exportIds).toEqual([]);
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('ids')).toEqual([]);
  });

  it('uses empty selected ids when selectedItems wrapper value is null', async () => {
    listSelectedItems.current = { value: null };
    const wrapper = await mountPartnerList();
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('ids')).toEqual([]);
  });

  it('uses empty selected ids when selectedItems ref has no value', async () => {
    listSelectedItems.current = { value: undefined };
    const wrapper = await mountPartnerList();
    await nextTick();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    expect(panel.props('ids')).toEqual([]);
  });

  it('syncs export panel open state from child updates', async () => {
    const wrapper = await mountPartnerList();
    const panel = wrapper.findComponent({ name: 'ExportPanelStub' });
    await panel.vm.$emit('update:modelValue', false);
    expect(panel.props('modelValue')).toBe(false);
  });

  it('does not fail import refresh when list view exposes no refresh', async () => {
    const wrapper = await mountPartnerList();
    (wrapper.vm as any).listViewRef = null;
    expect(() => (wrapper.vm as any).onImported()).not.toThrow();
  });
});
