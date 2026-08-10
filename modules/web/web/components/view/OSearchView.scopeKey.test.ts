// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

const { sfSearch, actorState, routeState } = vi.hoisted(() => ({
  sfSearch: vi.fn(async () => [] as any[]),
  actorState: { id: 'me' as string },
  routeState: { path: '/web/widgets/99/edit?x=1' },
}));

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({ _t: (msg: string) => msg, _lt: (msg: string) => msg }),
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

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<any>('vue-router');
  return {
    ...actual,
    useRoute: () => routeState,
  };
});

import OSearchView from './OSearchView.vue';

const OSearchStub = defineComponent({
  name: 'OSearch',
  props: ['store', 'placeholder', 'currentKeyword', 'currentAppliedFilters', 'currentAppliedGroups', 'defaultFilters'],
  emits: ['query-update', 'defaults-ready'],
  setup() {
    return () => h('div', { class: 'o-search-stub' });
  },
});

describe('OSearchView ScopeKey from route', () => {
  beforeEach(() => {
    actorState.id = 'me';
    routeState.path = '/web/widgets/99/edit?x=1';
    sfSearch.mockReset();
    sfSearch.mockResolvedValue([]);
  });

  it('passes normalized ScopeKey into SavedFilter Search', async () => {
    mount(OSearchView as any, {
      props: {
        store: { application: 'demo', modelName: 'Widget', state: { queryState: {} } },
        initialEmit: false,
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(sfSearch.mock.calls[0]![0].And).toEqual(
      expect.arrayContaining([['ScopeKey', '=', '/web/widgets/:id/edit']])
    );
  });

  it('uses empty ScopeKey when route.path is missing', async () => {
    routeState.path = undefined as any;
    mount(OSearchView as any, {
      props: {
        store: { application: 'demo', modelName: 'Widget', state: { queryState: {} } },
        initialEmit: false,
      },
      global: { stubs: { OSearch: OSearchStub } },
    });
    await flushPromises();
    expect(sfSearch.mock.calls[0]![0].And).toEqual(expect.arrayContaining([['ScopeKey', '=', '']]));
  });
});
