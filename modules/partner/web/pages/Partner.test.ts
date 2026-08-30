// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { config, shallowMount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import PartnerPage from './Partner.vue';

config.global.renderStubDefaultSlot = true;

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ fullPath: '/partner/partners/p1', params: { id: 'p1' }, path: '/partner/partners/p1', query: {} }),
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: vi.fn(() => ({
    storeId: 'Partner_/partner/partners/p1',
    fullModelName: 'partner.Partner',
    state: { queryState: {}, result: undefined },
  })),
}));

vi.mock('@/web/web/stores/storeScopeManager', () => ({
  useScopeManager: () => ({ menuScopeManager: {} }),
}));

vi.mock('@/auth/web/stores/auth', () => ({
  useAuthStore: () => ({
    identity: {
      metadata: {
        activeCompanyId: 'cmp-1',
        enabledCompanyIds: ['cmp-1'],
      },
    },
  }),
}));

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

const routeGlobal = {
  install(app: { config: { globalProperties: Record<string, unknown> } }) {
    app.config.globalProperties.$route = {
      fullPath: '/partner/partners/p1',
      params: { id: 'p1' },
      path: '/partner/partners/p1',
      query: {},
    };
  },
};

describe('Partner page OPage store coverage', () => {
  it('mounts Partner form page with store on OPage', () => {
    const wrapper = shallowMount(PartnerPage as any, {
      props: { recordId: 'p1', viewMode: 'display' },
      global: {
        plugins: [i18n, routeGlobal],
        stubs: {
          OPage: {
            name: 'OPage',
            props: ['store'],
            template: '<div class="opage-stub"><slot /></div>',
          },
          PartnerFormView: true,
        },
      },
    });
    expect(wrapper.exists()).toBe(true);
    const page = wrapper.findComponent({ name: 'OPage' });
    expect(page.exists()).toBe(true);
    expect(page.props('store')).toBeTruthy();
    wrapper.unmount();
  });
});
