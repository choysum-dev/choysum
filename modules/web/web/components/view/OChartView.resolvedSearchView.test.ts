// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, markRaw, reactive } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

const { applyMock, awaitFieldSelectionMock } = vi.hoisted(() => ({
  applyMock: vi.fn(async () => {}),
  awaitFieldSelectionMock: vi.fn(async () => {}),
}));

vi.mock('@/web/web/controllers/chartController', () => ({
  createChartController: vi.fn(() => ({
    vm: reactive({
      loading: false,
      error: null,
      result: null,
    }),
    apply: applyMock,
  })),
}));

vi.mock('@/web/web/query/utils/registry/fieldReady', () => ({
  awaitFieldSelection: (...args: any[]) => awaitFieldSelectionMock(...args),
}));

vi.mock('@/web/web/query/utils/registry/metric', () => ({
  exportMetrics: () => [],
}));

vi.mock('@/web/web/components/chart/chartTypeAdapter', () => ({
  resolveChartAdapter: () => null,
  ensureEChartsRegistered: () => {},
  chartTypeRegistry: {},
}));

vi.mock('vue-echarts', () => ({
  default: defineComponent({ name: 'VChartStub', setup: () => () => null }),
}));

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({ _t: (msg: string) => msg, _lt: (msg: string) => msg }),
  };
});

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  };
});

import OChartView from './OChartView.vue';

function makeStore() {
  return {
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

const stubs = {
  OViewContainer: {
    template: `<div class="ovc"><slot name="header" /><slot name="fields" /><slot /></div>`,
  },
  'el-button': true,
  'el-icon': true,
  'el-select': true,
  'el-option': true,
  'el-tooltip': true,
  'el-button-group': true,
};

describe('OChartView resolvedSearchView', () => {
  it('renders markRaw searchView and covers the falsy resolvedSearchView branch', async () => {
    const SearchProbe = markRaw(
      defineComponent({
        name: 'ChartSearchProbe',
        setup: () => () => h('div', { class: 'chart-search-probe' }),
      })
    );

    const withSearch = mount(OChartView as any, {
      props: {
        store: makeStore(),
        searchView: SearchProbe,
        showHeader: true,
        showActions: false,
        showChartControls: false,
        refreshAction: false,
      },
      global: { stubs },
    });
    try {
      await flushPromises();
      expect(withSearch.find('.chart-search-probe').exists()).toBe(true);
      expect(withSearch.find('.o-chart__search').exists()).toBe(true);
    } finally {
      withSearch.unmount();
    }

    const withoutSearch = mount(OChartView as any, {
      props: {
        store: makeStore(),
        searchView: undefined,
        showHeader: true,
        showActions: false,
        showChartControls: false,
        refreshAction: false,
      },
      global: { stubs },
    });
    try {
      await flushPromises();
      expect(withoutSearch.find('.o-chart__search').exists()).toBe(false);
    } finally {
      withoutSearch.unmount();
    }
  });
});
