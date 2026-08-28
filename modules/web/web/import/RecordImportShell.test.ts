// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { nextTick } from 'vue';

const { getCurrentRequestContext } = vi.hoisted(() => ({
  getCurrentRequestContext: vi.fn(() => ({ activeCompanyId: 'cmp-1' })),
}));

vi.mock('@/core/rpc/context', () => ({
  getCurrentRequestContext,
}));

vi.mock('./ImportPanel.vue', () => ({
  default: {
    name: 'ImportPanelStub',
    props: ['modelValue', 'model', 'companyId', 'columnMapping', 'uploadHint'],
    emits: ['update:modelValue', 'imported'],
    template:
      '<div data-test="import-panel" :data-model="model" :data-company-id="companyId" :data-hint="uploadHint || \'\'" :data-mapping="JSON.stringify(columnMapping || {})" :data-open="String(modelValue)"><button data-test="emit-imported" @click="$emit(\'imported\')">done</button></div>',
  },
}));

describe('RecordImportShell', () => {
  beforeEach(() => {
    getCurrentRequestContext.mockReturnValue({ activeCompanyId: 'cmp-1' });
    vi.resetModules();
  });

  it('passes direct props through to ImportPanel', async () => {
    const { default: RecordImportShell } = await import('./RecordImportShell.vue');
    const wrapper = mount(RecordImportShell, {
      props: {
        model: 'partner.Partner',
        companyId: 'cmp-direct',
        columnMapping: { Name: 'name' },
        uploadHint: 'hint',
        open: true,
      },
    });
    const panel = wrapper.find('[data-test="import-panel"]');
    expect(panel.attributes('data-model')).toBe('partner.Partner');
    expect(panel.attributes('data-company-id')).toBe('cmp-direct');
    expect(panel.attributes('data-hint')).toBe('hint');
    expect(panel.attributes('data-mapping')).toBe(JSON.stringify({ Name: 'name' }));
    expect(panel.attributes('data-open')).toBe('true');
  });

  it('resolves model and defaults from config when direct model is omitted', async () => {
    const { default: RecordImportShell } = await import('./RecordImportShell.vue');
    const wrapper = mount(RecordImportShell, {
      props: {
        config: {
          model: 'from.config',
          import: {
            enabled: true,
            uploadHint: 'from-config',
            columnMapping: { Code: 'code' },
          },
        },
        open: false,
      },
    });
    const panel = wrapper.find('[data-test="import-panel"]');
    expect(panel.attributes('data-model')).toBe('from.config');
    expect(panel.attributes('data-company-id')).toBe('cmp-1');
    expect(panel.attributes('data-hint')).toBe('from-config');
    expect(panel.attributes('data-mapping')).toBe(JSON.stringify({ Code: 'code' }));
  });

  it('prefers direct props over config values', async () => {
    getCurrentRequestContext.mockReturnValue({});
    const { default: RecordImportShell } = await import('./RecordImportShell.vue');
    const wrapper = mount(RecordImportShell, {
      props: {
        model: 'direct.Model',
        uploadHint: 'direct-hint',
        columnMapping: { A: 'a' },
        config: {
          model: 'config.Model',
          import: {
            enabled: true,
            uploadHint: 'config-hint',
            columnMapping: { B: 'b' },
          },
        },
      },
    });
    const panel = wrapper.find('[data-test="import-panel"]');
    expect(panel.attributes('data-model')).toBe('direct.Model');
    expect(panel.attributes('data-hint')).toBe('direct-hint');
    expect(panel.attributes('data-mapping')).toBe(JSON.stringify({ A: 'a' }));
    expect(panel.attributes('data-company-id')).toBe('');
  });

  it('forwards imported and open updates', async () => {
    const { default: RecordImportShell } = await import('./RecordImportShell.vue');
    const wrapper = mount(RecordImportShell, {
      props: {
        model: 'partner.Partner',
        open: false,
      },
    });
    await wrapper.find('[data-test="emit-imported"]').trigger('click');
    expect(wrapper.emitted('imported')).toBeTruthy();
    await wrapper.setProps({ open: true });
    await nextTick();
    expect(wrapper.find('[data-test="import-panel"]').attributes('data-open')).toBe('true');
  });
});
