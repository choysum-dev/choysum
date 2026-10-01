// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { defineComponent, h, ref } from 'vue';
import ListPagination from './ListPagination.vue';

describe('ListPagination', () => {
  test('maps page/pageSize changes to paginateState limit/offset', async () => {
    const events: Array<{ limit: number; offset: number }> = [];
    const mounted = mountApp(ListPagination as any, {
      props: {
        store: {},
        total: 100,
        limit: 20,
        offset: 20,
      },
      on: {
        onPaginateState: (payload: { limit: number; offset: number }) => {
          events.push(payload);
        },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-pagination]')).not.toBeNull();
    const state = mounted.setupState() as any;
    expect(state?.pageModel).toBe(2);
    expect(state?.pageSizeModel).toBe(20);
    state.pageModel = 1;
    await flushPromises();
    expect(events.some((e) => e.limit === 20 && e.offset === 0)).toBe(true);
    state.pageSizeModel = '50';
    await flushPromises();
    expect(events.some((e) => e.limit === 50 && e.offset === 0)).toBe(true);
    mounted.unmount();
  });

  test('defers mount while total is unknown and offset is restored', async () => {
    const events: Array<{ limit: number; offset: number }> = [];
    const total = ref(0);
    const Host = defineComponent({
      setup() {
        return () =>
          h(ListPagination as any, {
            store: {},
            total: total.value,
            limit: 20,
            offset: 40,
            onPaginateState: (payload: { limit: number; offset: number }) => {
              events.push(payload);
            },
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.q('[data-testid=choy-pagination]')).toBeNull();
    expect(events.length).toBe(0);
    total.value = 100;
    await flushPromises();
    expect(mounted.q('[data-testid=choy-pagination]')).not.toBeNull();
    expect(events.some((e) => e.offset === 0 && e.limit === 20)).toBe(false);
    mounted.unmount();
  });
});
