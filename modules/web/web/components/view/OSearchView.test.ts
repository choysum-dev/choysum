// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, onMounted } from 'vue';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({ _t: (msg: string) => msg, _lt: (msg: string) => msg }),
  };
});

import OSearchView from './OSearchView.vue';

const stubState = vi.hoisted(() => ({
  /** Defaults OSearch would emit after its single UserFilter load. */
  mountDefaults: [] as any[],
}));

const OSearchStub = defineComponent({
  name: 'OSearch',
  props: ['store', 'placeholder', 'currentKeyword', 'currentAppliedFilters', 'currentAppliedGroups', 'defaultFilters'],
  emits: ['query-update', 'defaults-ready'],
  setup(props, { emit }) {
    onMounted(() => {
      emit('defaults-ready', stubState.mountDefaults.slice());
    });
    return () =>
      h('div', { class: 'o-search-stub' }, [
        h('pre', { class: 'code-defaults' }, JSON.stringify(props.defaultFilters || [])),
        h('pre', { class: 'applied' }, JSON.stringify(props.currentAppliedFilters || [])),
        h('pre', { class: 'keyword' }, String(props.currentKeyword ?? '')),
        h(
          'button',
          {
            type: 'button',
            class: 'emit-defaults-ready',
            onClick: () => emit('defaults-ready', stubState.mountDefaults.slice()),
          },
          'defaults-ready'
        ),
        h(
          'button',
          {
            type: 'button',
            class: 'emit-defaults-ready-nonarray',
            onClick: () => emit('defaults-ready', { name: 'NotArray' } as any),
          },
          'defaults-ready-nonarray'
        ),
        h(
          'button',
          {
            type: 'button',
            class: 'emit-query-update',
            onClick: () =>
              emit('query-update', {
                keyword: 'from-child',
                appliedFilters: [],
                appliedGroups: [],
              }),
          },
          'query-update'
        ),
      ]);
  },
});

function makeStore(patch: Record<string, any> = {}) {
  return {
    application: 'demo',
    modelName: 'Widget',
    state: { queryState: {} },
    ...patch,
  } as any;
}

describe('OSearchView favorites defaults (single child load)', () => {
  beforeEach(() => {
    stubState.mountDefaults = [];
  });

  it('waits for defaults-ready before first query-update and applies emitted IsDefault', async () => {
    stubState.mountDefaults = [
      { name: 'PrivateDef', query: { And: [['P', '=', 1]] }, selected: true },
      { name: 'Code', query: ['C', '=', 1], selected: false },
    ];
    const wrapper = mount(OSearchView as any, {
      props: {
        store: makeStore(),
        defaultFilters: [{ name: 'Code', query: ['C', '=', 1], selected: true }],
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(wrapper.emitted('query-update')?.length).toBe(1);
    // Parent passes code-only defaults to OSearch (child owns IsDefault merge).
    expect(JSON.parse(wrapper.find('.code-defaults').text())[0]).toMatchObject({
      name: 'Code',
      selected: true,
    });
    const applied = JSON.parse(wrapper.find('.applied').text());
    expect(applied[0]).toMatchObject({ name: 'PrivateDef' });
  });

  it('supports initialEmit=false (no query-update) and refresh via defaults-ready', async () => {
    stubState.mountDefaults = [{ name: 'SharedOnly', query: ['S', '=', 1], selected: true }];
    const wrapper = mount(OSearchView as any, {
      props: { store: makeStore(), initialEmit: false },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(wrapper.emitted('query-update')).toBeUndefined();
    expect(JSON.parse(wrapper.find('.applied').text())[0]).toMatchObject({ name: 'SharedOnly' });

    stubState.mountDefaults = [{ name: 'LaterPrivate', query: ['L', '=', 1], selected: true }];
    await wrapper.find('.emit-defaults-ready').trigger('click');
    await flushPromises();
    await nextTick();
    expect(wrapper.emitted('query-update')).toBeUndefined();
    expect(JSON.parse(wrapper.find('.applied').text())[0]).toMatchObject({ name: 'LaterPrivate' });
  });

  it('falls back to code defaults when child emits empty favorites defaults', async () => {
    stubState.mountDefaults = [{ name: 'Code', query: ['C', '=', 1], selected: true }];
    const wrapper = mount(OSearchView as any, {
      props: {
        store: makeStore(),
        defaultFilters: [{ name: 'Code', query: ['C', '=', 1], selected: true }],
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(JSON.parse(wrapper.find('.applied').text())[0]).toMatchObject({ name: 'Code' });
  });

  it('accepts singleton defaultFilters and queryState.defaultFilters', async () => {
    stubState.mountDefaults = [{ name: 'Solo', query: ['S', '=', 1], selected: true }];
    const wrapper = mount(OSearchView as any, {
      props: {
        store: makeStore(),
        defaultFilters: { name: 'Solo', query: ['S', '=', 1], selected: true },
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(JSON.parse(wrapper.find('.code-defaults').text())[0]).toMatchObject({ name: 'Solo', selected: true });

    stubState.mountDefaults = [{ name: 'FromQs', query: ['Q', '=', 1], selected: true }];
    const qsWrapper = mount(OSearchView as any, {
      props: {
        store: makeStore({
          state: { queryState: { defaultFilters: [{ name: 'FromQs', query: ['Q', '=', 1], selected: true }] } },
        }),
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(JSON.parse(qsWrapper.find('.code-defaults').text())[0].name).toBe('FromQs');
  });

  it('emits first query-update only once across repeated defaults-ready', async () => {
    stubState.mountDefaults = [{ name: 'A', query: ['A', '=', 1], selected: true }];
    const wrapper = mount(OSearchView as any, {
      props: { store: makeStore() },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(wrapper.emitted('query-update')?.length).toBe(1);

    stubState.mountDefaults = [{ name: 'B', query: ['B', '=', 1], selected: true }];
    await wrapper.find('.emit-defaults-ready').trigger('click');
    await flushPromises();
    expect(wrapper.emitted('query-update')?.length).toBe(1);
    expect(JSON.parse(wrapper.find('.applied').text())[0].name).toBe('B');
  });

  it('emits only one query-update when defaults-ready races before nextTick settles', async () => {
    stubState.mountDefaults = [{ name: 'Race', query: ['R', '=', 1], selected: true }];
    const wrapper = mount(OSearchView as any, {
      props: { store: makeStore() },
      global: { stubs: { OSearch: OSearchStub } },
    });
    // Fire a second defaults-ready in the same turn as mount emit, before flush.
    void wrapper.find('.emit-defaults-ready').trigger('click');
    void wrapper.find('.emit-defaults-ready').trigger('click');
    await flushPromises();
    await nextTick();
    expect(wrapper.emitted('query-update')?.length).toBe(1);
  });

  it('forwards child query-update and covers keyword / non-array defaults branches', async () => {
    stubState.mountDefaults = [];
    const wrapper = mount(OSearchView as any, {
      props: {
        store: makeStore({
          state: { queryState: { keyword: 'from-store' } },
        }),
        keyword: 'from-prop',
        initialEmit: false,
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(wrapper.find('.keyword').text()).toBe('from-prop');

    const before = wrapper.emitted('query-update')?.length ?? 0;
    await wrapper.find('.emit-query-update').trigger('click');
    expect((wrapper.emitted('query-update')?.length ?? 0)).toBe(before + 1);
    expect(wrapper.emitted('query-update')?.at(-1)?.[0]).toMatchObject({ keyword: 'from-child' });

    await wrapper.find('.emit-defaults-ready-nonarray').trigger('click');
    await flushPromises();
    await nextTick();
    // Non-array payload becomes []; with initialEmit=false no extra query-update.
    expect(wrapper.emitted('query-update')?.length ?? 0).toBe(before + 1);
  });

  it('reads store keyword when prop keyword is absent', async () => {
    stubState.mountDefaults = [];
    const wrapper = mount(OSearchView as any, {
      props: {
        store: makeStore({
          state: { queryState: { keyword: 'qs-only' } },
        }),
        initialEmit: false,
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(wrapper.find('.keyword').text()).toBe('qs-only');
  });

  it('treats empty or non-string store keyword as absent', async () => {
    stubState.mountDefaults = [];
    const emptyKw = mount(OSearchView as any, {
      props: {
        store: makeStore({ state: { queryState: { keyword: '' } } }),
        initialEmit: false,
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(emptyKw.find('.keyword').text()).toBe('');

    const numKw = mount(OSearchView as any, {
      props: {
        store: makeStore({ state: { queryState: { keyword: 12 as any } } }),
        initialEmit: false,
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(numKw.find('.keyword').text()).toBe('');
  });

  it('ignores non-array queryState.defaultFilters', async () => {
    stubState.mountDefaults = [];
    const wrapper = mount(OSearchView as any, {
      props: {
        store: makeStore({
          state: { queryState: { defaultFilters: { name: 'Bad' } as any } },
        }),
        initialEmit: false,
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(JSON.parse(wrapper.find('.code-defaults').text())).toEqual([]);
  });

  it('skips first-frame emit when initialEmit flips false during nextTick', async () => {
    stubState.mountDefaults = [{ name: 'Flip', query: ['F', '=', 1], selected: true }];
    const wrapper = mount(OSearchView as any, {
      props: { store: makeStore(), initialEmit: true },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await wrapper.setProps({ initialEmit: false });
    await flushPromises();
    await nextTick();
    expect(wrapper.emitted('query-update')).toBeUndefined();
  });
});
