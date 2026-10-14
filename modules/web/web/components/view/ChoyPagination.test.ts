// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { defineComponent, h, ref } from 'vue';
import ChoyPagination from './ChoyPagination.vue';

describe('ChoyPagination', () => {
  test('renders editable range without rows-per-page select', async () => {
    const page = ref(1);
    const pageSize = ref(20);
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyPagination as any, {
            total: 12,
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
    expect(mounted.q('select')).toBeNull();
    const rangeBtn = mounted.q('.choy-pagination__range-value') as HTMLButtonElement | null;
    expect(rangeBtn).not.toBeNull();
    expect(rangeBtn!.textContent?.trim()).toBe('1-12');
    expect(mounted.q('.choy-pagination__total')?.textContent?.trim()).toBe('12');
    mounted.unmount();
  });

  test('editing range jumps page by start index', async () => {
    const page = ref(1);
    const pageSize = ref(10);
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
    await state.startEditWith('21');
    await state.commitEdit();
    await flushPromises();
    expect(page.value).toBe(3);
    expect(pageSize.value).toBe(10);
    mounted.unmount();
  });


  test('unchanged range draft does not rewrite pageSize', async () => {
    const page = ref(6);
    const pageSize = ref(20);
    const mounted = mountApp(ChoyPagination as any, {
      props: {
        total: 105,
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
    await state.startEdit();
    await state.commitEdit();
    await flushPromises();
    expect(pageSize.value).toBe(20);
    expect(page.value).toBe(6);
    mounted.unmount();
  });

  test('editing min-max updates pageSize window and page', async () => {
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
    await state.startEditWith('1-50');
    await state.commitEdit();
    await flushPromises();
    expect(pageSize.value).toBe(50);
    expect(page.value).toBe(1);
    mounted.unmount();
  });

  test('parent pageSize change still resets page to 1', async () => {
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
    pageSize.value = 10;
    await flushPromises();
    expect(page.value).toBe(1);
    expect(pageSize.value).toBe(10);
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
