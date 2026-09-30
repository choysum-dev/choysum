// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
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
    // Attr fallthrough may replace ChoyPagination's data-testid with list-pagination.
    const el =
      mounted.q('[data-testid=list-pagination]') || mounted.q('[data-testid=choy-pagination]');
    expect(el).not.toBeNull();
    const state = mounted.setupState() as any;
    expect(state?.pageModel).toBe(2);
    expect(state?.pageSizeModel).toBe(20);
    state.pageModel = 1;
    await flushPromises();
    expect(events.some((e) => e.limit === 20 && e.offset === 0)).toBe(true);
    state.pageSizeModel = 50;
    await flushPromises();
    expect(events.some((e) => e.limit === 50 && e.offset === 0)).toBe(true);
    mounted.unmount();
  });
});
