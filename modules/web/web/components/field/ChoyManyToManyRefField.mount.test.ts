// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, defineComponent, h, nextTick, ref } from 'vue';
import { ChoyMessage } from '../../composables/useChoyMessage';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import ChoyFieldBase from './ChoyFieldBase.vue';
import ChoyManyToManyRefField from './ChoyManyToManyRefField.vue';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';

const searchExpose = {
  selectedItems: [] as any[],
};

type CapturedSearchListProps = {
  hostPageTitle?: boolean;
  selectionMode?: string;
  forcedCondition?: unknown;
  relationConditionSource?: { model: string; field: string };
};

const lastSearchListProps = ref<CapturedSearchListProps | null>(null);

const SearchListStub = defineComponent({
  name: 'SearchListStub',
  inheritAttrs: false,
  props: {
    hostPageTitle: { type: Boolean, default: true },
    showSelectionColumn: { type: Boolean, default: true },
    selectionMode: { type: String, default: 'multiple' },
    forcedCondition: { type: [Object, Array] as any, default: undefined },
    relationConditionSource: { type: Object as any, default: undefined },
  },
  setup(props, { expose }) {
    expose(searchExpose);
    lastSearchListProps.value = {
      hostPageTitle: props.hostPageTitle,
      selectionMode: props.selectionMode,
      forcedCondition: props.forcedCondition,
      relationConditionSource: props.relationConditionSource,
    };
    return () =>
      h('div', {
        class: 'search-list-stub',
        'data-test': 'search-list',
        'data-host-page-title': props.hostPageTitle ? '1' : '0',
        'data-selection-mode': String(props.selectionMode || ''),
      });
  },
});

function makeBinding(opts?: {
  items?: any[];
  prop?: string;
  isEditMode?: boolean;
  relationStore?: any;
  meta?: any;
}): {
  binding: UseField;
  items: any;
  insertItem: ReturnType<typeof fnRecorder>;
  removeItemAt: ReturnType<typeof fnRecorder>;
} {
  const items = ref(opts?.items ? [...opts.items] : []);
  const insertItem = fnRecorder((row: any) => {
    items.value = [...items.value, row];
  });
  const removeItemAt = fnRecorder((i: number) => {
    items.value = items.value.filter((_: any, idx: number) => idx !== i);
  });
  const fieldValue = computed({
    get: () => items.value,
    set: (v: any) => {
      items.value = Array.isArray(v) ? v : [];
    },
  });
  const binding = {
    env: {
      isForm: true,
      isEditMode: opts?.isEditMode !== false,
      viewMode: opts?.isEditMode === false ? 'display' : 'edit',
      fieldPrefix: null,
    },
    prop: opts?.prop || 'TagRefIds',
    meta: (opts?.meta || { type: 'ManyToManyRef', relationModel: 'demo.Tag' }) as any,
    fieldRef: () => fieldValue as any,
    fieldRefOf: () => fieldValue as any,
    recordRef: () => computed(() => ({ Id: 'parent' })) as any,
    registerFields: () => {},
    relationStore: opts?.relationStore,
    store: undefined,
    asMutableArray: () => ({
      getItems: () => items.value,
      insertItem,
      removeItemAt,
      clearItems: () => {
        items.value = [];
      },
    }),
    asView: () => ({ fieldValue: () => fieldValue }) as any,
  } as any;
  return { binding, items, insertItem, removeItemAt };
}

const origWarn = ChoyMessage.warning;

function installStubs(mode: 'edit' | 'display' | 'both' = 'edit') {
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
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
      const edit = () => slots.edit?.({});
      const display = () => slots.display?.({});
      return () =>
        h(
          'div',
          { class: 'field-base-stub' },
          mode === 'both' ? [edit(), display()] : mode === 'display' ? [display()] : [edit()]
        );
    },
  });
  stubSfc(ChoyDialog as any, {
    name: 'Dialog',
    props: { open: { type: Boolean, default: false } },
    emits: ['update:open'],
    setup(props: any, { slots }: any) {
      return () =>
        h('div', { class: 'dialog', 'data-open': props.open ? '1' : '0' }, [
          props.open ? slots.default?.() : null,
        ]);
    },
  });
  stubSfc(ChoyDialogContent as any, {
    name: 'DialogContent',
    setup(_: any, { slots, attrs }: any) {
      return () => h('div', { class: ['dialog-content', attrs.class] }, slots.default?.());
    },
  });
  stubSfc(ChoyDialogTitle as any, {
    name: 'DialogTitle',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'dialog-title' }, slots.default?.());
    },
  });
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    props: {
      type: { type: String, default: 'button' },
      variant: { type: String, default: 'default' },
      disabled: { type: Boolean, default: false },
      size: { type: String, default: undefined },
    },
    emits: ['click'],
    setup(props: any, { emit, slots, attrs }: any) {
      return () => {
        const kids = slots.default?.() || [];
        let text = '';
        for (const k of kids as any[]) {
          if (typeof k === 'string' || typeof k === 'number') text += String(k);
          else if (k && typeof k.children === 'string') text += k.children;
        }
        const aria = String(attrs['aria-label'] || attrs.ariaLabel || '');
        const lower = `${text} ${aria}`.toLowerCase();
        let testId = String(attrs['data-testid'] || 'btn');
        if (/ok|确定/.test(lower)) testId = 'dialog-ok';
        else if (/cancel|取消/.test(lower)) testId = 'dialog-cancel';
        else if (/delete|删除|remove/.test(lower) || testId === 'choy-m2m-ref-row-remove') testId = 'row-delete';
        else if (/add|添加|line/.test(lower) || testId === 'choy-m2m-ref-add-row') testId = 'add-row';
        return h(
          'button',
          {
            type: 'button',
            class: 'btn',
            'data-test': testId,
            'data-testid': attrs['data-testid'] || testId,
            'data-variant': props.variant || '',
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
  stubSfc(ChoyTableHost as any, {
    name: 'ChoyTableHost',
    props: {
      data: { type: Array, default: () => [] },
      rowKey: { type: String, default: undefined },
      rowHeight: { type: Number, default: undefined },
      headerHeight: { type: Number, default: undefined },
      footerHeight: { type: Number, default: undefined },
      tableHeight: { type: Number, default: undefined },
      store: { type: Object, default: undefined },
      baseIndex: { type: Number, default: undefined },
      virtualize: { type: Boolean, default: true },
    },
    setup(props: any, { slots, expose }: any) {
      const scrollToRow = fnRecorder();
      expose({ scrollToRow });
      return () =>
        h(
          'div',
          {
            class: 'ov-table-stub',
            'data-rows': String((props.data || []).length),
            'data-height': String(props.tableHeight ?? ''),
            'data-footer': String(props.footerHeight ?? ''),
            'data-first-id': String((props.data || [])[0]?.Id ?? ''),
            'data-first-name': String((props.data || [])[0]?.DisplayName ?? (props.data || [])[0]?.Name ?? ''),
          },
          [
            slots.default?.(),
            !(props.data || []).length ? slots.empty?.() : null,
            props.footerHeight
              ? h('div', { class: 'ov-footer-stub', 'data-testid': 'choy-m2m-ref-table-footer' }, slots.footer?.())
              : null,
          ]
        );
    },
  });
  stubSfc(ChoyTableColumn as any, {
    name: 'ChoyTableColumn',
    props: {
      type: { type: String, default: undefined },
      label: { type: String, default: undefined },
      width: { type: [Number, String], default: undefined },
      vColumnProps: { type: Object, default: undefined },
    },
    setup(props: any, { slots }: any) {
      return () => {
        const nodes: any[] = [h('div', { class: 'ov-column-stub', 'data-label': props.label || props.type || '' })];
        if (slots.default) {
          nodes.push(h('div', { class: 'ov-actions-cell' }, slots.default({ $index: 0, row: {} })));
        }
        return h('div', { class: 'ov-column-wrap' }, nodes);
      };
    },
  });
}

describe('ChoyManyToManyRefField mount coverage', () => {
  let lastOnchangeResult: any;
  let msgWarn: ReturnType<typeof fnRecorder>;
  let restoreFactory: (() => void) | undefined;

  beforeEach(() => {
    lastOnchangeResult = ref(null);
    msgWarn = fnRecorder();
    ChoyMessage.warning = msgWarn as typeof ChoyMessage.warning;
    searchExpose.selectedItems = [];
    installStubs('edit');
  });

  afterEach(() => {
    ChoyMessage.warning = origWarn;
    restoreFactory?.();
    restoreFactory = undefined;
    restoreSfc(ChoyFieldBase as any);
    restoreSfc(ChoyDialog as any);
    restoreSfc(ChoyDialogContent as any);
    restoreSfc(ChoyDialogTitle as any);
    restoreSfc(ChoyButton as any);
    restoreSfc(ChoyViewScope as any);
    restoreSfc(ChoyTableHost as any);
    restoreSfc(ChoyTableColumn as any);
  });

  function mountField(props: Record<string, unknown>) {
    return mountApp(ChoyManyToManyRefField as any, {
      props: { renderMode: 'form', showIndex: true, ...props },
      provide: { lastOnchangeResult },
    });
  }

  test('hydrates string ids via Search and renders edit table rows', async () => {
    const Search = fnRecorder(async () => [
      { Id: 't1', DisplayName: 'Tag One' },
      { Id: 't2', DisplayName: 'Tag Two' },
    ]);
    const { binding } = makeBinding({
      items: ['t1', 't2'],
      relationStore: {
        fullModelName: 'demo.Tag',
        storeId: 'rel-hydrate',
        Search,
      },
    });
    const m = mountField({ binding, searchList: SearchListStub, minTableHeight: 100, maxTableHeight: 500 });
    await flushPromises();
    await nextTick();
    expect(Search.calls.length).toBeGreaterThanOrEqual(1);
    expect(m.q('.ov-table-stub')?.getAttribute('data-rows')).toBe('2');
    expect(m.q('.ov-table-stub')?.getAttribute('data-first-id')).toBe('t1');
    expect(m.q('.ov-table-stub')?.getAttribute('data-first-name')).toBe('Tag One');
    expect(m.q('[data-test="add-row"]')).toBeTruthy();
    m.unmount();
  });

  test('display mode renders table without add actions', async () => {
    restoreSfc(ChoyFieldBase as any);
    installStubs('display');
    const Search = fnRecorder(async () => [{ Id: '1', DisplayName: 'A' }, { Id: '2', DisplayName: 'B' }]);
    const { binding } = makeBinding({
      items: ['1', '2'],
      isEditMode: false,
      relationStore: { fullModelName: 'demo.Tag', storeId: 'rel-disp', Search },
    });
    const m = mountField({ binding, searchList: SearchListStub });
    await flushPromises();
    await nextTick();
    expect(m.q('.ov-table-stub')?.getAttribute('data-rows')).toBe('2');
    expect(m.q('[data-test="add-row"]')).toBeFalsy();
    m.unmount();
  });

  test('empty edit still mounts table host; footer Add-line is the empty affordance', async () => {
    const { binding } = makeBinding({
      items: [],
      relationStore: { fullModelName: 'demo.Tag', storeId: 'rel-empty' },
    });
    const m = mountField({ binding, searchList: SearchListStub });
    await nextTick();
    expect(m.q('.ov-table-stub')).toBeTruthy();
    expect(m.q('.ov-table-stub')?.getAttribute('data-rows')).toBe('0');
    expect(m.q('[data-testid="choy-m2m-ref-empty"]')).toBeFalsy();
    expect(m.q('[data-test="add-row"]')).toBeTruthy();
    m.unmount();

    const m2 = mountField({
      binding: makeBinding({ items: [], relationStore: { fullModelName: 'demo.Tag' } }).binding,
    });
    await nextTick();
    expect(m2.q('[data-testid="choy-m2m-ref-add-unavailable"]')).toBeTruthy();
    expect(m2.q('[data-test="add-row"]')).toBeFalsy();
    m2.unmount();
  });

  test('openPicker warns without relationStore and opens dialog when resolved', async () => {
    const missing = makeBinding({ items: [], relationStore: undefined, meta: { type: 'ManyToManyRef' } });
    const m1 = mountField({ binding: missing.binding, searchList: SearchListStub });
    await nextTick();
    m1.click('[data-test="add-row"]');
    await nextTick();
    expect(msgWarn.calls.length).toBe(0);
    expect(m1.q('.dialog')?.getAttribute('data-open')).toBe('0');
    m1.unmount();

    const relationStore = {
      fullModelName: 'demo.Tag',
      storeId: 'rel-picker',
      state: {
        queryState: {
          keyword: 'stale',
          appliedFilters: [{ And: [['Name', 'ilike', 'x']] }],
          pagination: { limit: 40, offset: 80 },
        },
      },
      Search: fnRecorder(async () => []),
    };
    lastSearchListProps.value = null;
    const { binding } = makeBinding({
      items: ['keep'],
      relationStore,
    });
    const m2 = mountField({
      binding,
      searchList: SearchListStub,
      searchViewTitle: 'Pick tag refs',
      searchViewWidth: 800,
      condition: ['Active', '=', true],
      store: { fullModelName: 'sale.Order' },
    });
    await nextTick();
    m2.click('[data-test="add-row"]');
    await nextTick();
    expect(m2.q('.dialog')?.getAttribute('data-open')).toBe('1');
    expect(m2.q('.search-list-stub')).toBeTruthy();
    expect(m2.q('.search-list-stub')?.getAttribute('data-host-page-title')).toBe('0');
    expect(m2.q('.search-list-stub')?.getAttribute('data-selection-mode')).toBe('multiple');
    expect((m2.q('.dialog-title')?.textContent || '').includes('Pick tag refs')).toBe(true);
    expect(relationStore.state.queryState.keyword).toBeUndefined();
    expect(relationStore.state.queryState.appliedFilters).toEqual([]);
    expect(relationStore.state.queryState.pagination).toEqual({ limit: 40, offset: 0 });
    expect((lastSearchListProps.value as CapturedSearchListProps | null)?.relationConditionSource).toEqual({
      model: 'sale.Order',
      field: 'TagRefIds',
    });
    expect((lastSearchListProps.value as CapturedSearchListProps | null)?.forcedCondition).toEqual({
      And: [
        ['Id', 'not in', ['keep']],
        ['Active', '=', true],
      ],
    });
    m2.click('[data-test="dialog-cancel"]');
    await nextTick();
    expect(m2.q('.dialog')?.getAttribute('data-open')).toBe('0');
    m2.unmount();
  });

  test('confirmAdd inserts new string ids, skips duplicates, and unwraps record wrappers', async () => {
    const { binding, items, insertItem } = makeBinding({
      items: ['exist'],
      relationStore: {
        fullModelName: 'demo.Tag',
        storeId: 'rel-confirm',
        Search: fnRecorder(async () => [{ Id: 'exist', DisplayName: 'E' }]),
      },
    });
    lastOnchangeResult.value = {
      condition: [
        { field: 'TagRefIds', condition: ['Kind', '=', 'x'] },
        { field: 'Other', condition: ['Y', '=', 1] },
      ],
    };
    const m = mountField({
      binding,
      searchList: SearchListStub,
      condition: [['Active', '=', true]],
    });
    await flushPromises();
    await nextTick();
    m.click('[data-test="add-row"]');
    await nextTick();

    searchExpose.selectedItems = [];
    m.click('[data-test="dialog-ok"]');
    await flushPromises();
    expect(insertItem.calls.length).toBe(0);

    m.click('[data-test="add-row"]');
    await nextTick();
    searchExpose.selectedItems = [{ Id: 'exist', Name: 'E' }];
    m.click('[data-test="dialog-ok"]');
    await flushPromises();
    expect(insertItem.calls.length).toBe(0);

    m.click('[data-test="add-row"]');
    await nextTick();
    searchExpose.selectedItems = {
      value: [
        { kind: 'record', payload: { Id: 'new1', Name: 'N1' } },
        { type: 'record', record: { Id: 'new2', Name: 'N2' } },
        { Id: 'exist', Name: 'dup' },
      ],
    } as any;
    m.click('[data-test="dialog-ok"]');
    await flushPromises();
    await nextTick();
    expect(insertItem.calls.length).toBe(2);
    expect(items.value).toEqual(['exist', 'new1', 'new2']);
    expect(m.q('.ov-table-stub')?.getAttribute('data-rows')).toBe('3');
    m.unmount();
  });

  test('onRemove deletes row via actions column', async () => {
    const { binding, removeItemAt } = makeBinding({
      items: ['a', 'b'],
      relationStore: {
        fullModelName: 'demo.Tag',
        storeId: 'rel-remove',
        Search: fnRecorder(async () => [
          { Id: 'a', DisplayName: 'A' },
          { Id: 'b', DisplayName: 'B' },
        ]),
      },
    });
    const m = mountField({ binding, searchList: SearchListStub });
    await flushPromises();
    await nextTick();
    const del = m.q('[data-test="row-delete"]') as HTMLElement | null;
    expect(del).toBeTruthy();
    expect(del!.getAttribute('data-variant')).toBe('ghost');
    del!.click();
    expect(removeItemAt.calls.length).toBe(1);
    expect(removeItemAt.calls[0]?.[0]).toBe(0);
    m.unmount();
  });

  test('relationStore falls back to createStoreByModel via targetModel', async () => {
    restoreFactory = replaceStoreFactory('demo.FallbackTag', () => ({
      fullModelName: 'demo.FallbackTag',
      storeId: 'fallback-tag',
      Search: fnRecorder(async () => []),
      NameSearch: fnRecorder(async () => []),
    }));
    const { binding } = makeBinding({
      items: [],
      relationStore: undefined,
      meta: { type: 'ManyToManyRef' },
    });
    const m = mountField({
      binding,
      searchList: SearchListStub,
      targetModel: 'demo.FallbackTag',
    });
    await nextTick();
    m.click('[data-test="add-row"]');
    await nextTick();
    expect(m.q('.dialog')?.getAttribute('data-open')).toBe('1');
    m.unmount();
  });

  test('list-cell display shows count summary instead of table', async () => {
    restoreSfc(ChoyFieldBase as any);
    installStubs('display');
    const { binding } = makeBinding({
      items: ['a', 'b'],
      isEditMode: false,
    });
    (binding.env as any).isForm = false;
    const m = mountField({ binding });
    await nextTick();
    const summary = m.q('[data-testid="choy-m2m-ref-summary"]');
    expect(summary).toBeTruthy();
    expect(summary?.textContent?.trim()).toBe('2 records');
    expect(m.q('.ov-table-stub')).toBeNull();
    m.unmount();
  });
});
