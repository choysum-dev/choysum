// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, defineComponent, inject, type Component } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyPage from '../layout/ChoyPage.vue';
import ChoyFormView from '../view/ChoyFormView.vue';
import ChoyListView from '../view/ChoyListView.vue';
import ChoySearchView from '../view/ChoySearchView.vue';
import ChoyViewScope from '../view/ChoyViewScope.vue';
import ChoyButtonBox from '../view/ChoyButtonBox.vue';
import ChoyStatInfo from '../view/ChoyStatInfo.vue';
import ChoyVColumn from '../vtable/ChoyVColumn.vue';
import ChoyVarcharField from './ChoyVarcharField.vue';
import ChoyTextField from './ChoyTextField.vue';
import ChoyBooleanField from './ChoyBooleanField.vue';
import ChoySelectionField from './ChoySelectionField.vue';
import ChoyDatetimeField from './ChoyDatetimeField.vue';
import ChoyDateField from './ChoyDateField.vue';
import ChoyNumberField from './ChoyNumberField.vue';
import ChoyJsonField from './ChoyJsonField.vue';
import ChoyImageField from './ChoyImageField.vue';
import ChoyVirtualField from './ChoyVirtualField.vue';
import ChoyManyToOneField from './ChoyManyToOneField.vue';
import ChoyManyToManyField from './ChoyManyToManyField.vue';
import ChoyOneToManyField from './ChoyOneToManyField.vue';
import OFormView from '../view/OFormView.vue';
import OListView from '../view/OListView.vue';
import OSearchView from '../view/OSearchView.vue';
import OVarCharField from './OVarCharField.vue';
import OTextField from './OTextField.vue';
import OBooleanField from './OBooleanField.vue';
import OSelectionField from './OSelectionField.vue';
import ODatetimeField from './ODatetimeField.vue';
import ODateField from './ODateField.vue';
import OIntField from './OIntField.vue';
import ODecimalField from './ODecimalField.vue';
import OBigintField from './OBigintField.vue';
import ONumberField from './ONumberField.vue';
import OJsonobjectField from './OJsonobjectField.vue';
import OImageField from './OImageField.vue';
import OVirtualField from './OVirtualField.vue';
import OManyToOneField from './OManyToOneField.vue';
import OManyToOneRefField from './OManyToOneRefField.vue';
import OManyToManyField from './OManyToManyField.vue';
import OManyToManyRefTagsField from './OManyToManyRefTagsField.vue';
import OManyToManyRefTreeField from './OManyToManyRefTreeField.vue';
import OOneToManyField from './OOneToManyField.vue';
import OStatInfo from '../view/OStatInfo.vue';
import OVColumn from '../vtable/OVColumn.vue';

const fakeStore = { modelName: 'auth.User', meta: { fields: {} } } as any;

function stubHost(Comp: Component, testId: string) {
  stubSfc(Comp, {
    props: { store: null, prop: String, valueMode: String, widget: String, valueFormat: String },
    setup: ((props: any) => {
      return () =>
        h('div', {
          'data-test': testId,
          'data-prop': String(props.prop || ''),
          'data-value-mode': String(props.valueMode || ''),
          'data-widget': String(props.widget || ''),
          'data-value-format': String(props.valueFormat || ''),
        });
    }) as any,
  });
}

describe('Choy store-mode field hosts', () => {
  const stubs: any[] = [
    OVarCharField,
    OTextField,
    OBooleanField,
    OSelectionField,
    ODatetimeField,
    ODateField,
    OIntField,
    ODecimalField,
    OBigintField,
    ONumberField,
    OJsonobjectField,
    OImageField,
    OVirtualField,
    OManyToOneField,
    OManyToOneRefField,
    OManyToManyField,
    OManyToManyRefTagsField,
    OManyToManyRefTreeField,
    OOneToManyField,
    OFormView,
    OListView,
    OSearchView,
    OStatInfo,
    OVColumn,
  ];

  beforeEach(() => {
    stubHost(OVarCharField as any, 'o-varchar');
    stubHost(OTextField as any, 'o-text');
    stubHost(OBooleanField as any, 'o-boolean');
    stubHost(OSelectionField as any, 'o-selection');
    stubHost(ODatetimeField as any, 'o-datetime');
    stubHost(ODateField as any, 'o-date');
    stubHost(OIntField as any, 'o-int');
    stubHost(ODecimalField as any, 'o-decimal');
    stubHost(OBigintField as any, 'o-bigint');
    stubHost(ONumberField as any, 'o-number');
    stubHost(OJsonobjectField as any, 'o-json');
    stubHost(OImageField as any, 'o-image');
    stubHost(OVirtualField as any, 'o-virtual');
    stubHost(OManyToOneField as any, 'o-m2o');
    stubHost(OManyToOneRefField as any, 'o-m2o-ref');
    stubHost(OManyToManyField as any, 'o-m2m');
    stubHost(OManyToManyRefTagsField as any, 'o-m2m-tags');
    stubHost(OManyToManyRefTreeField as any, 'o-m2m-tree');
    stubHost(OOneToManyField as any, 'o-o2m');
    stubHost(OFormView as any, 'o-form');
    stubHost(OListView as any, 'o-list');
    stubHost(OSearchView as any, 'o-search');
    stubHost(OStatInfo as any, 'o-stat');
    stubHost(OVColumn as any, 'o-vcolumn');
  });

  afterEach(() => {
    for (const c of stubs) restoreSfc(c);
  });

  async function mountField(Comp: any, props: Record<string, unknown>) {
    const Host = defineComponent({
      setup() {
        return () => h(Comp, props as any);
      },
    });
    const wrapper = mountApp(Host);
    await flushPromises();
    return wrapper;
  }

  test('scalar fields host O* when store+prop set', async () => {
    const cases: Array<[any, string]> = [
      [ChoyVarcharField, 'o-varchar'],
      [ChoyTextField, 'o-text'],
      [ChoyBooleanField, 'o-boolean'],
      [ChoySelectionField, 'o-selection'],
      [ChoyDatetimeField, 'o-datetime'],
      [ChoyDateField, 'o-date'],
      [ChoyJsonField, 'o-json'],
      [ChoyImageField, 'o-image'],
      [ChoyVirtualField, 'o-virtual'],
      [ChoyOneToManyField, 'o-o2m'],
    ];
    for (const [Comp, id] of cases) {
      const w = await mountField(Comp, { store: fakeStore, prop: 'Name' });
      const host = w.q(`[data-test=${id}]`);
      expect(host).not.toBeNull();
      expect(host?.getAttribute('data-prop')).toBe('Name');
      w.unmount();
    }
  });

  test('NumberField store mode routes integer / bigint / decimal / float hosts', async () => {
    const intW = await mountField(ChoyNumberField, {
      store: fakeStore,
      prop: 'Padding',
      mode: 'integer',
    });
    expect(intW.q('[data-test=o-int]')).not.toBeNull();
    expect(intW.q('[data-test=o-decimal]')).toBeNull();
    intW.unmount();

    const bigW = await mountField(ChoyNumberField, {
      store: fakeStore,
      prop: 'NextNumber',
      mode: 'bigint',
    });
    expect(bigW.q('[data-test=o-bigint]')).not.toBeNull();
    bigW.unmount();

    const decW = await mountField(ChoyNumberField, {
      store: fakeStore,
      prop: 'Rate',
      mode: 'decimal',
    });
    expect(decW.q('[data-test=o-decimal]')).not.toBeNull();
    decW.unmount();

    const floatW = await mountField(ChoyNumberField, {
      store: fakeStore,
      prop: 'Factor',
      mode: 'float',
    });
    expect(floatW.q('[data-test=o-number]')).not.toBeNull();
    expect(floatW.q('[data-test=o-decimal]')).toBeNull();
    floatW.unmount();

    // Default mode is decimal so Decimal-typed domain fields stay on ODecimalField.
    const defaultW = await mountField(ChoyNumberField, {
      store: fakeStore,
      prop: 'Rounding',
    });
    expect(defaultW.q('[data-test=o-decimal]')).not.toBeNull();
    defaultW.unmount();
  });

  test('DateField store mode passes date-only valueFormat', async () => {
    const w = await mountField(ChoyDateField, {
      store: fakeStore,
      prop: 'Date',
    });
    const host = w.q('[data-test=o-date]');
    expect(host).not.toBeNull();
    expect(host?.getAttribute('data-value-format')).toBe('YYYY-MM-DD');
    w.unmount();
  });

  test('NumberField chrome mode renders anchor and uses bigint inputmode', async () => {
    const w = await mountField(ChoyNumberField, {
      label: 'Count',
      mode: 'bigint',
      modelValue: 42,
    });
    const root = w.q('[data-anchor="choy.number-field"]');
    expect(root).not.toBeNull();
    expect(w.q('[data-test=o-bigint]')).toBeNull();
    const input = w.q('input') as HTMLInputElement | null;
    expect(input?.getAttribute('inputmode')).toBe('numeric');
    w.unmount();
  });

  test('NumberField chrome commits draft and syncs host model changes', async () => {
    const emitted: Array<number | null> = [];
    let propsBag: Record<string, unknown> | null = null;
    const w = mountApp(ChoyNumberField as any, {
      reactiveProps: true,
      props: {
        label: 'Amount',
        mode: 'float',
        modelValue: 1,
      },
      on: {
        'onUpdate:modelValue': (v: number | null) => {
          emitted.push(v);
          if (propsBag) propsBag.modelValue = v;
        },
      },
    });
    propsBag = w.props;
    await flushPromises();

    const input = w.q('input') as HTMLInputElement | null;
    expect(input).not.toBeNull();
    input!.value = '3.5';
    input!.dispatchEvent(new Event('input', { bubbles: true }));
    input!.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(emitted[emitted.length - 1]).toBe(3.5);
    expect(input!.value).toBe('3.5');

    // Host rewrite path: external model change while draft still looks different.
    input!.value = 'x';
    input!.dispatchEvent(new Event('input', { bubbles: true }));
    propsBag.modelValue = 9;
    await flushPromises();
    expect(input!.value).toBe('9');

    w.unmount();
  });

  test('ManyToOne store mode uses Ref by default and record when valueMode=record', async () => {
    const refW = await mountField(ChoyManyToOneField, {
      store: fakeStore,
      prop: 'CompanyId',
    });
    const refHost = refW.q('[data-test=o-m2o-ref]');
    expect(refHost).not.toBeNull();
    expect(refHost?.getAttribute('data-prop')).toBe('CompanyId');
    expect(refW.q('[data-test=o-m2o]')).toBeNull();
    refW.unmount();

    const recW = await mountField(ChoyManyToOneField, {
      store: fakeStore,
      prop: 'CompanyId',
      valueMode: 'record',
    });
    const recHost = recW.q('[data-test=o-m2o]');
    expect(recHost).not.toBeNull();
    expect(recHost?.getAttribute('data-prop')).toBe('CompanyId');
    expect(recHost?.getAttribute('data-value-mode')).toBe('record');
    expect(recW.q('[data-test=o-m2o-ref]')).toBeNull();
    recW.unmount();
  });

  test('ManyToOne chrome mode renders without search via noop resolver', async () => {
    const w = await mountField(ChoyManyToOneField, { label: 'Company' });
    expect(w.q('[data-anchor="choy.many-to-one-field"]')).not.toBeNull();
    expect(w.q('[data-test=o-m2o-ref]')).toBeNull();
    w.unmount();
  });

  test('ManyToMany store mode routes ref tags/tree and default OManyToMany', async () => {
    const defW = await mountField(ChoyManyToManyField, {
      store: fakeStore,
      prop: 'RoleIds',
      widget: 'list',
    });
    expect(defW.q('[data-test=o-m2m]')).not.toBeNull();
    defW.unmount();

    const tagsW = await mountField(ChoyManyToManyField, {
      store: fakeStore,
      prop: 'RoleIds',
      widget: 'tags',
      valueMode: 'ref',
    });
    expect(tagsW.q('[data-test=o-m2m-tags]')).not.toBeNull();
    tagsW.unmount();

    const treeW = await mountField(ChoyManyToManyField, {
      store: fakeStore,
      prop: 'RoleIds',
      widget: 'tree',
      valueMode: 'ref',
    });
    expect(treeW.q('[data-test=o-m2m-tree]')).not.toBeNull();
    treeW.unmount();
  });

  test('Form/List/Search store mode hosts O* engines', async () => {
    const form = await mountField(ChoyFormView, { store: fakeStore });
    expect(form.q('[data-test=o-form]')).not.toBeNull();
    form.unmount();

    const list = await mountField(ChoyListView, { store: fakeStore });
    expect(list.q('[data-test=o-list]')).not.toBeNull();
    list.unmount();

    const search = await mountField(ChoySearchView, { store: fakeStore });
    expect(search.q('[data-test=o-search]')).not.toBeNull();
    search.unmount();
  });

  test('FormView chrome forwards attrs and named slot chrome paths', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyFormView,
            { title: 'T', 'data-test': 'form-chrome', class: 'extra-class' } as any,
            {
              breadcrumb: () => h('span', { 'data-test': 'crumb' }),
              'system-actions': () => h('span', { 'data-test': 'sys' }),
              'user-actions': () => h('span', { 'data-test': 'usr' }),
              statusbar: () => h('span', { 'data-test': 'status' }),
              'button-box': () => h('span', { 'data-test': 'bbox' }),
              'header-right': () => h('span', { 'data-test': 'right' }),
              default: () => h('span', 'body'),
            },
          );
      },
    });
    const w = mountApp(Host);
    await flushPromises();
    const root = w.q('[data-anchor="choy.form-view"]') as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root?.getAttribute('data-test')).toBe('form-chrome');
    expect(w.q('[data-test=crumb]')).not.toBeNull();
    expect(w.q('[data-test=sys]')).not.toBeNull();
    expect(w.q('[data-test=usr]')).not.toBeNull();
    expect(w.q('[data-test=status]')).not.toBeNull();
    expect(w.q('[data-test=bbox]')).not.toBeNull();
    expect(w.q('[data-test=right]')).not.toBeNull();
    w.unmount();
  });

  test('FormView store mode forwards named slots to OFormView', async () => {
    stubSfc(OFormView, {
      props: { store: null },
      setup: ((_p: any, { slots }: any) => {
        return () =>
          h('div', { 'data-test': 'o-form-slots' }, [
            slots.breadcrumb ? h('div', { 'data-test': 'slot-crumb' }, slots.breadcrumb()) : null,
            slots['button-box']
              ? h('div', { 'data-test': 'slot-bbox' }, slots['button-box']())
              : null,
            slots.default?.(),
          ]);
      }) as any,
    });
    try {
      const Host = defineComponent({
        setup() {
          return () =>
            h(
              ChoyFormView,
              { store: fakeStore },
              {
                breadcrumb: () => h('span', 'b'),
                'button-box': () => h('span', 'bb'),
                default: () => h('span', 'd'),
              },
            );
        },
      });
      const w = mountApp(Host);
      await flushPromises();
      expect(w.q('[data-test=o-form-slots]')).not.toBeNull();
      expect(w.q('[data-test=slot-crumb]')).not.toBeNull();
      expect(w.q('[data-test=slot-bbox]')).not.toBeNull();
      w.unmount();
    } finally {
      restoreSfc(OFormView as any);
      stubHost(OFormView as any, 'o-form');
    }
  });

  test('SearchView chrome submit and ListView chrome paths', async () => {
    const queries: Array<{ keyword: string }> = [];
    const sw = mountApp(ChoySearchView as any, {
      props: { keyword: 'hello' },
      on: {
        onQueryUpdate: (q: { keyword: string }) => {
          queries.push(q);
        },
      },
    });
    await flushPromises();
    expect(sw.q('[data-anchor="choy.search-view"]')).not.toBeNull();
    // QJS has no KeyboardEvent; drive the same handlers the template binds.
    const searchState = sw.setupState();
    searchState.submit();
    await flushPromises();
    expect(queries.length).toBeGreaterThan(0);
    expect(queries[0]?.keyword).toBe('hello');

    const beforeKey = queries.length;
    searchState.onKeydown({
      key: 'Enter',
      isComposing: false,
      keyCode: 13,
      preventDefault() {},
    });
    await flushPromises();
    expect(queries.length).toBeGreaterThan(beforeKey);

    // Ignored composing / non-Enter keydowns.
    const beforeIgnored = queries.length;
    searchState.onKeydown({
      key: 'Enter',
      isComposing: true,
      keyCode: 13,
      preventDefault() {},
    });
    searchState.onKeydown({
      key: 'Enter',
      isComposing: false,
      keyCode: 229,
      preventDefault() {},
    });
    searchState.onKeydown({
      key: 'a',
      isComposing: false,
      keyCode: 65,
      preventDefault() {},
    });
    await flushPromises();
    expect(queries.length).toBe(beforeIgnored);

    // Cover disabled early-return.
    const beforeDisabled = queries.length;
    const sd = mountApp(ChoySearchView as any, {
      props: { disabled: true, keyword: 'x' },
      on: {
        onQueryUpdate: (q: { keyword: string }) => {
          queries.push(q);
        },
      },
    });
    await flushPromises();
    sd.setupState().submit();
    sd.setupState().onKeydown({
      key: 'Enter',
      isComposing: false,
      keyCode: 13,
      preventDefault() {},
    });
    await flushPromises();
    expect(queries.length).toBe(beforeDisabled);
    sd.unmount();
    sw.unmount();

    const rowClicks: unknown[] = [];
    const lw = mountApp(ChoyListView as any, {
      props: {
        columns: [{ accessorKey: 'name', header: 'Name' }],
        data: [{ name: 'a' }],
      },
      on: {
        onRowClick: (row: unknown) => {
          rowClicks.push(row);
        },
      },
      slots: {
        header: () => h('span', { 'data-test': 'list-header' }),
        search: () => h('span', { 'data-test': 'list-search' }),
      },
    });
    await flushPromises();
    expect(lw.q('[data-anchor="choy.list-view"]')).not.toBeNull();
    expect(lw.q('[data-test=list-header]')).not.toBeNull();
    expect(lw.q('[data-test=list-search]')).not.toBeNull();
    const listState = lw.setupState();
    listState?.onRowSelection?.(['1']);
    listState?.onRowClick?.({ name: 'a' });
    await flushPromises();
    expect(rowClicks.length).toBeGreaterThan(0);
    lw.unmount();

    // Undeclared on* attrs forward via v-on once (keys stripped for toHandlers).
    const loadHits: unknown[] = [];
    stubSfc(OFormView as any, {
      props: { store: null },
      emits: ['load-success'],
      setup: ((_props: any, { emit }: any) => {
        return () =>
          h('button', {
            'data-test': 'emit-load',
            onClick: () => emit('load-success', { ok: true }),
          });
      }) as any,
    });
    const formListen = mountApp(ChoyFormView as any, {
      props: { store: fakeStore },
      on: {
        onLoadSuccess: (payload: unknown) => {
          loadHits.push(payload);
        },
      },
    });
    await flushPromises();
    formListen.click('[data-test=emit-load]');
    await flushPromises();
    expect(loadHits).toEqual([{ ok: true }]);
    formListen.unmount();
    restoreSfc(OFormView as any);
    stubHost(OFormView as any, 'o-form');

    const listListen = await mountField(ChoyListView, {
      store: fakeStore,
      onSelectionChange: () => undefined,
    });
    expect(listListen.q('[data-test=o-list]')).not.toBeNull();
    listListen.unmount();
    const searchListen = await mountField(ChoySearchView, {
      store: fakeStore,
      onDefaultsReady: () => undefined,
    });
    expect(searchListen.q('[data-test=o-search]')).not.toBeNull();
    searchListen.unmount();
  });

  test('OneToMany chrome covers rowKey/title/subtitle helpers', async () => {
    const model = [
      { Id: '1', Name: 'Alpha', Note: 'n1' },
      { Name: '', Note: '' },
    ];
    const w = mountApp(ChoyOneToManyField as any, {
      props: {
        modelValue: model,
        widget: 'kanban',
        titleField: 'Name',
        subtitleField: 'Note',
        editable: true,
      },
    });
    await flushPromises();
    expect(w.q('[data-anchor="choy.one-to-many-field"]')).not.toBeNull();
    expect(w.text()).toContain('Alpha');
    expect(w.text()).toContain('n1');
    // Custom rowId path + empty subtitle early-return.
    const w2 = mountApp(ChoyOneToManyField as any, {
      props: {
        modelValue: [{ Name: 'Beta' }],
        widget: 'kanban',
        titleField: 'Name',
        rowId: (row: { Name: string }) => `k-${row.Name}`,
      },
    });
    await flushPromises();
    expect(w2.text()).toContain('Beta');
    w2.unmount();
    w.unmount();
  });

  test('field prop under ChoyPage store hosts O* without explicit store prop', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyPage, { store: fakeStore, title: 'Page' }, () =>
            h(ChoyVarcharField, { prop: 'Name' }),
          );
      },
    });
    const w = mountApp(Host);
    await flushPromises();
    expect(w.q('[data-test=o-varchar]')).not.toBeNull();
    expect(w.q('[data-anchor="choy.varchar-field"]')).toBeNull();
    w.unmount();
  });

  test('store List/Search forward engine events and Search bind props', async () => {
    const rows: unknown[] = [];
    stubSfc(OListView as any, {
      props: { store: null },
      emits: ['row-click'],
      setup: ((_props: any, { emit }: any) => {
        return () =>
          h('button', {
            'data-test': 'emit-row',
            onClick: () => emit('row-click', { row: { name: 'r1' }, rowIndex: 0 }),
          });
      }) as any,
    });
    const list = mountApp(ChoyListView as any, {
      props: { store: fakeStore },
      on: {
        onRowClick: (row: unknown) => {
          rows.push(row);
        },
      },
    });
    await flushPromises();
    list.click('[data-test=emit-row]');
    await flushPromises();
    expect(rows).toEqual([{ name: 'r1' }]);
    list.unmount();
    restoreSfc(OListView as any);

    // Nullish payload.row must not emit the wrapper object as the row.
    stubSfc(OListView as any, {
      props: { store: null },
      emits: ['row-click'],
      setup: ((_props: any, { emit }: any) => {
        return () =>
          h('div', [
            h('button', {
              'data-test': 'emit-row-null',
              onClick: () => emit('row-click', { row: null, rowIndex: 0 }),
            }),
            h('button', {
              'data-test': 'emit-row-bare',
              onClick: () => emit('row-click', { name: 'bare' }),
            }),
          ]);
      }) as any,
    });
    const guardRows: unknown[] = [];
    const listGuard = mountApp(ChoyListView as any, {
      props: { store: fakeStore },
      on: {
        onRowClick: (row: unknown) => {
          guardRows.push(row);
        },
      },
    });
    await flushPromises();
    listGuard.click('[data-test=emit-row-null]');
    await flushPromises();
    expect(guardRows).toEqual([]);
    listGuard.click('[data-test=emit-row-bare]');
    await flushPromises();
    expect(guardRows).toEqual([{ name: 'bare' }]);
    listGuard.unmount();
    restoreSfc(OListView as any);
    stubHost(OListView as any, 'o-list');

    const queries: Array<{ keyword: string }> = [];
    let seenBind: Record<string, unknown> = {};
    stubSfc(OSearchView as any, {
      props: { store: null, placeholder: String, disabled: Boolean },
      emits: ['query-update'],
      setup: ((props: any, { emit }: any) => {
        seenBind = {
          placeholder: props.placeholder,
          disabled: props.disabled,
        };
        return () =>
          h('button', {
            'data-test': 'emit-query',
            onClick: () => {
              const payload = {
                keyword: '  acme  ',
                appliedFilters: [{ children: [{ field: 'name', operator: '=', value: 'a' }] }],
              };
              emit('query-update', payload);
              props.onQueryUpdate?.(payload);
            },
          });
      }) as any,
    });
    const search = mountApp(ChoySearchView as any, {
      props: {
        store: fakeStore,
        placeholder: 'Find…',
        disabled: true,
      },
      on: {
        onQueryUpdate: (q: { keyword: string; filters: unknown[] }) => {
          queries.push(q as any);
        },
      },
    });
    await flushPromises();
    expect(seenBind.placeholder).toBe('Find…');
    expect(seenBind.disabled).toBe(true);
    search.click('[data-test=emit-query]');
    await flushPromises();
    expect(queries.length).toBeGreaterThan(0);
    expect(queries[0]?.keyword).toBe('acme');
    expect((queries[0] as any)?.filters?.length).toBe(1);
    search.unmount();
    restoreSfc(OSearchView as any);
    stubHost(OSearchView as any, 'o-search');
  });

  test('ViewScope / ButtonBox / StatInfo / VColumn mount', async () => {
    let injectedMode: unknown = undefined;
    const Probe = defineComponent({
      setup() {
        injectedMode = inject('view-mode');
        return () => h('span', { 'data-test': 'scope-probe' });
      },
    });
    const scopeHost = defineComponent({
      setup() {
        return () =>
          h(ChoyViewScope, { viewMode: 'display', container: 'List' }, () => h(Probe));
      },
    });
    const scope = mountApp(scopeHost);
    await flushPromises();
    expect(scope.q('[data-test=scope-probe]')).not.toBeNull();
    expect((injectedMode as { value?: string } | null)?.value).toBe('display');
    scope.unmount();

    const HostBox = defineComponent({
      setup() {
        return () => h(ChoyButtonBox, null, () => h('button', 'Go'));
      },
    });
    const box = mountApp(HostBox);
    await flushPromises();
    expect(box.q('.choy-button-box')).not.toBeNull();
    box.unmount();

    const emptyBox = mountApp(
      defineComponent({
        setup() {
          return () => h(ChoyButtonBox, null, () => null);
        },
      }),
    );
    await flushPromises();
    expect(emptyBox.q('.choy-button-box')).toBeNull();
    emptyBox.unmount();

    const stat = await mountField(ChoyStatInfo, { label: 'Roles', value: 3 });
    expect(stat.q('[data-test=o-stat]')).not.toBeNull();
    stat.unmount();

    stubSfc(OVColumn as any, {
      setup: ((_props: any, { slots }: any) => {
        return () =>
          h(
            'div',
            {
              'data-test': 'o-vcolumn',
              'data-has-slot': slots.default ? '1' : '0',
            },
            slots.default?.({
              row: { Id: '1' },
              column: {},
              $index: 0,
              store: fakeStore,
            }),
          );
      }) as any,
    });
    const ColHost = defineComponent({
      setup() {
        return () =>
          h(ChoyVColumn as any, null, {
            default: (slotProps: any) =>
              h('span', { 'data-test': 'cell', 'data-id': String(slotProps?.row?.Id ?? '') }),
          });
      },
    });
    const col = mountApp(ColHost);
    await flushPromises();
    expect(col.q('[data-test=o-vcolumn]')).not.toBeNull();
    expect(col.q('[data-test=o-vcolumn]')?.getAttribute('data-has-slot')).toBe('1');
    expect(col.q('[data-test=cell]')?.getAttribute('data-id')).toBe('1');
    col.unmount();

    // No consumer default slot → do not forward an empty slot to OVColumn.
    const bare = mountApp(ChoyVColumn as any, { props: { label: 'Name' } });
    await flushPromises();
    expect(bare.q('[data-test=o-vcolumn]')?.getAttribute('data-has-slot')).toBe('0');
    bare.unmount();
    restoreSfc(OVColumn as any);
    stubHost(OVColumn as any, 'o-vcolumn');
  });

  test('field chrome forwards attrs when inheritAttrs is false', async () => {
    const w = await mountField(ChoyVarcharField, {
      label: 'Name',
      'data-test': 'varchar-chrome',
    });
    const root = w.q('[data-anchor="choy.varchar-field"]') as HTMLElement | null;
    expect(root?.getAttribute('data-test')).toBe('varchar-chrome');
    w.unmount();
  });
});
