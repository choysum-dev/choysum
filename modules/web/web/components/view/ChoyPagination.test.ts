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

  test('clamps page when it exceeds totalPages', async () => {
    const page = ref(9);
    const pageSize = ref(20);
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyPagination as any, {
            total: 40,
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
    // total 40 / size 20 → 2 pages; page 9 clamps to 2.
    expect(page.value).toBe(2);
    mounted.unmount();
  });

  test('startEdit focuses and selects the range input when available', async () => {
    const mounted = mountApp(ChoyPagination as any, {
      props: { total: 50, page: 1, pageSize: 10 },
    });
    await flushPromises();
    const state = mounted.setupState() as any;
    await state.startEdit();
    await flushPromises();
    const input = mounted.q('.choy-pagination__range-input') as any;
    expect(input).not.toBeNull();
    let focused = false;
    let selected = false;
    input.focus = () => {
      focused = true;
    };
    input.select = () => {
      selected = true;
    };
    // Re-enter edit so startEdit's nextTick path calls the stubs.
    await state.startEdit();
    await flushPromises();
    expect(focused).toBe(true);
    expect(selected).toBe(true);
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

  test('normalizedTotal and effectivePageSize handle NaN and fractional values', async () => {
    const page = ref(1);
    const pageSize = ref(10.7);
    const mounted = mountApp(ChoyPagination as any, {
      props: {
        total: Number.NaN,
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
    expect(mounted.q('.choy-pagination__total')?.textContent?.trim()).toBe('0');
    // formatChoyPagerRangeValue collapses empty ranges to a single "0".
    expect(mounted.q('.choy-pagination__range-value')?.textContent?.trim()).toBe('0');
    mounted.unmount();

    const page2 = ref(2);
    const pageSize2 = ref(10.7);
    const mounted2 = mountApp(ChoyPagination as any, {
      props: {
        total: 25.9,
        page: page2.value,
        pageSize: pageSize2.value,
      },
      on: {
        'onUpdate:page': (v: number) => {
          page2.value = v;
        },
        'onUpdate:pageSize': (v: number) => {
          pageSize2.value = v;
        },
      },
    });
    await flushPromises();
    // floor(total)=25, floor(pageSize)=10 → page 2 is 11-20
    expect(mounted2.q('.choy-pagination__total')?.textContent?.trim()).toBe('25');
    expect(mounted2.q('.choy-pagination__range-value')?.textContent?.trim()).toBe('11-20');
    mounted2.unmount();

    // Invalid pageSize falls back to 20 (same as applyChoyPagerRangeEdit).
    const mounted3 = mountApp(ChoyPagination as any, {
      props: { total: 40, page: 1, pageSize: 0 },
    });
    await flushPromises();
    expect(mounted3.q('.choy-pagination__range-value')?.textContent?.trim()).toBe('1-20');
    mounted3.unmount();
  });

  test('keyboard Enter commits, Escape cancels, invalid draft no-ops', async () => {
    const page = ref(1);
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
    await state.startEditWith('41');
    expect(mounted.q('.choy-pagination__range-input')).not.toBeNull();
    await state.commitEdit();
    await flushPromises();
    expect(page.value).toBe(3);

    await state.startEditWith('nope');
    state.cancelEdit();
    await flushPromises();
    expect(mounted.q('.choy-pagination__range-input')).toBeNull();
    expect(pageSize.value).toBe(20);

    await state.startEditWith('55');
    state.onEditKeydown?.({ key: 'Enter', preventDefault() {} });
    await flushPromises();
    expect(page.value).toBe(3);

    await state.startEditWith('??');
    state.onEditKeydown?.({ key: 'Escape', preventDefault() {} });
    await flushPromises();
    expect(mounted.q('.choy-pagination__range-input')).toBeNull();

    await state.startEditWith('???');
    await state.commitEdit();
    await flushPromises();
    expect(pageSize.value).toBe(20);

    // Cover the @input handler (including a null target → empty draft).
    await state.startEditWith('1');
    expect(typeof state.onEditInput).toBe('function');
    state.onEditInput({ target: { value: '31' } });
    state.onEditInput({ target: null });
    state.cancelEdit();
    mounted.unmount();
  });

  test('startEdit is a no-op when disabled or total is empty', async () => {
    const mounted = mountApp(ChoyPagination as any, {
      props: { total: 0, page: 1, pageSize: 20, disabled: false },
    });
    await flushPromises();
    const state = mounted.setupState() as any;
    await state.startEdit();
    expect(mounted.q('.choy-pagination__range-input')).toBeNull();
    mounted.unmount();

    const mounted2 = mountApp(ChoyPagination as any, {
      props: { total: 10, page: 1, pageSize: 20, disabled: true },
    });
    await flushPromises();
    const state2 = mounted2.setupState() as any;
    await state2.startEdit();
    expect(mounted2.q('.choy-pagination__range-input')).toBeNull();
    mounted2.unmount();

    // NaN total → normalizedTotal 0; range edit stays disabled.
    const mounted3 = mountApp(ChoyPagination as any, {
      props: { total: Number.NaN, page: 1, pageSize: 20, disabled: false },
    });
    await flushPromises();
    const rangeBtn = mounted3.q('.choy-pagination__range-value') as HTMLButtonElement | null;
    expect(rangeBtn?.disabled).toBe(true);
    const state3 = mounted3.setupState() as any;
    await state3.startEdit();
    expect(mounted3.q('.choy-pagination__range-input')).toBeNull();
    mounted3.unmount();
  });

  test('prev and next advance when enabled', async () => {
    const page = ref(2);
    const pageSize = ref(10);
    let pager: { goNext: () => void; goPrev: () => void } | null = null;
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyPagination as any, {
            ref: (el: any) => {
              pager = el;
            },
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
    expect(pager).not.toBeNull();
    pager!.goNext();
    await flushPromises();
    expect(page.value).toBe(3);
    pager!.goPrev();
    await flushPromises();
    expect(page.value).toBe(2);
    mounted.unmount();
  });
});
