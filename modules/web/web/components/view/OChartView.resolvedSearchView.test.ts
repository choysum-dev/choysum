// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, markRaw } from 'vue';

import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OViewContainer from './OViewContainer.vue';
import OChartView from './OChartView.vue';

function makeStore() {
  return {
    storeId: 'chart-s',
    fieldsMetadata: {},
    state: {
      queryState: {
        keyword: '',
        appliedFilters: [],
        appliedGroups: [],
        keywordFields: [],
        pagination: { limit: 20, offset: 0 },
      },
    },
  } as any;
}

describe('OChartView resolvedSearchView', () => {
  afterEach(() => {
    restoreSfc(OViewContainer);
  });

  test('renders markRaw searchView and covers the falsy resolvedSearchView branch', async () => {
    stubSfc(OViewContainer, {
      name: 'OViewContainer',
      setup(_props: any, { slots }: any) {
        return () => h('div', { class: 'ovc' }, [slots.header?.(), slots.fields?.(), slots.default?.()]);
      },
    });

    const SearchProbe = markRaw(
      defineComponent({
        name: 'ChartSearchProbe',
        setup: () => () => h('div', { class: 'chart-search-probe' }),
      })
    );

    const withSearch = mountApp(OChartView as any, {
      props: {
        store: makeStore(),
        searchView: SearchProbe,
        showHeader: true,
        showActions: false,
        showChartControls: false,
        refreshAction: false,
      },
      stubs: {
        ElButton: true,
        ElIcon: true,
        ElSelect: true,
        ElOption: true,
        ElTooltip: true,
        ElButtonGroup: true,
      },
    });
    try {
      await flushPromises();
      expect(withSearch.q('.chart-search-probe')).toBeTruthy();
      expect(withSearch.q('.o-chart__search')).toBeTruthy();
    } finally {
      withSearch.unmount();
    }

    const withoutSearch = mountApp(OChartView as any, {
      props: {
        store: makeStore(),
        searchView: undefined,
        showHeader: true,
        showActions: false,
        showChartControls: false,
        refreshAction: false,
      },
      stubs: {
        ElButton: true,
        ElIcon: true,
        ElSelect: true,
        ElOption: true,
        ElTooltip: true,
        ElButtonGroup: true,
      },
    });
    try {
      await flushPromises();
      expect(withoutSearch.q('.o-chart__search')).toBeFalsy();
    } finally {
      withoutSearch.unmount();
    }
  });
});
