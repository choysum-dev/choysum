// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick, onMounted } from 'vue';

import { flushPromises, mountApp, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import OSearchView from './OSearchView.vue';
import OSearch from '@/web/web/components/view/search/OSearch.vue';

const stubState = {
  /** Defaults OSearch would emit after its single UserFilter load. */
  mountDefaults: [] as any[],
};

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

function mountSearchView(props: Record<string, any>) {
  const emitted: Record<string, any[][]> = {};
  const track = (name: string) => (...args: any[]) => {
    (emitted[name] ||= []).push(args);
  };
  stubSfc(OSearch, OSearchStub);
  const m = mountApp(OSearchView as any, {
    reactiveProps: true,
    props,
    on: {
      onQueryUpdate: track('query-update'),
    },
  });
  return { ...m, emitted };
}

describe('OSearchView favorites defaults (single child load)', () => {
  beforeEach(() => {
    stubState.mountDefaults = [];
  });
  afterEach(() => {
    restoreSfc(OSearch);
  });

  test('waits for defaults-ready before first query-update and applies emitted IsDefault', async () => {
    stubState.mountDefaults = [
      { name: 'PrivateDef', query: { And: [['P', '=', 1]] }, selected: true },
      { name: 'Code', query: ['C', '=', 1], selected: false },
    ];
    const { unmount, q, emitted } = mountSearchView({
      store: makeStore(),
      defaultFilters: [{ name: 'Code', query: ['C', '=', 1], selected: true }],
    });
    await flushPromises();
    expect(emitted['query-update']?.length).toBe(1);
    expect(JSON.parse(q('.code-defaults')!.textContent!)[0]).toMatchObject({
      name: 'Code',
      selected: true,
    });
    const applied = JSON.parse(q('.applied')!.textContent!);
    expect(applied[0]).toMatchObject({ name: 'PrivateDef' });
    unmount();
  });

  test('supports initialEmit=false (no query-update) and refresh via defaults-ready', async () => {
    stubState.mountDefaults = [{ name: 'SharedOnly', query: ['S', '=', 1], selected: true }];
    const { unmount, q, click, emitted } = mountSearchView({
      store: makeStore(),
      initialEmit: false,
    });
    await flushPromises();
    expect(emitted['query-update']).toBeUndefined();
    expect(JSON.parse(q('.applied')!.textContent!)[0]).toMatchObject({ name: 'SharedOnly' });

    stubState.mountDefaults = [{ name: 'LaterPrivate', query: ['L', '=', 1], selected: true }];
    click('.emit-defaults-ready');
    await flushPromises();
    await nextTick();
    expect(emitted['query-update']).toBeUndefined();
    expect(JSON.parse(q('.applied')!.textContent!)[0]).toMatchObject({ name: 'LaterPrivate' });
    unmount();
  });

  test('falls back to code defaults when child emits empty favorites defaults', async () => {
    stubState.mountDefaults = [{ name: 'Code', query: ['C', '=', 1], selected: true }];
    const { unmount, q } = mountSearchView({
      store: makeStore(),
      defaultFilters: [{ name: 'Code', query: ['C', '=', 1], selected: true }],
    });
    await flushPromises();
    expect(JSON.parse(q('.applied')!.textContent!)[0]).toMatchObject({ name: 'Code' });
    unmount();
  });

  test('accepts singleton defaultFilters and queryState.defaultFilters', async () => {
    stubState.mountDefaults = [{ name: 'Solo', query: ['S', '=', 1], selected: true }];
    const first = mountSearchView({
      store: makeStore(),
      defaultFilters: { name: 'Solo', query: ['S', '=', 1], selected: true },
    });
    await flushPromises();
    expect(JSON.parse(first.q('.code-defaults')!.textContent!)[0]).toMatchObject({ name: 'Solo', selected: true });
    first.unmount();
    restoreSfc(OSearch);

    stubState.mountDefaults = [{ name: 'FromQs', query: ['Q', '=', 1], selected: true }];
    const qs = mountSearchView({
      store: makeStore({
        state: { queryState: { defaultFilters: [{ name: 'FromQs', query: ['Q', '=', 1], selected: true }] } },
      }),
    });
    await flushPromises();
    expect(JSON.parse(qs.q('.code-defaults')!.textContent!)[0].name).toBe('FromQs');
    qs.unmount();
  });

  test('emits first query-update only once across repeated defaults-ready', async () => {
    stubState.mountDefaults = [{ name: 'A', query: ['A', '=', 1], selected: true }];
    const { unmount, q, click, emitted } = mountSearchView({ store: makeStore() });
    await flushPromises();
    expect(emitted['query-update']?.length).toBe(1);

    stubState.mountDefaults = [{ name: 'B', query: ['B', '=', 1], selected: true }];
    click('.emit-defaults-ready');
    await flushPromises();
    expect(emitted['query-update']?.length).toBe(1);
    expect(JSON.parse(q('.applied')!.textContent!)[0].name).toBe('B');
    unmount();
  });

  test('emits only one query-update when defaults-ready races before nextTick settles', async () => {
    stubState.mountDefaults = [{ name: 'Race', query: ['R', '=', 1], selected: true }];
    const { unmount, click, emitted } = mountSearchView({ store: makeStore() });
    click('.emit-defaults-ready');
    click('.emit-defaults-ready');
    await flushPromises();
    await nextTick();
    expect(emitted['query-update']?.length).toBe(1);
    unmount();
  });

  test('forwards child query-update and covers keyword / non-array defaults branches', async () => {
    stubState.mountDefaults = [];
    const { unmount, q, click, emitted } = mountSearchView({
      store: makeStore({
        state: { queryState: { keyword: 'from-store' } },
      }),
      keyword: 'from-prop',
      initialEmit: false,
    });
    await flushPromises();
    expect(q('.keyword')!.textContent).toBe('from-prop');

    const before = emitted['query-update']?.length ?? 0;
    click('.emit-query-update');
    expect(emitted['query-update']?.length ?? 0).toBe(before + 1);
    expect(emitted['query-update']?.at(-1)?.[0]).toMatchObject({ keyword: 'from-child' });

    click('.emit-defaults-ready-nonarray');
    await flushPromises();
    await nextTick();
    expect(emitted['query-update']?.length ?? 0).toBe(before + 1);
    unmount();
  });

  test('reads store keyword when prop keyword is absent', async () => {
    stubState.mountDefaults = [];
    const { unmount, q } = mountSearchView({
      store: makeStore({
        state: { queryState: { keyword: 'qs-only' } },
      }),
      initialEmit: false,
    });
    await flushPromises();
    expect(q('.keyword')!.textContent).toBe('qs-only');
    unmount();
  });

  test('treats empty or non-string store keyword as absent', async () => {
    stubState.mountDefaults = [];
    const emptyKw = mountSearchView({
      store: makeStore({ state: { queryState: { keyword: '' } } }),
      initialEmit: false,
    });
    await flushPromises();
    expect(emptyKw.q('.keyword')!.textContent).toBe('');
    emptyKw.unmount();
    restoreSfc(OSearch);

    const numKw = mountSearchView({
      store: makeStore({ state: { queryState: { keyword: 12 as any } } }),
      initialEmit: false,
    });
    await flushPromises();
    expect(numKw.q('.keyword')!.textContent).toBe('');
    numKw.unmount();
  });

  test('ignores non-array queryState.defaultFilters', async () => {
    stubState.mountDefaults = [];
    const { unmount, q } = mountSearchView({
      store: makeStore({
        state: { queryState: { defaultFilters: { name: 'Bad' } as any } },
      }),
      initialEmit: false,
    });
    await flushPromises();
    expect(JSON.parse(q('.code-defaults')!.textContent!)).toEqual([]);
    unmount();
  });

  test('skips first-frame emit when initialEmit flips false during nextTick', async () => {
    stubState.mountDefaults = [{ name: 'Flip', query: ['F', '=', 1], selected: true }];
    const { unmount, props, emitted } = mountSearchView({
      store: makeStore(),
      initialEmit: true,
    });
    props.initialEmit = false;
    await flushPromises();
    await nextTick();
    expect(emitted['query-update']).toBeUndefined();
    unmount();
  });
});
