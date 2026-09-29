// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, defineComponent } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyPage from '../layout/ChoyPage.vue';
import { usePageContext } from '@/web/web/composables/usePageContext';

const fakeStore = {
  modelName: 'auth.User',
  meta: { fields: {} },
} as any;

describe('Choy page store context', () => {
  test('ChoyPage provides store to descendants via page context', async () => {
    let seen: unknown = undefined;
    const Probe = defineComponent({
      setup() {
        const ctx = usePageContext();
        seen = ctx?.store.value;
        return () => h('div', { 'data-test': 'probe' });
      },
    });

    const Host = defineComponent({
      setup() {
        return () => h(ChoyPage, { store: fakeStore, title: 'Engine' }, () => h(Probe));
      },
    });
    const wrapper = mountApp(Host);
    await flushPromises();
    expect(wrapper.q('[data-test=probe]')).not.toBeNull();
    expect(seen).toBe(fakeStore);
    wrapper.unmount();
  });

  test('nested ChoyPage without store keeps ancestor page store', async () => {
    let seen: unknown = undefined;
    const Probe = defineComponent({
      setup() {
        const ctx = usePageContext();
        seen = ctx?.store.value;
        return () => h('div', { 'data-test': 'nested-probe' });
      },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyPage, { store: fakeStore, title: 'Outer' }, () =>
            h(ChoyPage, { title: 'Inner' }, () => h(Probe)),
          );
      },
    });
    const wrapper = mountApp(Host);
    await flushPromises();
    expect(wrapper.q('[data-test=nested-probe]')).not.toBeNull();
    expect(seen).toBe(fakeStore);
    wrapper.unmount();
  });
});
