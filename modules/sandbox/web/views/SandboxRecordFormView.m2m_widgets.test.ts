// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { buildPageMountGlobal } from '@choysum/page-mount';
import ChoyManyToManyTagsField from '@/web/web/components/field/ChoyManyToManyTagsField.vue';
import ChoyManyToManyTreeField from '@/web/web/components/field/ChoyManyToManyTreeField.vue';
import SandboxRecordFormView from './SandboxRecordFormView.vue';

function makeStore() {
  return {
    fullModelName: 'sandbox.SandboxRecord',
    storeId: 'sandbox-form-m2m-' + Math.random().toString(36).slice(2),
    fieldsMetadata: {
      Tags: { type: 'ManyToMany', relation: 'sandbox.SandboxTag' },
      TagChips: { type: 'ManyToMany', relation: 'sandbox.SandboxTag' },
      Categories: { type: 'ManyToMany', relation: 'sandbox.SandboxCategory' },
      TagRefIds: { type: 'ManyToManyRef', relation: 'sandbox.SandboxTag' },
      CategoryPathIds: { type: 'ManyToManyRef', relation: 'sandbox.SandboxCategory' },
      NameChar: { type: 'char' },
      Title: { type: 'varchar' },
      Code: { type: 'varchar' },
      IsActive: { type: 'boolean' },
      Sequence: { type: 'int' },
      State: { type: 'selection' },
      Lines: { type: 'OneToMany', relation: 'sandbox.SandboxLine' },
    },
    state: {
      current: {
        Id: 'rec-1',
        NameChar: 'Sample',
        Title: 'Sample',
        Code: 'SAMPLE',
        IsActive: true,
        Sequence: 10,
        State: 'draft',
        Tags: [],
        TagChips: [],
        Categories: [],
        TagRefIds: [],
        CategoryPathIds: [],
        Lines: [],
      },
      queryState: { keyword: '', appliedFilters: [], appliedGroups: [], keywordFields: [], pagination: { limit: 20, offset: 0 } },
      result: { total: 0 },
      selection: [],
      planCache: new Map(),
    },
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_c: any, fn: any) => fn(),
    Browse: async () => null,
    FieldsGet: async () => ({}),
  } as any;
}

describe('SandboxRecordFormView M2M Tags/Tree widgets', () => {
  beforeEach(() => {
    stubSfc(ChoyManyToManyTagsField as any, {
      name: 'ChoyManyToManyTagsField',
      props: { prop: { type: String, default: '' } },
      setup(props: any) {
        return () => h('div', { 'data-testid': 'sandbox-m2m-tags', 'data-prop': props.prop });
      },
    } as any);
    stubSfc(ChoyManyToManyTreeField as any, {
      name: 'ChoyManyToManyTreeField',
      props: { prop: { type: String, default: '' } },
      setup(props: any) {
        return () => h('div', { 'data-testid': 'sandbox-m2m-tree', 'data-prop': props.prop });
      },
    } as any);
  });

  afterEach(() => {
    restoreSfc(ChoyManyToManyTagsField as any);
    restoreSfc(ChoyManyToManyTreeField as any);
  });

  test('wires TagChips to ChoyManyToManyTagsField and Categories to ChoyManyToManyTreeField', async () => {
    const { plugins } = buildPageMountGlobal({
      route: { path: '/sandbox/records/rec-1', fullPath: '/sandbox/records/rec-1', name: 'SandboxRecordDetail' },
    });
    const { unmount, el } = mountApp(SandboxRecordFormView as any, {
      props: {
        store: makeStore(),
        recordId: 'rec-1',
        viewMode: 'edit',
        showHeader: false,
      },
      plugins,
    });
    await flushPromises();

    const tags = el.querySelector('[data-testid="sandbox-m2m-tags"]') as HTMLElement | null;
    const tree = el.querySelector('[data-testid="sandbox-m2m-tree"]') as HTMLElement | null;
    expect(tags).toBeTruthy();
    expect(tags?.getAttribute('data-prop')).toBe('TagChips');
    expect(tree).toBeTruthy();
    expect(tree?.getAttribute('data-prop')).toBe('Categories');

    unmount();
  });
});
