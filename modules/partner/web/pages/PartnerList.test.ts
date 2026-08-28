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
  RecordExportShell: {
    name: 'RecordExportShellStub',
    props: ['model', 'store', 'listRef', 'companyId', 'open'],
    emits: ['update:open'],
    template: '<div data-test="export-shell-stub" :data-model="model" :data-open="String(open)" />',
  },
}));

vi.mock('@/web/web/import', () => ({
  RecordImportShell: {
    name: 'RecordImportShellStub',
    props: ['model', 'companyId', 'uploadHint', 'open'],
    emits: ['update:open', 'imported'],
    template:
      '<div data-test="import-shell-stub" :data-model="model" :data-company-id="companyId" :data-open="String(open)"><button data-test="emit-imported" @click="$emit(\'imported\')">import</button></div>',
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
        OPage: {
          template: '<div><div class="title-actions"><slot name="title-actions" /></div><slot /></div>',
        },
        OPageIoMenu: {
          name: 'OPageIoMenuStub',
          props: ['items'],
          template:
            '<div data-test="io-menu"><button v-for="item in items" :key="item.key" :data-test="`io-${item.key}`" @click="item.onClick()">{{ item.label }}</button></div>',
        },
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
    vi.resetModules();
  });

  it('refreshes list after import completes', async () => {
    const wrapper = await mountPartnerList();
    await wrapper.find('[data-test="emit-imported"]').trigger('click');
    expect(refresh).toHaveBeenCalled();
  });

  it('renders import and export shells for partner.Partner', async () => {
    const wrapper = await mountPartnerList();
    const importShell = wrapper.find('[data-test="import-shell-stub"]');
    const exportShell = wrapper.find('[data-test="export-shell-stub"]');
    expect(importShell.attributes('data-model')).toBe('partner.Partner');
    expect(exportShell.attributes('data-model')).toBe('partner.Partner');
    expect(importShell.attributes('data-company-id')).toBe('cmp-1');
  });

  it('opens import and export from title IO menu', async () => {
    const wrapper = await mountPartnerList();
    await wrapper.find('[data-test="io-import"]').trigger('click');
    expect(wrapper.find('[data-test="import-shell-stub"]').attributes('data-open')).toBe('true');
    await wrapper.find('[data-test="io-export"]').trigger('click');
    expect(wrapper.find('[data-test="export-shell-stub"]').attributes('data-open')).toBe('true');
  });

  it('uses empty company id when request context has no company', async () => {
    const ctx = await import('@/core/rpc/context');
    (ctx.getCurrentRequestContext as any).mockReturnValue({});
    const wrapper = await mountPartnerList();
    expect(wrapper.find('[data-test="import-shell-stub"]').attributes('data-company-id')).toBe('');
  });

  it('does not fail import refresh when list view exposes no refresh', async () => {
    const wrapper = await mountPartnerList();
    (wrapper.vm as any).listViewRef = null;
    expect(() => (wrapper.vm as any).onImported()).not.toThrow();
  });

  it('passes list view ref to export shell', async () => {
    const wrapper = await mountPartnerList();
    await nextTick();
    const shell = wrapper.findComponent({ name: 'RecordExportShellStub' });
    expect(shell.props('listRef')).toBeTruthy();
    expect(shell.props('store')?.storeId).toBe('Partner_/partner/partners');
  });
});
