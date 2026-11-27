// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, KeepAlive, nextTick, ref } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useDetailPathCollectionSoftRefresh } from './useDetailPathCollectionSoftRefresh';

describe('useDetailPathCollectionSoftRefresh', () => {
  test('soft-refreshes on KeepAlive reactivate when detailPath is set', async () => {
    const calls: number[] = [];
    const showCached = ref(true);
    const CachedPage = defineComponent({
      name: 'SoftRefreshPage',
      setup() {
        useDetailPathCollectionSoftRefresh({
          detailPath: '/sandbox/records/:id',
          hasSettled: () => true,
          refresh: () => {
            calls.push(1);
          },
        });
        return () => h('div', { 'data-test': 'cached' }, 'list');
      },
    });
    const Host = defineComponent({
      name: 'SoftRefreshHost',
      setup() {
        return () =>
          h(KeepAlive, null, {
            default: () =>
              showCached.value ? h(CachedPage) : h('div', { 'data-test': 'form' }, 'form'),
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    await nextTick();
    expect(calls.length).toBe(0);

    showCached.value = false;
    await nextTick();
    showCached.value = true;
    await flushPromises();
    await nextTick();
    await nextTick();
    expect(calls.length).toBe(1);

    mounted.unmount();
  });

  test('skips soft-refresh when detailPath is null', async () => {
    const calls: number[] = [];
    const showCached = ref(true);
    const CachedPage = defineComponent({
      name: 'NoDetailPathPage',
      setup() {
        useDetailPathCollectionSoftRefresh({
          detailPath: null,
          hasSettled: () => true,
          refresh: () => {
            calls.push(1);
          },
        });
        return () => h('div', { 'data-test': 'cached' }, 'picker');
      },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(KeepAlive, null, {
            default: () =>
              showCached.value ? h(CachedPage) : h('div', { 'data-test': 'other' }, 'other'),
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    showCached.value = false;
    await nextTick();
    showCached.value = true;
    await flushPromises();
    await nextTick();
    expect(calls.length).toBe(0);
    mounted.unmount();
  });

  test('defers soft-refresh until collection settles after reactivate', async () => {
    const calls: number[] = [];
    const showCached = ref(true);
    const settled = ref(false);
    const CachedPage = defineComponent({
      name: 'UnsettlePage',
      setup() {
        useDetailPathCollectionSoftRefresh({
          detailPath: '/sandbox/records/:id',
          hasSettled: () => settled.value,
          refresh: () => {
            calls.push(1);
          },
        });
        return () => h('div', { 'data-test': 'cached' }, 'list');
      },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(KeepAlive, null, {
            default: () =>
              showCached.value ? h(CachedPage) : h('div', { 'data-test': 'form' }, 'form'),
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    showCached.value = false;
    await nextTick();
    showCached.value = true;
    await flushPromises();
    await nextTick();
    expect(calls.length).toBe(0);

    settled.value = true;
    await flushPromises();
    await nextTick();
    expect(calls.length).toBe(1);
    mounted.unmount();
  });
});
