// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { defineComponent, h, ref } from 'vue';
import ChoyPagination from './ChoyPagination.vue';

describe('ChoyPagination', () => {
  test('changing pageSize via select resets page to 1', async () => {
    const page = ref(3);
    const pageSize = ref(20);
    const Host = defineComponent({
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
    // Parent-driven pageSize change resets page through the model watch.
    pageSize.value = 10;
    await flushPromises();
    expect(page.value).toBe(1);
    expect(pageSize.value).toBe(10);
    page.value = 2;
    await flushPromises();
    const select = mounted.q('select') as HTMLSelectElement | null;
    expect(select).not.toBeNull();
    select!.value = '50';
    select!.dispatchEvent(new Event('change', { bubbles: true }));
    await flushPromises();
    expect(page.value).toBe(1);
    expect(pageSize.value).toBe(50);
    mounted.unmount();
  });

  test('includes current pageSize when missing from options', async () => {
    const page = ref(1);
    const pageSize = ref(25);
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyPagination as any, {
            total: 100,
            page: page.value,
            pageSize: pageSize.value,
            pageSizeOptions: [10, 20, 50],
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
    const select = mounted.q('select') as HTMLSelectElement | null;
    expect(select).not.toBeNull();
    // QJS select.value may not mirror :value; assert the option list instead.
    const values = Array.from(select!.options).map((o) => String(o.value));
    expect(values).toContain('25');
    expect(values.join(',')).toBe('10,20,25,50');
    expect(pageSize.value).toBe(25);
    mounted.unmount();
  });

  test('ignores invalid pageSize change events', async () => {
    const page = ref(2);
    const pageSize = ref(20);
    const mounted = mountApp(ChoyPagination as any, {
      props: {
        total: 100,
        page: page.value,
        pageSize: pageSize.value,
      },
      on: {
        'onUpdate:page': (v: number) => {
          page.value = v;
        },
        'onUpdate:pageSize': (v: number) => {
          pageSize.value = v;
        },
      },
    });
    await flushPromises();
    const state = mounted.setupState() as any;
    expect(typeof state?.onPageSizeChange).toBe('function');
    state.onPageSizeChange({ target: { value: '0' } } as any);
    state.onPageSizeChange({ target: { value: 'NaN' } } as any);
    await flushPromises();
    expect(pageSize.value).toBe(20);
    expect(page.value).toBe(2);
    mounted.unmount();
  });

  test('prev/next no-op when disabled', async () => {
    const page = ref(3);
    const mounted = mountApp(ChoyPagination as any, {
      props: {
        total: 100,
        disabled: true,
        page: page.value,
        pageSize: 20,
      },
      on: {
        'onUpdate:page': (v: number) => {
          page.value = v;
        },
      },
    });
    await flushPromises();
    const state = mounted.setupState() as any;
    expect(typeof state?.goPrev).toBe('function');
    expect(typeof state?.goNext).toBe('function');
    state.goPrev();
    state.goNext();
    await flushPromises();
    expect(page.value).toBe(3);
    mounted.unmount();
  });
});
