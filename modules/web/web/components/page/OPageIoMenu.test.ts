// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import OPageIoMenu from './OPageIoMenu.vue';
import type { PageIoMenuItem } from '@/web/web/composables/recordIoTypes';

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

function mountMenu(items: PageIoMenuItem[]) {
  return mount(OPageIoMenu, {
    props: { items },
    global: {
      plugins: [i18n],
      stubs: {
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
      },
    },
  });
}

describe('OPageIoMenu', () => {
  it('renders visible items and ignores hidden ones', () => {
    const onImport = vi.fn();
    const onExport = vi.fn();
    const wrapper = mountMenu([
      { key: 'import', label: 'Import', onClick: onImport },
      { key: 'export', label: 'Export', hidden: true, onClick: onExport },
    ]);
    expect(wrapper.find('[data-test="page-io-menu-trigger"]').exists()).toBe(true);
    expect(wrapper.find('[data-test="page-io-menu-import"]').exists()).toBe(true);
    expect(wrapper.find('[data-test="page-io-menu-export"]').exists()).toBe(false);
  });

  it('hides the dropdown when every item is hidden', () => {
    const wrapper = mountMenu([
      { key: 'import', label: 'Import', hidden: true, onClick: () => undefined },
    ]);
    expect(wrapper.find('[data-test="dropdown"]').exists()).toBe(false);
  });

  it('invokes onClick for the commanded item and ignores unknown keys', async () => {
    const onImport = vi.fn();
    const wrapper = mountMenu([
      { key: 'import', label: 'Import', onClick: onImport },
    ]);
    await wrapper.findComponent({ name: 'ElDropdown' }).vm.$emit('command', 'import');
    expect(onImport).toHaveBeenCalledTimes(1);
    await wrapper.findComponent({ name: 'ElDropdown' }).vm.$emit('command', 'missing');
    expect(onImport).toHaveBeenCalledTimes(1);
  });
});
