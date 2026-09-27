// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, defineComponent } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyPage from '../layout/ChoyPage.vue';
import ChoyFormView from './ChoyFormView.vue';
import ChoyListView from './ChoyListView.vue';
import { useOPageContext } from '@/web/web/composables/usePageContext';
import OFormView from './OFormView.vue';

const fakeStore = {
  modelName: 'auth.User',
  meta: { fields: {} },
} as any;

describe('Choy store engine', () => {
  test('ChoyPage provides store to descendants via page context', async () => {
    let seen: unknown = undefined;
    const Probe = defineComponent({
      setup() {
        const ctx = useOPageContext();
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

  test('ChoyFormView store mode hosts OFormView root class', async () => {
    stubSfc(OFormView, {
      props: { store: null },
      setup: ((_props: any, { slots }: any) => {
        return () =>
          h('div', { class: 'form-view__content', 'data-test': 'o-form-host' }, slots.default?.());
      }) as any,
    });
    try {
      const Host = defineComponent({
        setup() {
          return () => h(ChoyFormView, { store: fakeStore }, () => h('span', 'field'));
        },
      });
      const wrapper = mountApp(Host);
      await flushPromises();
      expect(wrapper.q('[data-test=o-form-host]')).not.toBeNull();
      expect(wrapper.q('.choy-form-view')).toBeNull();
      wrapper.unmount();
    } finally {
      restoreSfc(OFormView);
    }
  });

  test('ChoyFormView chrome mode keeps choy-form-view without store', async () => {
    const Host = defineComponent({
      setup() {
        return () => h(ChoyFormView, { title: 'Chrome' }, () => h('span', 'body'));
      },
    });
    const wrapper = mountApp(Host);
    await flushPromises();
    expect(wrapper.q('.choy-form-view')).not.toBeNull();
    expect(wrapper.q('.form-view__content')).toBeNull();
    wrapper.unmount();
  });

  test('ChoyListView chrome mode still renders DataTable without store', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyListView, {
            columns: [{ accessorKey: 'name', header: 'Name' }],
            data: [{ name: 'a' }],
          } as any);
      },
    });
    const wrapper = mountApp(Host);
    await flushPromises();
    expect(wrapper.q('[data-anchor="choy.list-view"]')).not.toBeNull();
    wrapper.unmount();
  });
});
