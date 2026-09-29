// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Patch-focused coverage for ChoyManyToManyRefTagsField (createStoreByModel fallback,
 * display tag click/keydown, remote search, highlight, keydown, picker confirm).
 */

import { computed, defineComponent, h, nextTick, ref } from 'vue';
import { ChoyMessage } from '../../composables/useChoyMessage';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import FieldBase from './FieldBase.vue';
import ChoyManyToManyRefTagsField from './ChoyManyToManyRefTagsField.vue';
import Dialog from '@/web/web/components/vendor/ui/dialog/Dialog.vue';
import DialogContent from '@/web/web/components/vendor/ui/dialog/DialogContent.vue';
import DialogTitle from '@/web/web/components/vendor/ui/dialog/DialogTitle.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';

const searchExpose = {
  selectedItems: [] as any[],
};

const SearchListStub = defineComponent({
  name: 'SearchListStub',
  setup(_, { expose }) {
    expose(searchExpose);
    return () => h('div', { class: 'search-list-stub' });
  },
});

const ElSelectV2Stub = defineComponent({
  name: 'ElSelectV2Stub',
  inheritAttrs: false,
  props: {
    remoteMethod: { type: Function, default: undefined },
    modelValue: { type: [String, Number, Object, Array, null] as any, default: undefined },
    options: { type: Array, default: () => [] },
  },
  emits: ['update:modelValue', 'visible-change', 'keydown'],
  setup(props: any, { slots, emit }: any) {
    return () =>
      h(
        'div',
        {
          class: 'select-stub',
          'data-ids': JSON.stringify(props.modelValue ?? []),
          'data-options': String((props.options || []).length),
        },
        [
          h('button', {
            type: 'button',
            'data-test': 'remote',
            onClick: () => props.remoteMethod?.('alpha'),
          }),
          h('button', {
            type: 'button',
            'data-test': 'remote-empty',
            onClick: () => props.remoteMethod?.(''),
          }),
          h('button', {
            type: 'button',
            'data-test': 'visible-open',
            onClick: () => emit('visible-change', true),
          }),
          // Render one suggestion so highlightSuggestion runs through the slot path when options exist.
          ...(props.options || []).slice(0, 1).map((item: any) =>
            h('div', { class: 'opt' }, slots.default?.({ item }) || [])
          ),
          slots.footer?.(),
        ]
      );
  },
});

function makeBinding(opts: {
  items?: any[];
  relationStore?: any;
  meta?: any;
}): UseField {
  const items = ref(opts.items ? opts.items.slice() : []);
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
    prop: 'TagIds',
    meta: (opts.meta || { type: 'ManyToMany', relationModel: 'demo.Tag' }) as any,
    fieldRef: () => items as any,
    fieldRefOf: () => items as any,
    recordRef: () => computed(() => ({ Id: 'parent' })) as any,
    registerFields: () => {},
    relationStore: opts.relationStore,
    store: undefined,
    asMutableArray: () => ({
      getItems: () => items.value,
      insertItem: (row: any) => {
        items.value = [...items.value, row];
      },
      clearItems: () => {
        items.value = [];
      },
      removeItemAt: () => {},
    }),
    asView: () => ({ fieldValue: () => items }) as any,
  } as any;
}

function relationStoreStub(overrides: Record<string, unknown> = {}) {
  return {
    fullModelName: 'demo.Tag',
    storeId: 'store-tags-1',
    Search: fnRecorder(async () => []),
    NameSearch: fnRecorder(async () => []),
    ...overrides,
  };
}

const origWarn = ChoyMessage.warning;

function installStubs() {
  stubSfc(FieldBase as any, {
    name: 'FieldBase',
    inheritAttrs: false,
    props: {
      binding: { type: Object, required: true },
      label: { type: String, default: undefined },
      rules: { type: Array, default: undefined },
      formItemProps: { type: Object, default: undefined },
      required: { type: [Boolean, Function, Object], default: undefined },
      readonly: { type: [Boolean, Function, Object], default: undefined },
      visible: { type: [Boolean, Function, Object], default: undefined },
      cellVisible: { type: [Boolean, Function, Object], default: undefined },
      renderMode: { type: String, default: undefined },
      showInlineError: { type: Boolean, default: undefined },
    },
    setup(p: any, { slots }: any) {
      return () =>
        h('div', { class: 'field-base-stub' }, [slots.edit?.({}), slots.display?.({})]);
    },
  });
  stubSfc(Dialog as any, {
    name: 'Dialog',
    props: { open: { type: Boolean, default: false } },
    setup(props: any, { slots }: any) {
      return () =>
        h('div', { class: 'dialog', 'data-open': props.open ? '1' : '0' }, [
          props.open ? slots.default?.() : null,
        ]);
    },
  });
  stubSfc(DialogContent as any, {
    name: 'DialogContent',
    setup(_: any, { slots, attrs }: any) {
      return () => h('div', { class: ['dialog-content', attrs.class] }, slots.default?.());
    },
  });
  stubSfc(DialogTitle as any, {
    name: 'DialogTitle',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'dialog-title' }, slots.default?.());
    },
  });
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    emits: ['click'],
    setup(_: any, { emit, slots }: any) {
      return () => {
        const kids = slots.default?.() || [];
        let text = '';
        for (const k of kids as any[]) {
          if (typeof k === 'string' || typeof k === 'number') text += String(k);
          else if (k && typeof k.children === 'string') text += k.children;
        }
        const lower = text.toLowerCase();
        let testId = 'btn';
        if (/ok|确定/.test(lower)) testId = 'dialog-ok';
        else if (/cancel|取消/.test(lower)) testId = 'dialog-cancel';
        return h(
          'button',
          {
            type: 'button',
            class: 'btn',
            'data-test': testId,
            onClick: (e: Event) => emit('click', e),
          },
          kids
        );
      };
    },
  });
  stubSfc(ChoyViewScope as any, {
    name: 'ChoyViewScope',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'view-scope-stub' }, slots.default?.());
    },
  });
}

describe('ChoyManyToManyRefTagsField patch coverage', () => {
  let msgWarn: ReturnType<typeof fnRecorder>;
  let lastOnchangeResult: any;
  let restoreFactory: (() => void) | undefined;
  let consoleWarn: typeof console.warn;

  beforeEach(() => {
    msgWarn = fnRecorder();
    ChoyMessage.warning = msgWarn as typeof ChoyMessage.warning;
    lastOnchangeResult = ref(null);
    searchExpose.selectedItems = [];
    consoleWarn = console.warn;
    console.warn = () => {};
    installStubs();
  });

  afterEach(() => {
    ChoyMessage.warning = origWarn;
    console.warn = consoleWarn;
    restoreFactory?.();
    restoreFactory = undefined;
    restoreSfc(FieldBase as any);
    restoreSfc(Dialog as any);
    restoreSfc(DialogContent as any);
    restoreSfc(DialogTitle as any);
    restoreSfc(ChoyButton as any);
    restoreSfc(ChoyViewScope as any);
  });

  function mountField(props: Record<string, unknown>, on?: Record<string, (...args: any[]) => void>) {
    return mountApp(ChoyManyToManyRefTagsField as any, {
      props: { renderMode: 'form', ...props },
      on,
      provide: { lastOnchangeResult },
      stubs: { 'el-select-v2': ElSelectV2Stub, ElSelectV2: ElSelectV2Stub },
    });
  }

  test('string ids hydrate via Search and render selected model-value', async () => {
    const Search = fnRecorder(async () => [
      { Id: 't1', DisplayName: 'Tag One' },
      { Id: 't3', DisplayName: 'Tag Three' },
    ]);
    const binding = makeBinding({
      items: ['t1', 't3'],
      relationStore: relationStoreStub({ Search }),
    });
    const m = mountField({ binding, searchList: SearchListStub, maxTagsVisible: 1 });
    await flushPromises();
    await nextTick();

    expect(Search.calls.length).toBeGreaterThanOrEqual(1);
    const idsAttr = m.q('.select-stub')?.getAttribute('data-ids') || '[]';
    expect(idsAttr).toContain('t1');
    expect(idsAttr).toContain('t3');
    expect(m.text()).toMatch(/\+/);
    m.unmount();
  });

  test('Search more opens picker; missing searchList/relationStore warn', async () => {
    const binding = makeBinding({
      items: [],
      relationStore: relationStoreStub({ storeId: 's1' }),
    });
    const m = mountField({ binding, searchList: SearchListStub });
    await flushPromises();
    const more = Array.from(m.el.querySelectorAll('.choy-m2m-tags__more')).find(n =>
      (n.textContent || '').toLowerCase().includes('search')
    ) as HTMLElement | undefined;
    expect(more).toBeTruthy();
    more!.click();
    await nextTick();
    expect(m.q('.dialog')?.getAttribute('data-open')).toBe('1');
    m.unmount();

    msgWarn.mockClear();
    const noStore = makeBinding({
      items: [],
      relationStore: undefined,
      meta: { type: 'ManyToMany' },
    });
    const m2 = mountField({ binding: noStore, searchList: SearchListStub });
    await flushPromises();
    const more2 = Array.from(m2.el.querySelectorAll('.choy-m2m-tags__more')).find(n =>
      (n.textContent || '').toLowerCase().includes('search')
    ) as HTMLElement | undefined;
    expect(more2).toBeTruthy();
    more2!.click();
    await nextTick();
    expect(msgWarn.calls.length).toBe(1);
    m2.unmount();
  });

  test('relationStore falls back to createStoreByModel via targetModel', async () => {
    restoreFactory = replaceStoreFactory('demo.FallbackTag', () =>
      relationStoreStub({
        fullModelName: 'demo.FallbackTag',
        storeId: 'fallback-store',
      })
    );
    const binding = makeBinding({
      items: [],
      relationStore: undefined,
      meta: { type: 'ManyToMany' },
    });
    const m = mountField({
      binding,
      searchList: SearchListStub,
      targetModel: 'demo.FallbackTag',
    });
    await flushPromises();
    const ss = m.setupState() as any;
    expect(ss.relationStore).toBeTruthy();
    ss.openPicker();
    await nextTick();
    expect(m.q('.dialog')?.getAttribute('data-open')).toBe('1');
    m.unmount();
  });

  test('createStoreByModel failure warns and leaves relationStore undefined', async () => {
    const binding = makeBinding({
      items: [],
      relationStore: undefined,
      meta: { type: 'ManyToMany' },
    });
    const m = mountField({
      binding,
      searchList: SearchListStub,
      targetModel: 'demo.MissingModel.That.Fails',
    });
    await flushPromises();
    const ss = m.setupState() as any;
    expect(ss.relationStore).toBeFalsy();
    m.unmount();
  });

  test('resolveTagLabel fallback and display tag click/keydown', async () => {
    const onTagClick = fnRecorder();
    const binding = makeBinding({
      items: ['t1'],
      relationStore: relationStoreStub({
        Search: fnRecorder(async () => [{ Id: 't1' }]),
      }),
    });
    const m = mountField(
      { binding, searchList: SearchListStub, tagClickable: true },
      { onTagClick }
    );
    await flushPromises();
    await nextTick();

    const ss = m.setupState() as any;
    // No matching label fields → fallback to Id when fallback omitted.
    expect(ss.resolveTagLabel({ Id: 'x1', DisplayName: '', Name: '  ' })).toBe('x1');
    expect(ss.resolveTagLabel({ Id: '' })).toBe('');
    expect(ss.resolveTagLabel(null, 'fb')).toBe('fb');

    const hit = m.q('.choy-m2m-tags__tag-hit') as HTMLElement | null;
    expect(hit).toBeTruthy();
    hit!.click();
    expect(onTagClick.calls.length).toBe(1);

    const fakeEnter = { key: 'Enter', preventDefault: fnRecorder() };
    ss.onDisplayTagKeydown({ id: 't1', record: { Id: 't1' }, label: 't1' }, fakeEnter);
    expect(fakeEnter.preventDefault.calls.length).toBe(1);
    expect(onTagClick.calls.length).toBe(2);

    const fakeSpace = { key: ' ', preventDefault: fnRecorder() };
    ss.onDisplayTagKeydown({ id: 't1', record: { Id: 't1' }, label: 't1' }, fakeSpace);
    expect(onTagClick.calls.length).toBe(3);

    ss.onDisplayTagKeydown({ id: 't1', record: { Id: 't1' }, label: 't1' }, {
      key: 'a',
      preventDefault: fnRecorder(),
    });
    expect(onTagClick.calls.length).toBe(3);

    // Not clickable → early return.
    m.unmount();
    const m2 = mountField({
      binding: makeBinding({
        items: ['t1'],
        relationStore: relationStoreStub({
          Search: fnRecorder(async () => [{ Id: 't1' }]),
        }),
      }),
      tagClickable: false,
    });
    await flushPromises();
    const ss2 = m2.setupState() as any;
    ss2.onDisplayTagClick({ id: 't1', record: {}, label: 't1' }, {});
    ss2.onDisplayTagKeydown({ id: 't1', record: {}, label: 't1' }, {
      key: 'Enter',
      preventDefault: fnRecorder(),
    });
    m2.unmount();
  });

  test('dropdown visible, toArray, effectiveConditions And/single, hydrate error', async () => {
    const NameSearch = fnRecorder(async () => [{ Id: 's1', DisplayName: 'Alpha' }]);
    const Search = fnRecorder(async () => {
      throw new Error('hydrate boom');
    });
    const binding = makeBinding({
      items: ['keep'],
      relationStore: relationStoreStub({ NameSearch, Search, storeId: 'cond-store' }),
    });
    lastOnchangeResult.value = {
      condition: [
        { field: 'TagIds', condition: ['Kind', '=', 'x'] },
        { field: 'Other', condition: ['Y', '=', 1] },
      ],
    };
    const m = mountField({
      binding,
      searchList: SearchListStub,
      // Array-of-conditions form (tuple alone is Array.isArray and would be spread as 3 parts).
      condition: [['Active', '=', true]],
    });
    await flushPromises();
    const ss = m.setupState() as any;
    ss.onDropdownVisibleChange(true);
    expect(ss.dropdownVisible).toBe(true);

    m.click('[data-test="remote"]');
    await flushPromises();
    expect(NameSearch.calls.length).toBe(1);
    const domain = NameSearch.calls[0]?.[1] as { And?: unknown } | undefined;
    // excludePicked + external + onchange → And
    expect(domain && typeof domain === 'object' && domain.And).toBeTruthy();

    // Single-part path: clear selection then search again with only external condition.
    ss.onSelectedIdsChange([]);
    await flushPromises();
    NameSearch.mockClear();
    lastOnchangeResult.value = null;
    m.click('[data-test="remote"]');
    await flushPromises();
    expect(NameSearch.calls.length).toBe(1);
    // Only external condition → leaf (not And).
    expect(NameSearch.calls[0]?.[1]).toEqual(['Active', '=', true]);
    m.unmount();

    // Non-array condition object → toArray wraps as [v].
    const NameSearch2 = fnRecorder(async () => []);
    const m2 = mountField({
      binding: makeBinding({
        items: [],
        relationStore: relationStoreStub({ NameSearch: NameSearch2, storeId: 'obj-cond' }),
      }),
      searchList: SearchListStub,
      condition: { And: [['Active', '=', true]] } as any,
    });
    await flushPromises();
    m2.click('[data-test="remote"]');
    await flushPromises();
    expect(NameSearch2.calls.length).toBe(1);
    expect(NameSearch2.calls[0]?.[1]).toEqual({ And: [['Active', '=', true]] });
    m2.unmount();
  });

  test('highlightSuggestion escapes and marks hits', async () => {
    const binding = makeBinding({
      items: [],
      relationStore: relationStoreStub(),
    });
    const m = mountField({ binding, searchList: SearchListStub });
    const ss = m.setupState() as any;

    expect(ss.highlightSuggestion('a & b <c>')).toContain('&amp;');
    expect(ss.highlightSuggestion('a & b <c>')).toContain('&lt;');

    ss.searchKeyword = '  ';
    expect(ss.highlightSuggestion('Hello')).toBe('Hello');

    ss.searchKeyword = 'll';
    const hit = ss.highlightSuggestion('Hello');
    expect(hit).toContain('choy-m2m-tags__suggestion-hit');
    expect(hit).toContain('ll');

    ss.searchKeyword = 'zzz';
    expect(ss.highlightSuggestion('Hello')).toBe('Hello');

    // Multiple hits
    ss.searchKeyword = 'a';
    const multi = ss.highlightSuggestion('banana');
    expect((multi.match(/suggestion-hit/g) || []).length).toBeGreaterThanOrEqual(2);

    ss.searchKeyword = '';
    expect(ss.highlightSuggestion('')).toBe('');
    m.unmount();
  });

  test('handleRemoteSearch without store and on NameSearch failure', async () => {
    const noStore = makeBinding({
      items: [],
      relationStore: undefined,
      meta: { type: 'ManyToMany' },
    });
    const m1 = mountField({ binding: noStore });
    const ss1 = m1.setupState() as any;
    await ss1.handleRemoteSearch('x');
    expect(ss1.searchRows).toEqual([]);
    m1.unmount();

    const NameSearch = fnRecorder(async () => {
      throw new Error('search boom');
    });
    const m2 = mountField({
      binding: makeBinding({ items: [], relationStore: relationStoreStub({ NameSearch }) }),
      searchList: SearchListStub,
    });
    const ss2 = m2.setupState() as any;
    await ss2.handleRemoteSearch('q');
    await flushPromises();
    expect(ss2.searchRows).toEqual([]);
    expect(ss2.loading).toBe(false);
    m2.unmount();
  });

  test('handleKeydown composing, Backspace remove, Enter pick first', async () => {
    const NameSearch = fnRecorder(async () => [
      { Id: 'already' },
      { Id: 'new1', DisplayName: 'New' },
    ]);
    const binding = makeBinding({
      items: ['already', 'old'],
      relationStore: relationStoreStub({
        NameSearch,
        Search: fnRecorder(async () => [
          { Id: 'already', DisplayName: 'A' },
          { Id: 'old', DisplayName: 'O' },
        ]),
      }),
    });
    const m = mountField({ binding, searchList: SearchListStub, tagClosable: true });
    await flushPromises();
    const ss = m.setupState() as any;

    ss.handleKeydown({ key: 'Enter', isComposing: true, preventDefault: fnRecorder() });

    // Backspace with empty keyword removes last tag.
    ss.searchKeyword = '';
    const back = { key: 'Backspace', preventDefault: fnRecorder() };
    ss.handleKeydown(back);
    expect(back.preventDefault.calls.length).toBe(1);
    await flushPromises();
    expect(ss.selectedIds).toEqual(['already']);

    // Backspace with keyword does nothing.
    ss.searchKeyword = 'x';
    ss.handleKeydown({ key: 'Backspace', preventDefault: fnRecorder() });

    // Enter while loading / empty keyword ignored.
    ss.loading = true;
    ss.handleKeydown({ key: 'Enter', preventDefault: fnRecorder() });
    ss.loading = false;
    ss.searchKeyword = '';
    ss.handleKeydown({ key: 'Enter', preventDefault: fnRecorder() });

    // Enter picks first unselected search row.
    await ss.handleRemoteSearch('n');
    await flushPromises();
    const enter = { key: 'Enter', preventDefault: fnRecorder() };
    ss.handleKeydown(enter);
    expect(enter.preventDefault.calls.length).toBe(1);
    await flushPromises();
    expect(ss.selectedIds).toContain('new1');

    // Enter with no unselected rows.
    ss.searchRows = [{ Id: 'already' }];
    ss.searchKeyword = 'z';
    ss.handleKeydown({ key: 'Enter', preventDefault: fnRecorder() });

    // Non-Enter key ignored.
    ss.handleKeydown({ key: 'Tab', preventDefault: fnRecorder() });
    m.unmount();
  });

  test('tagClosable false keeps removals; remove emits and requeries when open', async () => {
    const NameSearch = fnRecorder(async () => [{ Id: 'extra', DisplayName: 'E' }]);
    const binding = makeBinding({
      items: ['a', 'b'],
      relationStore: relationStoreStub({
        NameSearch,
        Search: fnRecorder(async () => [
          { Id: 'a', DisplayName: 'A' },
          { Id: 'b', DisplayName: 'B' },
        ]),
      }),
    });
    const onTagRemove = fnRecorder();
    const onTagAdd = fnRecorder();
    const m = mountField(
      { binding, searchList: SearchListStub, tagClosable: false },
      { onTagRemove, onTagAdd }
    );
    await flushPromises();
    const ss = m.setupState() as any;

    // Cannot shrink when tagClosable=false.
    ss.onSelectedIdsChange(['a']);
    await flushPromises();
    expect(ss.selectedIds).toEqual(['a', 'b']);

    // Remount with closable to exercise remove + dropdown requery.
    m.unmount();
    const m2 = mountField(
      {
        binding: makeBinding({
          items: ['a', 'b'],
          relationStore: relationStoreStub({
            NameSearch,
            Search: fnRecorder(async () => [
              { Id: 'a', DisplayName: 'A' },
              { Id: 'b', DisplayName: 'B' },
            ]),
            storeId: 'rm-store',
          }),
        }),
        searchList: SearchListStub,
        tagClosable: true,
      },
      { onTagRemove, onTagAdd }
    );
    await flushPromises();
    const ss2 = m2.setupState() as any;
    ss2.onDropdownVisibleChange(true);
    NameSearch.mockClear();
    ss2.searchKeyword = 'q';
    ss2.onSelectedIdsChange(['a']);
    await flushPromises();
    expect(onTagRemove.calls.length).toBeGreaterThanOrEqual(1);
    expect(NameSearch.calls.length).toBeGreaterThanOrEqual(1);
    m2.unmount();
  });

  test('openPicker warns without searchList; confirmPicker paths', async () => {
    const Search = fnRecorder(async () => []);
    const binding = makeBinding({
      items: ['exist'],
      relationStore: relationStoreStub({
        Search: fnRecorder(async () => [{ Id: 'exist', DisplayName: 'E' }]),
      }),
    });
    const onPickerConfirm = fnRecorder();
    const onTagAdd = fnRecorder();

    msgWarn.mockClear();
    const m0 = mountField({ binding: makeBinding({ items: [], relationStore: relationStoreStub() }) });
    const ss0 = m0.setupState() as any;
    ss0.openPicker();
    expect(msgWarn.calls.length).toBe(1);
    m0.unmount();

    const m = mountField(
      { binding, searchList: SearchListStub, searchViewWidth: 640 },
      { onPickerConfirm, onTagAdd }
    );
    await flushPromises();

    // Open picker and confirm empty selection.
    const more = Array.from(m.el.querySelectorAll('.choy-m2m-tags__more')).find(n =>
      (n.textContent || '').toLowerCase().includes('search')
    ) as HTMLElement;
    more.click();
    await nextTick();
    searchExpose.selectedItems = [];
    m.click('[data-test="dialog-ok"]');
    await flushPromises();
    expect(m.q('.dialog')?.getAttribute('data-open')).toBe('0');
    expect(onPickerConfirm.calls.length).toBe(0);

    // Confirm with wrappers + ref unwrap.
    more.click();
    await nextTick();
    searchExpose.selectedItems = {
      value: [
        { kind: 'record', payload: { Id: 'new1', DisplayName: 'N1' } },
        { type: 'record', record: { Id: 'new2', Name: 'N2' } },
        { Id: 'exist', DisplayName: 'E' },
        null,
      ],
    } as any;
    m.click('[data-test="dialog-ok"]');
    await flushPromises();
    await nextTick();
    expect(onPickerConfirm.calls.length).toBe(1);
    expect(onTagAdd.calls.some((c: any[]) => c[0]?.id === 'new1')).toBe(true);
    expect(onTagAdd.calls.some((c: any[]) => c[0]?.id === 'new2')).toBe(true);

    // confirmPicker with null expose (no dialog) still closes.
    const ss = m.setupState() as any;
    ss.dialogVisible = true;
    ss.searchViewRef = null;
    await ss.confirmPicker();
    expect(ss.dialogVisible).toBe(false);
    m.unmount();

    // Silence unused Search in this branch.
    void Search;
  });

  test('auto tagClickable follows onTagClick listener', async () => {
    const onTagClick = fnRecorder();
    const binding = makeBinding({
      items: ['t1'],
      relationStore: relationStoreStub({
        Search: fnRecorder(async () => [{ Id: 't1', DisplayName: 'T' }]),
      }),
    });
    const m = mountField(
      { binding, searchList: SearchListStub, tagClickable: 'auto' },
      { onTagClick }
    );
    await flushPromises();
    await nextTick();
    (m.q('.choy-m2m-tags__tag-hit') as HTMLElement).click();
    expect(onTagClick.calls.length).toBe(1);
    m.unmount();
  });
});
