// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { config, shallowMount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';

config.global.renderStubDefaultSlot = true;

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ fullPath: '/auth/coverage', params: { id: 'id-1' }, path: '/auth/coverage', query: {} }),
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
  fullModelName: 'auth.Coverage',
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
      fullPath: '/auth/coverage',
      params: { id: 'id-1' },
      path: '/auth/coverage',
      query: {},
    };
  },
};

const SKIP = new Set(['./Login.vue', './Logout.vue', './Register.vue']);

const pageModules = Object.fromEntries(
  Object.entries(
    import.meta.glob('./*.vue', { eager: true }) as Record<string, { default: object }>
  ).filter(([path]) => !SKIP.has(path))
);

describe('auth page OPage store / title coverage', () => {
  it('discovers business pages', () => {
    expect(Object.keys(pageModules).length).toBeGreaterThanOrEqual(17);
  });

  for (const [path, mod] of Object.entries(pageModules)) {
    it(`mounts ${path}`, () => {
      const isList = /List|Kanban/.test(path);
      const wrapper = shallowMount(mod.default as any, {
        props: isList ? {} : { recordId: 'id-1', viewMode: 'display' },
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
              template: '<div class="opage-stub" :data-title="title"><slot /></div>',
            },
          },
        },
      });
      expect(wrapper.exists()).toBe(true);
      const page = wrapper.findComponent({ name: 'OPage' });
      expect(page.exists()).toBe(true);
      expect(page.props('store')).toBeTruthy();
      if (isList && !path.includes('RoleFieldRule') && !path.includes('RoleMethodAccess') && !path.includes('RoleRecordRule') && !path.includes('RoleUiResource')) {
        // Title + import/export pages evaluate createTranslate / upload-hint _t.
        if (path.includes('List') || path.includes('Kanban')) {
          expect(page.props('title')).toBeTruthy();
        }
      }
      wrapper.unmount();
    });
  }
});
