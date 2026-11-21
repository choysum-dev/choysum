// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick, type Ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { createMenuPlugin } from '@/core/web/menu';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import {
  clearCompanyScopeView,
  publishCompanyScopeView,
} from '../stores/companyScopeView';
import { useMenuStore } from '../stores/menuStore';
import { useKeepAliveScopeKey } from './useKeepAliveScopeKey';

test('useKeepAliveScopeKey: sticky menu id ignores null flaps', async () => {
  clearCompanyScopeView();
  publishCompanyScopeView('co1', ['co1']);

  const menuPlugin = createMenuPlugin();
  menuPlugin.manager.addMenu({
    id: 'app',
    title: 'App',
    children: [
      { id: 'leaf.a', title: 'A', path: '/a' },
      { id: 'leaf.b', title: 'B', path: '/b' },
    ],
  } as any);

  const pinia = createPinia();
  setActivePinia(pinia);

  let keyRef: Ref<string> | null = null;
  let menuStore: ReturnType<typeof useMenuStore> | null = null;
  const Host = defineComponent({
    name: 'KeepAliveScopeHost',
    setup() {
      keyRef = useKeepAliveScopeKey();
      menuStore = useMenuStore();
      return () => h('div', keyRef!.value);
    },
  });
  const mounted = mountApp(Host as any, { plugins: [menuPlugin, pinia] });
  await flushPromises();

  const leafA = menuStore!.getMenu('leaf.a');
  expect(leafA).toBeTruthy();
  menuStore!.setActiveMenu(leafA!);
  await nextTick();
  expect(keyRef!.value).toBe('co1|co1::leaf.a');

  menuStore!.setActiveMenu(null);
  await nextTick();
  expect(keyRef!.value).toBe('co1|co1::leaf.a');

  const leafB = menuStore!.getMenu('leaf.b');
  expect(leafB).toBeTruthy();
  menuStore!.setActiveMenu(leafB!);
  await nextTick();
  expect(keyRef!.value).toBe('co1|co1::leaf.b');

  publishCompanyScopeView('co2', ['co2']);
  await nextTick();
  expect(keyRef!.value).toBe('co2|co2::leaf.b');

  clearCompanyScopeView();
  mounted.unmount();
});
