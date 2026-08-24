// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { config, mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';

config.global.renderStubDefaultSlot = true;

vi.mock('@/web/web/components/view/OListView.vue', () => ({
  default: { name: 'OListViewStub', template: '<div />' },
}));
vi.mock('@/web/web/components/view/OSearchView.vue', () => ({ default: { template: '<div />' } }));
vi.mock('@/web/web/components/vtable/OVColumn.vue', () => ({ default: { template: '<div />' } }));
vi.mock('@/web/web/components/field/OVarCharField.vue', () => ({ default: { template: '<div />' } }));
vi.mock('@/web/web/components/field/OIntField.vue', () => ({ default: { template: '<div />' } }));
vi.mock('@/web/web/components/field/OBooleanField.vue', () => ({ default: { template: '<div />' } }));
vi.mock('@/web/web/components/field/ODatetimeField.vue', () => ({ default: { template: '<div />' } }));
vi.mock('@/web/web/components/field/OManyToOneRefField.vue', () => ({ default: { template: '<div />' } }));
vi.mock('@/auth/web/composables/usePermission', () => ({
  usePermission: () => ({ hasAction: () => true }),
}));
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

describe('PartnerListView', () => {
  it('exposes list selection API and refresh helper', async () => {
    const PartnerListView = (await import('./PartnerListView.vue')).default;
    const wrapper = mount(PartnerListView, {
      props: { store: {} as any },
      global: { plugins: [i18n] },
    });
    expect(typeof (wrapper.vm as any).refresh).toBe('function');
    expect((wrapper.vm as any).selectedItems).toBeDefined();
  });
});
