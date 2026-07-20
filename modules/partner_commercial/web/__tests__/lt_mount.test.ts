// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { config, shallowMount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';

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

const viewModules = import.meta.glob('../views/*View.vue', { eager: true }) as Record<
  string,
  { default: object }
>;

describe('partner_commercial view _lt setup coverage', () => {
  for (const [path, mod] of Object.entries(viewModules)) {
    it(`mounts ${path} so createTranslate/_lt run`, () => {
      const wrapper = shallowMount(mod.default as any, {
        props: { store: fakeStore as any },
        global: {
          plugins: [i18n],
          stubs: true,
          // Options-API extensions return `_t` / `_lt` for the template; Vue warns on `_` prefixes.
          config: {
            warnHandler(msg) {
              if (/["_']_(?:t|lt)["_'].*should not start with/i.test(msg)) return;
              console.warn(`[Vue warn]: ${msg}`);
            },
          },
        },
      });
      expect(wrapper.exists()).toBe(true);
    });
  }
});
