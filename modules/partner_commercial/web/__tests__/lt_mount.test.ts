// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { shallowMount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import PartnerFormView from '../views/PartnerFormView.vue';

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

describe('partner_commercial view _lt setup coverage', () => {
  it('mounts PartnerFormView extension', () => {
    const wrapper = shallowMount(PartnerFormView, {
      props: { store: fakeStore as any },
      global: {
        plugins: [i18n],
        stubs: true,
        // Options-API extensions return `_t` / `_lt` for the template; Vue warns on `_` prefixes.
        config: {
          warnHandler(msg) {
            // createTranslate helpers are intentionally named `_t` / `_lt`.
            if (/["_']_(?:t|lt)["_'].*should not start with/i.test(msg)) return;
            console.warn(`[Vue warn]: ${msg}`);
          },
        },
      },
    });
    expect(wrapper.exists()).toBe(true);
  });
});
