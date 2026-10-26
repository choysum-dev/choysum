// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { buildPageMountGlobal } from '@choysum/page-mount';
import ChoyManyToManyTagsField from '@/web/web/components/field/ChoyManyToManyTagsField.vue';
import ChoyManyToManyTreeField from '@/web/web/components/field/ChoyManyToManyTreeField.vue';
import ChoyManyToManyRefField from '@/web/web/components/field/ChoyManyToManyRefField.vue';
import ChoyManyToManyRefTagsField from '@/web/web/components/field/ChoyManyToManyRefTagsField.vue';
import ChoyManyToManyRefTreeField from '@/web/web/components/field/ChoyManyToManyRefTreeField.vue';
import SandboxRecordFormView from './SandboxRecordFormView.vue';

function makeStore() {
  return {
    fullModelName: 'sandbox.SandboxRecord',
    storeId: 'sandbox-form-m2m-' + Math.random().toString(36).slice(2),
    fieldsMetadata: {
      Tags: { type: 'ManyToMany', relation: 'sandbox.SandboxTag' },
      TagChips: { type: 'ManyToMany', relation: 'sandbox.SandboxTag' },
      Categories: { type: 'ManyToMany', relation: 'sandbox.SandboxCategory' },
      TagTableRefIds: { type: 'ManyToManyRef', relation: 'sandbox.SandboxTag' },
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
        TagTableRefIds: [],
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
    stubSfc(ChoyManyToManyRefField as any, {
      name: 'ChoyManyToManyRefField',
      props: { prop: { type: String, default: '' } },
      setup(props: any) {
        return () => h('div', { 'data-testid': 'sandbox-m2m-ref-list', 'data-prop': props.prop });
      },
    } as any);
    stubSfc(ChoyManyToManyRefTagsField as any, {
      name: 'ChoyManyToManyRefTagsField',
      props: { prop: { type: String, default: '' } },
      setup(props: any) {
        return () => h('div', { 'data-testid': 'sandbox-m2m-ref-tags', 'data-prop': props.prop });
      },
    } as any);
    stubSfc(ChoyManyToManyRefTreeField as any, {
      name: 'ChoyManyToManyRefTreeField',
      props: { prop: { type: String, default: '' } },
      setup(props: any) {
        return () => h('div', { 'data-testid': 'sandbox-m2m-ref-tree', 'data-prop': props.prop });
      },
    } as any);
  });

  afterEach(() => {
    restoreSfc(ChoyManyToManyTagsField as any);
    restoreSfc(ChoyManyToManyTreeField as any);
    restoreSfc(ChoyManyToManyRefField as any);
    restoreSfc(ChoyManyToManyRefTagsField as any);
    restoreSfc(ChoyManyToManyRefTreeField as any);
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

  test('wires ManyToManyRef list + Tags + Tree widgets', async () => {
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

    const list = el.querySelector('[data-testid="sandbox-m2m-ref-list"]') as HTMLElement | null;
    const tags = el.querySelector('[data-testid="sandbox-m2m-ref-tags"]') as HTMLElement | null;
    const tree = el.querySelector('[data-testid="sandbox-m2m-ref-tree"]') as HTMLElement | null;
    expect(list).toBeTruthy();
    expect(list?.getAttribute('data-prop')).toBe('TagTableRefIds');
    expect(tags).toBeTruthy();
    expect(tags?.getAttribute('data-prop')).toBe('TagRefIds');
    expect(tree).toBeTruthy();
    expect(tree?.getAttribute('data-prop')).toBe('CategoryPathIds');

    unmount();
  });
});
