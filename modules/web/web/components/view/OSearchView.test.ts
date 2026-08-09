// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

const { sfSearch, actorState } = vi.hoisted(() => ({
  sfSearch: vi.fn(async () => [] as any[]),
  actorState: { id: 'me' as string },
}));

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({ _t: (msg: string) => msg }),
  };
});

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: (model: string) => {
    if (model === 'web.SavedFilter') return { Search: (...args: any[]) => sfSearch(...args) };
    return {};
  },
}));

vi.mock('@/web/web/composables/search/actorUserId', () => ({
  actorUserId: () => actorState.id,
}));

import OSearchView from './OSearchView.vue';

const OSearchStub = defineComponent({
  name: 'OSearch',
  props: ['store', 'placeholder', 'currentKeyword', 'currentAppliedFilters', 'currentAppliedGroups', 'defaultFilters'],
  emits: ['query-update', 'defaults-ready'],
  setup(props, { emit }) {
    return () =>
      h('div', { class: 'o-search-stub' }, [
        h('pre', { class: 'defaults' }, JSON.stringify(props.defaultFilters || [])),
        h(
          'button',
          {
            type: 'button',
            class: 'emit-defaults-ready',
            onClick: () => emit('defaults-ready', []),
          },
          'defaults-ready'
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

describe('OSearchView server defaults', () => {
  beforeEach(() => {
    actorState.id = 'me';
    sfSearch.mockReset();
    sfSearch.mockResolvedValue([]);
  });

  it('prefers private IsDefault over shared and emits first query-update after load', async () => {
    sfSearch.mockResolvedValue([
      {
        Id: 's1',
        Name: 'SharedDef',
        Condition: { And: [['S', '=', 1]] },
        IsDefault: true,
        UserId: null,
      },
      {
        Id: 'p1',
        Name: 'PrivateDef',
        Condition: { And: [['P', '=', 1]] },
        IsDefault: true,
        UserId: 'me',
      },
    ]);
    const wrapper = mount(OSearchView as any, {
      props: {
        store: makeStore(),
        defaultFilters: [{ name: 'Code', query: ['C', '=', 1], selected: true }],
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(wrapper.emitted('query-update')?.length).toBe(1);
    const defaultsText = wrapper.find('.defaults').text();
    const defaults = JSON.parse(defaultsText);
    expect(defaults[0]).toMatchObject({ name: 'PrivateDef', selected: true });
    expect(defaults.find((d: any) => d.name === 'Code')?.selected).toBe(false);
  });

  it('skips Search when application or modelName is missing', async () => {
    const wrapper = mount(OSearchView as any, {
      props: { store: makeStore({ application: '', modelName: 'Widget' }) },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(sfSearch).not.toHaveBeenCalled();
    expect(wrapper.emitted('query-update')?.length).toBe(1);
  });

  it('falls back to code defaults when Search throws', async () => {
    sfSearch.mockRejectedValue(new Error('unavailable'));
    const wrapper = mount(OSearchView as any, {
      props: {
        store: makeStore(),
        defaultFilters: [{ name: 'Code', query: ['C', '=', 1], selected: true }],
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    const defaults = JSON.parse(wrapper.find('.defaults').text());
    expect(defaults[0]).toMatchObject({ name: 'Code', selected: true });
  });

  it('reloads server defaults on defaults-ready and supports initialEmit=false', async () => {
    sfSearch.mockResolvedValue([
      { Id: 's1', Name: 'SharedOnly', Condition: {}, IsDefault: true, UserId: '' },
    ]);
    const wrapper = mount(OSearchView as any, {
      props: { store: makeStore(), initialEmit: false },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(wrapper.emitted('query-update')).toBeUndefined();
    expect(JSON.parse(wrapper.find('.defaults').text())[0].name).toBe('SharedOnly');

    sfSearch.mockResolvedValue([
      { Id: 'p2', Name: 'LaterPrivate', Condition: {}, IsDefault: true, UserId: 'me' },
    ]);
    await wrapper.find('.emit-defaults-ready').trigger('click');
    await flushPromises();
    await nextTick();
    expect(JSON.parse(wrapper.find('.defaults').text())[0].name).toBe('LaterPrivate');
  });

  it('accepts singleton defaultFilters and queryState.defaultFilters', async () => {
    const wrapper = mount(OSearchView as any, {
      props: {
        store: makeStore({ application: '', modelName: 'Widget' }),
        defaultFilters: { name: 'Solo', query: ['S', '=', 1], selected: true },
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(JSON.parse(wrapper.find('.defaults').text())[0]).toMatchObject({ name: 'Solo', selected: true });

    const qsWrapper = mount(OSearchView as any, {
      props: {
        store: makeStore({
          application: '',
          modelName: 'Widget',
          state: { queryState: { defaultFilters: [{ name: 'FromQs', query: ['Q', '=', 1], selected: true }] } },
        }),
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(JSON.parse(qsWrapper.find('.defaults').text())[0].name).toBe('FromQs');
  });

  it('skips Search when modelName is missing', async () => {
    const wrapper = mount(OSearchView as any, {
      props: { store: makeStore({ application: 'demo', modelName: '' }) },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(sfSearch).not.toHaveBeenCalled();
    expect(wrapper.emitted('query-update')?.length).toBe(1);
  });

  it('requests shared-only defaults when actor is empty', async () => {
    actorState.id = '';
    sfSearch.mockResolvedValue([{ Id: 's1', Name: 'Shared', Condition: {}, IsDefault: true, UserId: null }]);
    mount(OSearchView as any, {
      props: { store: makeStore() },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    const cond = sfSearch.mock.calls[0]![0] as any;
    expect(cond.And.at(-1)).toEqual({ Or: [['UserId', '=', null]] });
  });

  it('ignores stale server-default responses', async () => {
    let resolveSlow!: (rows: any[]) => void;
    const slow = new Promise<any[]>(resolve => {
      resolveSlow = resolve;
    });
    sfSearch.mockImplementationOnce(() => slow);
    sfSearch.mockResolvedValueOnce([
      { Id: 'p-fast', Name: 'FastPrivate', Condition: {}, IsDefault: true, UserId: 'me' },
    ]);
    const wrapper = mount(OSearchView as any, {
      props: { store: makeStore(), initialEmit: false },
      global: { stubs: { OSearch: OSearchStub } },
    });
    // First load is slow; trigger a second load that finishes first.
    await wrapper.find('.emit-defaults-ready').trigger('click');
    await flushPromises();
    await nextTick();
    expect(JSON.parse(wrapper.find('.defaults').text())[0].name).toBe('FastPrivate');

    resolveSlow([{ Id: 'p-slow', Name: 'SlowPrivate', Condition: {}, IsDefault: true, UserId: 'me' }]);
    await flushPromises();
    await nextTick();
    expect(JSON.parse(wrapper.find('.defaults').text())[0].name).toBe('FastPrivate');
  });
});
