// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { config, mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';

config.global.renderStubDefaultSlot = true;

const { refresh } = vi.hoisted(() => ({
  refresh: vi.fn(),
}));

vi.mock('vue-router', () => ({
  useRoute: () => ({ fullPath: '/partner/partners' }),
}));

vi.mock('@/core/rpc/context', () => ({
  getCurrentRequestContext: () => ({ activeCompanyId: 'cmp-1' }),
}));

vi.mock('@/web/web/stores/storeScopeManager', () => ({
  useScopeManager: () => ({ menuScopeManager: {} }),
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: vi.fn(() => ({ Search: vi.fn() })),
}));

vi.mock('../views/PartnerListView.vue', () => ({
  default: {
    name: 'PartnerListViewStub',
    props: ['store', 'createAction'],
    setup(_: unknown, { expose }: { expose: (exposed: Record<string, unknown>) => void }) {
      expose({ refresh });
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

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

describe('PartnerList page', () => {
  it('refreshes list after import completes', async () => {
    const PartnerList = (await import('./PartnerList.vue')).default;
    const wrapper = mount(PartnerList, {
      global: {
        plugins: [i18n],
        stubs: { OPage: { template: '<div><slot /></div>' } },
      },
    });
    expect(wrapper.text()).toContain('Import CSV');
    await wrapper.find('button').trigger('click');
    await wrapper.find('[data-test="emit-imported"]').trigger('click');
    expect(refresh).toHaveBeenCalled();
  });
});
