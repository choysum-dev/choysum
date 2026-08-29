// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { config, mount } from '@vue/test-utils';
import { defineComponent, h, nextTick } from 'vue';
import { createI18n } from 'vue-i18n';
import { provideOPageContext } from '@/web/web/composables/usePageContext';

config.global.renderStubDefaultSlot = true;

const { refresh, createStoreByModelMock } = vi.hoisted(() => ({
  refresh: vi.fn(),
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
    props: ['model', 'store', 'listRef', 'companyId', 'open', 'modelValue'],
    emits: ['update:open', 'update:modelValue'],
    template: '<div data-test="export-shell-stub" :data-model="model" :data-open="String(open ?? modelValue)" />',
  },
}));

vi.mock('@/web/web/import', () => ({
  RecordImportShell: {
    name: 'RecordImportShellStub',
    props: ['config', 'companyId', 'open', 'modelValue'],
    emits: ['update:open', 'update:modelValue', 'imported'],
    template:
      '<div data-test="import-shell-stub" :data-model="config?.model" :data-company-id="companyId || \'\'" :data-open="String(open ?? modelValue)"><button data-test="emit-imported" @click="$emit(\'imported\')">import</button></div>',
  },
}));

vi.mock('../views/PartnerListView.vue', () => ({
  default: {
    name: 'PartnerListViewStub',
    props: ['store', 'createAction'],
    setup(_: unknown, { expose }: { expose: (exposed: Record<string, unknown>) => void }) {
      expose({
        refresh,
        selectedItems: { value: [{ Id: 'p1' }, { Id: 'p2' }] },
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

const OPageStub = defineComponent({
  name: 'OPageStub',
  props: {
    title: { type: String, default: '' },
    store: { type: Object, default: undefined },
  },
  setup(props, { slots }) {
    provideOPageContext({ store: () => props.store });
    return () =>
      h('div', [
        h('div', { class: 'title-actions' }, slots['title-actions']?.()),
        slots.default?.(),
      ]);
  },
});

async function mountPartnerList(extraStubs: Record<string, unknown> = {}) {
  const PartnerList = (await import('./PartnerList.vue')).default;
  return mount(PartnerList, {
    global: {
      plugins: [i18n],
      stubs: {
        OPage: OPageStub,
        'el-dropdown': {
          name: 'ElDropdown',
          emits: ['command'],
          template: '<div data-test="dropdown"><slot /><slot name="dropdown" /></div>',
        },
        'el-dropdown-menu': { template: '<div><slot /></div>' },
        'el-dropdown-item': {
          props: ['command', 'disabled'],
          template:
            '<button type="button" :data-test="`page-io-menu-${command}`" :disabled="disabled"><slot /></button>',
        },
        'el-button': { template: '<button type="button"><slot /></button>' },
        'el-icon': { template: '<span><slot /></span>' },
        Setting: true,
        ...extraStubs,
      },
    },
  });
}

describe('PartnerList page', () => {
  beforeEach(async () => {
    createStoreByModelMock.mockReturnValue({
      Search: vi.fn(),
      storeId: 'Partner_/partner/partners',
      state: { result: { total: 3 }, queryState: { appliedFilters: [] } },
    });
    const ctx = await import('@/core/rpc/context');
    (ctx.getCurrentRequestContext as any).mockReset();
    (ctx.getCurrentRequestContext as any).mockReturnValue({ activeCompanyId: 'cmp-1' });
    refresh.mockClear();
    vi.resetModules();
  });

  it('provides page store to IO menu without an explicit menu store prop', async () => {
    const wrapper = await mountPartnerList();
    await nextTick();
    const menu = wrapper.findComponent({ name: 'OPageIoMenu' });
    expect(menu.props('config')).toMatchObject({ model: 'partner.Partner' });
    expect(menu.props('store')).toBeUndefined();
    expect(menu.props('listRef')).toBeTruthy();
    expect(wrapper.find('[data-test="export-shell-stub"]').attributes('data-model')).toBe('partner.Partner');
    const exportShell = wrapper.findComponent({ name: 'RecordExportShellStub' });
    expect(exportShell.props('store')?.storeId).toBe('Partner_/partner/partners');
  });

  it('opens import and export from title IO menu', async () => {
    const wrapper = await mountPartnerList();
    await wrapper.findComponent({ name: 'ElDropdown' }).vm.$emit('command', 'import');
    expect(wrapper.find('[data-test="import-shell-stub"]').attributes('data-open')).toBe('true');
    await wrapper.findComponent({ name: 'ElDropdown' }).vm.$emit('command', 'export');
    expect(wrapper.find('[data-test="export-shell-stub"]').attributes('data-open')).toBe('true');
  });

  it('refreshes list after import completes', async () => {
    const wrapper = await mountPartnerList();
    await wrapper.find('[data-test="emit-imported"]').trigger('click');
    expect(refresh).toHaveBeenCalled();
  });

  it('does not fail import refresh when list view exposes no refresh', async () => {
    refresh.mockClear();
    const wrapper = await mountPartnerList({
      PartnerListView: {
        name: 'PartnerListViewNoRefresh',
        template: '<div data-test="list-no-refresh" />',
      },
    });
    await wrapper.find('[data-test="emit-imported"]').trigger('click');
    expect(refresh).not.toHaveBeenCalled();
  });
});
