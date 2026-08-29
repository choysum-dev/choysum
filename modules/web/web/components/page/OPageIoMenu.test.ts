// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import type { PageIoMenuItem } from '@/web/web/composables/recordIoTypes';

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

vi.mock('@/web/web/import', () => ({
  RecordImportShell: {
    name: 'RecordImportShellStub',
    props: ['modelValue', 'config', 'companyId', 'open'],
    emits: ['update:open', 'update:modelValue', 'imported'],
    template:
      '<div data-test="import-shell-stub" :data-model="config?.model" :data-company-id="companyId || \'\'" :data-open="String(open ?? modelValue)"><button data-test="emit-imported" @click="$emit(\'imported\')">import</button></div>',
  },
}));

vi.mock('@/web/web/export', () => ({
  RecordExportShell: {
    name: 'RecordExportShellStub',
    props: ['modelValue', 'model', 'store', 'listRef', 'companyId', 'open'],
    emits: ['update:open', 'update:modelValue'],
    template:
      '<div data-test="export-shell-stub" :data-model="model" :data-open="String(open ?? modelValue)" />',
  },
}));

const menuStubs = {
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
};

async function mountMenu(props: Record<string, unknown> = {}) {
  const { default: OPageIoMenu } = await import('./OPageIoMenu.vue');
  return mount(OPageIoMenu, {
    props,
    global: {
      plugins: [i18n],
      stubs: menuStubs,
    },
  });
}

describe('OPageIoMenu', () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it('renders visible items and ignores hidden ones', async () => {
    const onImport = vi.fn();
    const onExport = vi.fn();
    const wrapper = await mountMenu({
      items: [
        { key: 'import', label: 'Import', onClick: onImport },
        { key: 'export', label: 'Export', hidden: true, onClick: onExport },
      ] satisfies PageIoMenuItem[],
    });
    expect(wrapper.find('[data-test="page-io-menu-trigger"]').exists()).toBe(true);
    expect(wrapper.find('[data-test="page-io-menu-import"]').exists()).toBe(true);
    expect(wrapper.find('[data-test="page-io-menu-export"]').exists()).toBe(false);
    expect(wrapper.find('[data-test="import-shell-stub"]').exists()).toBe(false);
  });

  it('hides the dropdown when every item is hidden', async () => {
    const wrapper = await mountMenu({
      items: [{ key: 'import', label: 'Import', hidden: true, onClick: () => undefined }],
    });
    expect(wrapper.find('[data-test="dropdown"]').exists()).toBe(false);
  });

  it('treats a missing items prop as an empty list', async () => {
    const wrapper = await mountMenu({});
    expect(wrapper.find('[data-test="dropdown"]').exists()).toBe(false);
  });

  it('invokes onClick for the commanded item and ignores unknown keys', async () => {
    const onImport = vi.fn();
    const wrapper = await mountMenu({
      items: [{ key: 'import', label: 'Import', onClick: onImport }],
    });
    await wrapper.findComponent({ name: 'ElDropdown' }).vm.$emit('command', 'import');
    expect(onImport).toHaveBeenCalledTimes(1);
    await wrapper.findComponent({ name: 'ElDropdown' }).vm.$emit('command', 'missing');
    expect(onImport).toHaveBeenCalledTimes(1);
  });

  it('does not invoke onClick for hidden items when commanded', async () => {
    const onImport = vi.fn();
    const onExport = vi.fn();
    const wrapper = await mountMenu({
      items: [
        { key: 'import', label: 'Import', onClick: onImport },
        { key: 'export', label: 'Export', hidden: true, onClick: onExport },
      ],
    });
    await wrapper.findComponent({ name: 'ElDropdown' }).vm.$emit('command', 'export');
    expect(onExport).not.toHaveBeenCalled();
    expect(onImport).not.toHaveBeenCalled();
  });

  it('does not invoke onClick for disabled items when commanded', async () => {
    const onImport = vi.fn();
    const wrapper = await mountMenu({
      items: [{ key: 'import', label: 'Import', disabled: true, onClick: onImport }],
    });
    await wrapper.findComponent({ name: 'ElDropdown' }).vm.$emit('command', 'import');
    expect(onImport).not.toHaveBeenCalled();
  });

  it('derives menu and panels from config', async () => {
    const refresh = vi.fn();
    const store = { storeId: 's1', state: { result: { total: 2 } } };
    const wrapper = await mountMenu({
      config: {
        model: 'partner.Partner',
        import: { enabled: true, uploadHint: 'hint' },
        export: { enabled: true },
      },
      store,
      listRef: { refresh, selectedItems: { value: [{ Id: '1' }] } },
    });
    expect(wrapper.find('[data-test="page-io-menu-import"]').exists()).toBe(true);
    expect(wrapper.find('[data-test="page-io-menu-export"]').exists()).toBe(true);
    expect(wrapper.find('[data-test="import-shell-stub"]').attributes('data-model')).toBe('partner.Partner');
    expect(wrapper.find('[data-test="export-shell-stub"]').attributes('data-model')).toBe('partner.Partner');

    await wrapper.findComponent({ name: 'ElDropdown' }).vm.$emit('command', 'import');
    expect(wrapper.find('[data-test="import-shell-stub"]').attributes('data-open')).toBe('true');

    await wrapper.find('[data-test="emit-imported"]').trigger('click');
    expect(refresh).toHaveBeenCalled();
    expect(wrapper.emitted('imported')).toBeTruthy();
  });

  it('skips export panel when store is missing', async () => {
    const wrapper = await mountMenu({
      config: {
        model: 'partner.Partner',
        import: { enabled: true },
        export: { enabled: true },
      },
    });
    expect(wrapper.find('[data-test="import-shell-stub"]').exists()).toBe(true);
    expect(wrapper.find('[data-test="export-shell-stub"]').exists()).toBe(false);
  });
});
