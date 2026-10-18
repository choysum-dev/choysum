// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Workspace height: default auto mode leaves table wrap without pixel height (flex-1).
 */
import { h } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyListView from './ChoyListView.vue';
import ChoyPagination from './ChoyPagination.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';

function makeStore() {
  return {
    fullModelName: 'partner.Partner',
    storeId: 'list-height-' + Math.random().toString(36).slice(2),
    fieldsMetadata: { Name: { type: 'varchar' } },
    state: {
      queryState: {
        keyword: '',
        appliedFilters: [],
        appliedGroups: [],
        keywordFields: [],
        pagination: { limit: 20, offset: 0 },
      },
      result: { total: 1 },
      selection: [],
      planCache: new Map(),
    },
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_c: any, fn: any) => fn(),
    Search: async () => [{ Id: '1', Name: 'A' }],
  } as any;
}

function stubListChrome() {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    setup: (_: any, { slots }: any) => () => h('button', slots.default?.()),
  });
  stubSfc(ChoyTableHost, {
    name: 'ChoyTableHost',
    setup: (_p: any, { slots }: any) => () => h('div', { 'data-stub': 'ChoyTableHost' }, slots.default?.()),
  });
  stubSfc(ChoyTableColumn, { name: 'ChoyTableColumn', setup: () => () => null });
  stubSfc(ChoyPagination, { name: 'ChoyPagination', setup: () => () => null });
  stubSfc(ListInlineEditScope, {
    name: 'ListInlineEditScope',
    setup: (p: any, { slots }: any) => () => h('div', { class: p.class }, slots.default?.()),
  });
}

describe('ChoyListView workspace height', () => {
  beforeEach(() => stubListChrome());
  afterEach(() => {
    restoreSfc(ChoyButton as any);
    restoreSfc(ChoyTableHost);
    restoreSfc(ChoyTableColumn);
    restoreSfc(ChoyPagination);
    restoreSfc(ListInlineEditScope);
  });

  test('auto mode leaves table height unlocked for flex fill', async () => {
    const { plugins } = buildPageMountGlobal({
      route: { name: 'PartnerList', path: '/partner/partners', fullPath: '/partner/partners' },
    });
    const mounted = mountApp(ChoyListView as any, {
      props: {
        store: makeStore(),
        showHeader: false,
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      plugins,
    });
    await flushPromises();
    const wrap = mounted.q('[data-anchor="choy.list.table"]') as HTMLElement | null;
    expect(wrap).not.toBeNull();
    expect(wrap!.style.height || '').toBe('');
    expect(wrap!.style.maxHeight || '').toBe('');
    expect(wrap!.className).toContain('flex-1');
    expect(wrap!.className.includes('bg-card')).toBe(false);
    expect(wrap!.className.includes('border-border')).toBe(false);
    expect(wrap!.className.includes('rounded-md')).toBe(false);
    expect(wrap!.className.includes('shadow-xs')).toBe(false);

    const sheet = mounted.q('[data-anchor="choy.view.sheet"]') as HTMLElement | null;
    expect(sheet).not.toBeNull();
    // White surface is shell inset <main>; view-body is scroll region only.
    expect(sheet!.className.includes('bg-card')).toBe(false);
    mounted.unmount();
  });
});
