// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { createTranslate } from '@/web/web/i18n';
import { shallowMount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { createPinia, setActivePinia } from 'pinia';
import { menus } from '../menu/menus';
import { routes } from '../router/routes';
import OFormView from '../components/view/OFormView.vue';

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    currentRoute: { value: { path: '/', params: {}, query: {}, meta: {} } },
  }),
  useRoute: () => ({ params: {}, query: {}, path: '/', meta: {} }),
}));

function source(relativePath: string): string {
  return readFileSync(resolve(__dirname, relativePath), 'utf8');
}

const fakeStore = {
  state: { queryState: {}, result: undefined, selection: [], planCache: new Map(), record: undefined },
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

describe('web shell _lt bindings', () => {
  it('pins TermReference titles on web menus and routes', () => {
    const homeTitle = createTranslate('web', { scope: 'web/menu/menus' })._lt('Home');
    const root = menus[0] as any;
    expect(root.title).toBe('Home');
    expect(root.titleText).toEqual(homeTitle);
    expect(routes.length).toBeGreaterThan(0);
    expect(source('../router/routes.ts')).toContain("const { _lt } = createTranslate('web'");
  });

  it('keeps OFormView details title on _lt', () => {
    const form = source('../components/view/OFormView.vue');
    expect(form).toContain("const { _t, _lt } = createTranslate('web'");
    expect(form).toContain("const detailsTitle = _lt('Details')");
    expect(form).not.toContain('_tRef');
  });

  it('mounts OFormView so detailsTitle _lt runs', () => {
    setActivePinia(createPinia());
    const wrapper = shallowMount(OFormView as any, {
      props: { store: fakeStore as any },
      global: {
        plugins: [i18n],
        stubs: {
          // Element Plus pieces used by the toolbar / form shell.
          'el-button': true,
          'el-icon': true,
          OPage: true,
          OBreadcrumb: true,
        },
        // v-loading overlay; Element Plus directive is not installed here.
        directives: {
          loading: {
            mounted() {},
            updated() {},
          },
        },
      },
    });
    expect(wrapper.exists()).toBe(true);
  });

  it('executes breadcrumbStore factory-default _lt titles', async () => {
    const expectedPage = createTranslate('web', { scope: 'web/stores/breadcrumbStore' })._lt('Page');
    const expectedDetails = createTranslate('web', { scope: 'web/stores/breadcrumbStore' })._lt('Details');
    const mod = await import('../stores/breadcrumbStore/index');
    expect(mod.useBreadcrumbStore).toBeTypeOf('function');
    expect(expectedPage.src).toBe('Page');
    expect(expectedDetails.src).toBe('Details');
    expect(source('../stores/breadcrumbStore/index.ts')).toContain("const { _lt } = createTranslate('web'");
    expect(source('../stores/breadcrumbStore/index.ts')).toContain("_lt('Page')");
    expect(source('../stores/breadcrumbStore/index.ts')).toContain("_lt('Details')");
  });
});
