// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, createApp } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises } from '@/web/web/__tests__/mountApp';
import App from './App.vue';
import { useI18nStore } from './stores';

describe('App', () => {
  test('wraps router-view in Element Plus config provider for dual-stack locale', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useI18nStore().localeCode = 'en';

    const errors: string[] = [];
    const el = document.createElement('div');
    document.body.appendChild(el);
    const app = createApp(App as any);
    app.use(pinia);
    app.component('el-config-provider', {
      props: { locale: null, size: String },
      setup(props: any, { slots }: any) {
        return () =>
          h(
            'div',
            { 'data-test': 'ep-config', 'data-size': String(props.size || '') },
            slots.default?.(),
          );
      },
    });
    app.component('router-view', {
      setup() {
        return () => h('div', { 'data-test': 'router-view' });
      },
    });
    app.config.errorHandler = (err: unknown) => {
      errors.push(String(err));
    };
    app.mount(el);
    await flushPromises();

    expect(errors).toEqual([]);
    expect(el.querySelector('.choy-app')).not.toBeNull();
    expect(el.querySelector('[data-test=ep-config]')?.getAttribute('data-size')).toBe('default');
    expect(el.querySelector('[data-test=router-view]')).not.toBeNull();
    app.unmount();
    el.remove();
  });
});
