// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, defineComponent, inject, ref, computed, type Component } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyPage from '../layout/ChoyPage.vue';
import ChoyFormView from '../view/ChoyFormView.vue';
import ChoyListView from '../view/ChoyListView.vue';
import ChoySearchView from '../view/ChoySearchView.vue';
import ListView from '../view/ChoyListView.vue';
import Search from '../view/search/Search.vue';
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
import ChoyIntField from './ChoyIntField.vue';
import ChoyBigintField from './ChoyBigintField.vue';
import ChoyDecimalField from './ChoyDecimalField.vue';
import ChoyNumberField from './ChoyNumberField.vue';
import ChoyJsonField from './ChoyJsonField.vue';
import ChoyImageField from './ChoyImageField.vue';
import ChoyVirtualField from './ChoyVirtualField.vue';
import ChoyManyToOneField from './ChoyManyToOneField.vue';
import ChoyManyToOneRefField from './ChoyManyToOneRefField.vue';
import ChoyManyToManyField from './ChoyManyToManyField.vue';
import ChoyManyToManyRefTagsField from './ChoyManyToManyRefTagsField.vue';
import ChoyManyToManyRefTreeField from './ChoyManyToManyRefTreeField.vue';
import ChoyOneToManyField from './ChoyOneToManyField.vue';
import ChoyOneToManyKanbanField from './ChoyOneToManyKanbanField.vue';
import FieldBase from './FieldBase.vue';

const fakeStore = { modelName: 'auth.User', meta: { fields: {} } } as any;

const STORE_FIELD_BASE_TEST_ID = 'store-field-base';

function stubStoreFieldBase() {
  stubSfc(FieldBase, {
    props: { binding: { type: Object, required: false }, label: String },
    setup: ((props: any) => {
      return () =>
        h('div', {
          'data-test': STORE_FIELD_BASE_TEST_ID,
          'data-prop': String(props.binding?.prop ?? ''),
        });
    }) as any,
  });
}

function stubHost(Comp: Component, testId: string) {
  stubSfc(Comp, {
    props: { store: null, prop: String },
    setup: ((props: any) => {
      return () =>
        h('div', {
          'data-test': testId,
          'data-prop': String(props.prop || ''),
        });
    }) as any,
  });
}

function installFieldBaseEditStub() {
  stubSfc(FieldBase as any, {
    props: { binding: { type: Object, required: false } },
    setup(p: any, { slots }: any) {
      const fieldValue = () => (p.binding as any).fieldRef();
      return () => h('div', { 'data-test': 'field-base-edit' }, slots.edit?.({ fieldValue }));
    },
  });
}

function makeFloatBinding(initial: number | null) {
  const value = ref<number | null>(initial);
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
    prop: 'Amount',
    meta: { type: 'float' } as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => ({})) as any,
    registerFields: () => {},
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
  };
}

function makeO2MKanbanBinding(items: Record<string, unknown>[]) {
  const list = ref(items.slice());
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
    prop: 'Contacts',
    meta: { type: 'oneToMany' } as any,
    fieldRef: () => list as any,
    fieldRefOf: () => list as any,
    recordRef: () => computed(() => ({ Id: '1' })) as any,
    registerFields: () => {},
    asMutableArray: () => ({
      getItems: () => list.value,
      insertItem: (row: Record<string, unknown>) => {
        list.value = [...list.value, row];
      },
      removeItemAt: () => {},
    }),
    store: undefined,
    asView: () => ({ fieldValue: () => list }) as any,
  };
}

describe('Choy store-mode field hosts', () => {
  const stubs: any[] = [
    FieldBase,
    ListView,
    ChoyFormView,
    ChoyListView,
    ChoySearchView,
    ChoyStatInfo,
    ChoyVColumn,
  ];

  beforeEach(() => {
    stubStoreFieldBase();
    stubHost(ChoyFormView as any, 'o-form');
    stubHost(ListView as any, 'o-list');
    stubHost(ChoyStatInfo as any, 'o-stat');
    stubHost(ChoyVColumn as any, 'o-vcolumn');
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

  test('scalar fields bind FieldBase when store+prop set', async () => {
    const merged: any[] = [
      ChoyVarcharField,
      ChoyTextField,
      ChoyBooleanField,
      ChoySelectionField,
      ChoyDatetimeField,
      ChoyDateField,
      ChoyJsonField,
      ChoyImageField,
    ];
    for (const Comp of merged) {
      const w = await mountField(Comp, { store: fakeStore, prop: 'Name' });
      const host = w.q(`[data-test=${STORE_FIELD_BASE_TEST_ID}]`);
      expect(host).not.toBeNull();
      expect(host?.getAttribute('data-prop')).toBe('Name');
      w.unmount();
    }
    const virtualW = await mountField(ChoyVirtualField, { store: fakeStore, prop: 'Name' });
    expect(virtualW.q(`[data-test=${STORE_FIELD_BASE_TEST_ID}]`)).toBeNull();
    virtualW.unmount();
    const o2mW = await mountField(ChoyOneToManyField, { store: fakeStore, prop: 'Contacts' });
    const o2mHost = o2mW.q(`[data-test=${STORE_FIELD_BASE_TEST_ID}]`);
    expect(o2mHost).not.toBeNull();
    expect(o2mHost?.getAttribute('data-prop')).toBe('Contacts');
    o2mW.unmount();
  });

  test('parallel numeric fields bind FieldBase when store+prop set', async () => {
    const numeric: Array<[Component, string]> = [
      [ChoyIntField, 'Padding'],
      [ChoyBigintField, 'NextNumber'],
      [ChoyDecimalField, 'Rate'],
      [ChoyNumberField, 'Factor'],
    ];
    for (const [Comp, prop] of numeric) {
      const w = await mountField(Comp, { store: fakeStore, prop });
      const host = w.q(`[data-test=${STORE_FIELD_BASE_TEST_ID}]`);
      expect(host).not.toBeNull();
      expect(host?.getAttribute('data-prop')).toBe(prop);
      w.unmount();
    }
  });

  test('DateField store mode binds FieldBase for the prop', async () => {
    const w = await mountField(ChoyDateField, {
      store: fakeStore,
      prop: 'Date',
    });
    const host = w.q(`[data-test=${STORE_FIELD_BASE_TEST_ID}]`);
    expect(host).not.toBeNull();
    expect(host?.getAttribute('data-prop')).toBe('Date');
    w.unmount();
  });

  test('ChoyBigintField edit input uses numeric inputmode', async () => {
    restoreSfc(FieldBase as any);
    const binding = makeFloatBinding(42);
    installFieldBaseEditStub();
    const w = mountApp(ChoyBigintField as any, {
      props: { binding, renderMode: 'form' },
    });
    await flushPromises();
    const input = w.q('input') as HTMLInputElement | null;
    expect(input?.getAttribute('inputmode')).toBe('numeric');
    w.unmount();
    restoreSfc(FieldBase as any);
    stubStoreFieldBase();
  });

  test('ChoyNumberField commits buffered float edits on blur', async () => {
    restoreSfc(FieldBase as any);
    const binding = makeFloatBinding(1);
    installFieldBaseEditStub();
    const w = mountApp(ChoyNumberField as any, {
      props: { binding, renderMode: 'form', bufferStrategy: 'live', commitOnBlur: true },
    });
    await flushPromises();
    const input = w.q('input') as HTMLInputElement | null;
    expect(input).not.toBeNull();
    input!.value = '3.5';
    input!.dispatchEvent(new Event('input', { bubbles: true }));
    input!.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.fieldRef().value).toBe(3.5);
    w.unmount();
    restoreSfc(FieldBase as any);
    stubStoreFieldBase();
  });

  test('parallel relation fields bind FieldBase when store+prop set', async () => {
    const relation: Array<[any, string]> = [
      [ChoyManyToOneField, 'CompanyId'],
      [ChoyManyToOneRefField, 'CurrencyId'],
      [ChoyManyToManyField, 'RoleIds'],
      [ChoyManyToManyRefTagsField, 'TagIds'],
      [ChoyManyToManyRefTreeField, 'CategoryIds'],
      [ChoyOneToManyKanbanField, 'Contacts'],
    ];
    for (const [Comp, prop] of relation) {
      const w = await mountField(Comp, { store: fakeStore, prop });
      const host = w.q(`[data-test=${STORE_FIELD_BASE_TEST_ID}]`);
      expect(host).not.toBeNull();
      expect(host?.getAttribute('data-prop')).toBe(prop);
      w.unmount();
    }
  });

  test('Form/List/Search store mode hosts store engines', async () => {
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

  test('Gallery form shell forwards attrs and named slot chrome paths', async () => {
    const GalleryFormShell = (await import('../../pages/GalleryFormShell.vue')).default;
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            GalleryFormShell,
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

  test('FormView store mode forwards named slots to FormView', async () => {
    stubSfc(ChoyFormView, {
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
              ChoyFormView as any,
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
      restoreSfc(ChoyFormView as any);
      stubHost(ChoyFormView as any, 'o-form');
    }
  });

  test('SearchView chrome submit paths', async () => {
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

    // Undeclared on* attrs forward via v-on once (keys stripped for toHandlers).
    const loadHits: unknown[] = [];
    stubSfc(ChoyFormView as any, {
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
    restoreSfc(ChoyFormView as any);
    stubHost(ChoyFormView as any, 'o-form');

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

  test('ChoyOneToManyKanbanField renders card title and subtitle from binding', async () => {
    restoreSfc(FieldBase as any);
    const binding = makeO2MKanbanBinding([
      { Id: '1', Name: 'Alpha', Email: 'n1' },
      { Name: '', Email: '' },
    ]);
    installFieldBaseEditStub();
    const w = mountApp(ChoyOneToManyKanbanField as any, {
      props: {
        binding,
        renderMode: 'form',
        cardTitleField: 'Name',
        cardSubtitleField: 'Email',
        editable: true,
      },
    });
    await flushPromises();
    expect(w.text()).toContain('Alpha');
    expect(w.text()).toContain('n1');
    w.unmount();
    restoreSfc(FieldBase as any);
    stubStoreFieldBase();
  });

  test('field prop under ChoyPage store hosts engines without explicit store prop', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyPage as any, { store: fakeStore, title: 'Page' }, () =>
            h(ChoyVarcharField as any, { prop: 'Name' }),
          );
      },
    });
    const w = mountApp(Host);
    await flushPromises();
    expect(w.q(`[data-test=${STORE_FIELD_BASE_TEST_ID}]`)).not.toBeNull();
    expect(w.q('[data-anchor="choy.varchar-field"]')).toBeNull();
    w.unmount();
  });

  test('store List/Search forward engine events and Search bind props', async () => {
    // ChoyListView is the store engine (no host unwrap). Store mode mounts the list itself.
    const list = await mountField(ChoyListView, {
      store: fakeStore,
      onSelectionChange: () => undefined,
    });
    expect(list.q('[data-test=o-list]')).not.toBeNull();
    list.unmount();

    const queries: Array<{ keyword: string }> = [];
    let seenBind: Record<string, unknown> = {};
    stubSfc(Search as any, {
      props: { store: null, placeholder: String },
      emits: ['query-update'],
      setup: ((props: any, { emit }: any) => {
        seenBind = {
          placeholder: props.placeholder,
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
    search.click('[data-test=emit-query]');
    await flushPromises();
    expect(queries.length).toBeGreaterThan(0);
    expect(queries[0]?.keyword).toBe('acme');
    expect((queries[0] as any)?.filters?.length).toBe(1);
    search.unmount();
    restoreSfc(Search as any);
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

    stubSfc(ChoyVColumn as any, {
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

    // No consumer default slot → do not forward an empty slot to VColumn.
    const bare = mountApp(ChoyVColumn as any, { props: { label: 'Name' } });
    await flushPromises();
    expect(bare.q('[data-test=o-vcolumn]')?.getAttribute('data-has-slot')).toBe('0');
    bare.unmount();
    restoreSfc(ChoyVColumn as any);
    stubHost(ChoyVColumn as any, 'o-vcolumn');
  });

});
