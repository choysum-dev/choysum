// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { config, shallowMount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';

config.global.renderStubDefaultSlot = true;

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ fullPath: '/meta/coverage', params: { id: 'id-1' }, path: '/meta/coverage', query: {} }),
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: vi.fn(() => fakeStore),
}));

vi.mock('@/web/web/stores/storeScopeManager', () => ({
  useScopeManager: () => ({ menuScopeManager: {} }),
}));

const fakeStore = {
  state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
  setContext: vi.fn(),
  getContext: vi.fn(),
  withContext: vi.fn(),
  fullModelName: 'meta.Coverage',
  storeId: 'coverage',
};

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
      fullPath: '/meta/coverage',
      params: { id: 'id-1' },
      path: '/meta/coverage',
      query: {},
    };
  },
};

const pageModules = import.meta.glob('./*.vue', { eager: true }) as Record<string, { default: object }>;

describe('meta page OPage store / title coverage', () => {
  it('discovers module pages', () => {
    expect(Object.keys(pageModules).length).toBeGreaterThanOrEqual(4);
  });

  for (const [path, mod] of Object.entries(pageModules)) {
    it(`mounts ${path}`, () => {
      const isDetail = path.includes('Detail');
      const wrapper = shallowMount(mod.default as any, {
        props: isDetail ? { recordId: 'id-1' } : {},
        global: {
          plugins: [i18n, routeGlobal],
          stubs: {
            OPage: {
              name: 'OPage',
              props: ['title', 'store'],
              template: '<div class="opage-stub" :data-title="title"><slot /></div>',
            },
          },
        },
      });
      expect(wrapper.exists()).toBe(true);
      const page = wrapper.findComponent({ name: 'OPage' });
      expect(page.exists()).toBe(true);
      expect(page.props('store')).toBeTruthy();
      if (!isDetail) {
        expect(page.props('title')).toBeTruthy();
      }
      wrapper.unmount();
    });
  }
});
