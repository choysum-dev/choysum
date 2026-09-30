// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { defineComponent, h, ref } from 'vue';
import ChoyPagination from './ChoyPagination.vue';

describe('ChoyPagination', () => {
  test('changing pageSize resets page to 1', async () => {
    const page = ref(3);
    const pageSize = ref(20);
    const Host = defineComponent({
      name: 'PaginationHost',
      setup() {
        return () =>
          h(ChoyPagination as any, {
            total: 100,
            page: page.value,
            pageSize: pageSize.value,
            'onUpdate:page': (v: number) => {
              page.value = v;
            },
            'onUpdate:pageSize': (v: number) => {
              pageSize.value = v;
            },
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(page.value).toBe(3);
    pageSize.value = 50;
    // Remount so the child watch sees the new pageSize via props/model update path.
    mounted.unmount();
    const mounted2 = mountApp(
      defineComponent({
        setup() {
          return () =>
            h(ChoyPagination as any, {
              total: 100,
              page: page.value,
              pageSize: pageSize.value,
              'onUpdate:page': (v: number) => {
                page.value = v;
              },
              'onUpdate:pageSize': (v: number) => {
                pageSize.value = v;
              },
            });
        },
      }) as any,
    );
    await flushPromises();
    // Trigger size change through the select so the component's watch runs.
    const select = mounted2.q('select') as HTMLSelectElement | null;
    expect(select).not.toBeNull();
    select!.value = '10';
    select!.dispatchEvent(new Event('change', { bubbles: true }));
    await flushPromises();
    expect(page.value).toBe(1);
    expect(pageSize.value).toBe(10);
    mounted2.unmount();
  });
});
