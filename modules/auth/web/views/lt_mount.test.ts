// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { config, shallowMount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';

// Stubbed OFormView/OListView must still render default slots so template
// field lines (the P6 :label removals) are exercised for coverage.
config.global.renderStubDefaultSlot = true;

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ params: {}, query: {}, path: '/' }),
}));

vi.mock('@/auth/web/composables/usePermission', () => ({
  usePermission: () => ({
    canRoute: () => true,
    hasAction: () => true,
  }),
}));

const fakeStore = {
  state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
  setContext: vi.fn(),
  getContext: vi.fn(),
  withContext: vi.fn(),
};

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

const viewModules = import.meta.glob('./*View.vue', { eager: true }) as Record<
  string,
  { default: object }
>;

describe('auth view _lt setup coverage', () => {
  for (const [path, mod] of Object.entries(viewModules)) {
    it(`mounts ${path} so createTranslate/_lt run`, () => {
      const wrapper = shallowMount(mod.default as any, {
        props: { store: fakeStore as any },
        global: {
          plugins: [i18n],
          stubs: true,
          // Views may use v-action; coverage mounts do not install the real directive.
          directives: {
            action: {
              mounted() {},
              updated() {},
            },
          },
        },
      });
      expect(wrapper.exists()).toBe(true);
    });
  }
});
