// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { config, shallowMount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';

config.global.renderStubDefaultSlot = true;

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ fullPath: '/base/coverage', params: { id: 'id-1' }, path: '/base/coverage', query: {} }),
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
  fullModelName: 'base.Coverage',
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
      fullPath: '/base/coverage',
      params: { id: 'id-1' },
      path: '/base/coverage',
      query: {},
    };
  },
};

const pageModules = import.meta.glob('./*.vue', { eager: true }) as Record<string, { default: object }>;

describe('base page OPage store / title coverage', () => {
  it('discovers list and form pages', () => {
    expect(Object.keys(pageModules).length).toBeGreaterThanOrEqual(20);
  });

  for (const [path, mod] of Object.entries(pageModules)) {
    it(`mounts ${path}`, () => {
      const isList = path.includes('List');
      const wrapper = shallowMount(mod.default as any, {
        props: {},
        global: {
          plugins: [i18n, routeGlobal],
          stubs: {
            OPage: {
              name: 'OPage',
              props: [
                'title',
                'store',
                'actionImport',
                'actionExport',
                'actionImportUploadHint',
                'actionImportColumnMapping',
                'actionListRef',
                'actionCompanyId',
              ],
              template:
                '<div class="opage-stub" :data-title="title" :data-import="String(!!actionImport)"><slot /></div>',
            },
          },
        },
      });
      expect(wrapper.exists()).toBe(true);
      const page = wrapper.findComponent({ name: 'OPage' });
      expect(page.exists()).toBe(true);
      expect(page.props('store')).toBeTruthy();
      if (isList) {
        expect(page.props('title')).toBeTruthy();
        // Boolean attrs compile to '' on stubs; presence (not undefined) is enough.
        expect(page.props('actionImport')).toBeDefined();
        expect(page.props('actionExport')).toBeDefined();
        expect(String(page.props('actionImportUploadHint') || '')).toMatch(/UTF-8 CSV/);
      }
      wrapper.unmount();
    });
  }
});
