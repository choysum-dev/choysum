// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, createApp } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import App from './App.vue';
import ChoyConfirmHost from './components/layout/ChoyConfirmHost.vue';
import Toaster from './components/vendor/ui/sonner/Sonner.vue';
import { useI18nStore } from './stores';

describe('App', () => {
  test('mounts router-view with toaster and confirm host', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useI18nStore().localeCode = 'en';

    stubSfc(Toaster as any, {
      name: 'Toaster',
      setup() {
        return () => h('div', { 'data-test': 'toaster' });
      },
    });
    stubSfc(ChoyConfirmHost as any, {
      name: 'ChoyConfirmHost',
      setup() {
        return () => h('div', { 'data-test': 'confirm-host' });
      },
    });

    const errors: string[] = [];
    const el = document.createElement('div');
    document.body.appendChild(el);
    const app = createApp(App as any);
    app.use(pinia);
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
    expect(el.querySelector('[data-test=router-view]')).not.toBeNull();
    expect(el.querySelector('[data-test=toaster]')).not.toBeNull();
    expect(el.querySelector('[data-test=confirm-host]')).not.toBeNull();
    app.unmount();
    el.remove();
    restoreSfc(Toaster as any);
    restoreSfc(ChoyConfirmHost as any);
  });
});
