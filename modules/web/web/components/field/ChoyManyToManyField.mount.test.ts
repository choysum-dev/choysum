// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, defineComponent, h, nextTick, ref } from 'vue';
import { ChoyMessage } from '../../composables/useChoyMessage';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import FieldBase from './FieldBase.vue';
import ChoyManyToManyField from './ChoyManyToManyField.vue';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';

const searchExpose = {
  selectedItems: [] as any[],
};

const SearchListStub = defineComponent({
  name: 'SearchListStub',
  setup(_, { expose }) {
    expose(searchExpose);
    return () => h('div', { class: 'search-list-stub', 'data-test': 'search-list' });
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
  const items = ref(opts?.items ? opts.items.map((x: any) => ({ ...x })) : []);
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
    prop: opts?.prop || 'TagIds',
    meta: (opts?.meta || { type: 'ManyToMany', relationModel: 'demo.Tag' }) as any,
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
    }),
    asView: () => ({ fieldValue: () => fieldValue }) as any,
  } as any;
  return { binding, items, insertItem, removeItemAt };
}

const origWarn = ChoyMessage.warning;

function installStubs(mode: 'edit' | 'display' | 'both' = 'edit') {
  stubSfc(FieldBase as any, {
    name: 'FieldBase',
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
    setup(props: any, { emit, slots }: any) {
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
        else if (/delete|删除/.test(lower)) testId = 'row-delete';
        else if (/add|添加/.test(lower)) testId = 'add-row';
        return h(
          'button',
          {
            type: 'button',
            class: 'btn',
            'data-test': testId,
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
      tableHeight: { type: Number, default: undefined },
      store: { type: Object, default: undefined },
      baseIndex: { type: Number, default: undefined },
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
          },
          [
            slots.default?.(),
            ...(props.data || []).map((_: any, i: number) =>
              h('div', { class: 'ov-row', 'data-index': String(i) }, [
                // Invoke Actions column default slot with $index.
                ...(Array.isArray(slots.default?.())
                  ? []
                  : []),
              ])
            ),
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
        // Render one actions cell so Delete is clickable.
        if (slots.default) {
          nodes.push(h('div', { class: 'ov-actions-cell' }, slots.default({ $index: 0, row: {} })));
        }
        return h('div', { class: 'ov-column-wrap' }, nodes);
      };
    },
  });
}

describe('ChoyManyToManyField mount coverage', () => {
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
    restoreSfc(FieldBase as any);
    restoreSfc(ChoyDialog as any);
    restoreSfc(ChoyDialogContent as any);
    restoreSfc(ChoyDialogTitle as any);
    restoreSfc(ChoyButton as any);
    restoreSfc(ChoyViewScope as any);
    restoreSfc(ChoyTableHost as any);
    restoreSfc(ChoyTableColumn as any);
  });

  function mountField(props: Record<string, unknown>) {
    return mountApp(ChoyManyToManyField as any, {
      props: { renderMode: 'form', showIndex: true, ...props },
      provide: { lastOnchangeResult },
    });
  }

  test('hydrates hidden __rowKey and renders edit table height', async () => {
    const row = { Id: 't1', Name: 'Tag1', __rowKey: 'seed-1' };
    const enumerableKey = { Id: 't2', Name: 'Tag2' };
    (enumerableKey as any).__rowKey = 'enum-seed';
    const { binding, items } = makeBinding({
      items: [row, enumerableKey, null],
      relationStore: { fullModelName: 'demo.Tag' },
    });
    const m = mountField({ binding, searchList: SearchListStub, minTableHeight: 100, maxTableHeight: 500 });
    await flushPromises();
    await nextTick();
    expect(m.q('.ov-table-stub')?.getAttribute('data-rows')).toBe('3');
    expect(Object.prototype.propertyIsEnumerable.call(items.value[0], '__rowKey')).toBe(false);
    expect(String((items.value[0] as any).__rowKey)).toBe('seed-1');
    expect(m.q('[data-test="add-row"]')).toBeTruthy();
    m.unmount();
  });

  test('allowRowEdit true still renders add/delete actions', async () => {
    const { binding } = makeBinding({
      items: [{ Id: '1' }],
      relationStore: { fullModelName: 'demo.Tag' },
    });
    const m = mountField({ binding, allowRowEdit: true, searchList: SearchListStub });
    await nextTick();
    expect(m.q('.ov-table-stub')).toBeTruthy();
    expect(m.q('[data-test="add-row"]')).toBeTruthy();
    m.unmount();
  });

  test('display mode renders table without add actions', async () => {
    restoreSfc(FieldBase as any);
    installStubs('display');
    const { binding } = makeBinding({
      items: [{ Id: '1' }, { Id: '2' }],
      isEditMode: false,
      relationStore: { fullModelName: 'demo.Tag' },
    });
    const m = mountField({ binding, searchList: SearchListStub });
    await nextTick();
    expect(m.q('.ov-table-stub')?.getAttribute('data-rows')).toBe('2');
    expect(m.q('[data-test="add-row"]')).toBeFalsy();
    m.unmount();
  });

  test('openPicker warns without relationStore and opens dialog when resolved', async () => {
    const missing = makeBinding({ items: [], relationStore: undefined, meta: { type: 'ManyToMany' } });
    const m1 = mountField({ binding: missing.binding, searchList: SearchListStub });
    await nextTick();
    m1.click('[data-test="add-row"]');
    await nextTick();
    expect(msgWarn.calls.length).toBe(1);
    expect(m1.q('.dialog')?.getAttribute('data-open')).toBe('0');
    m1.unmount();

    const { binding } = makeBinding({
      items: [{ Id: 'keep' }],
      relationStore: { fullModelName: 'demo.Tag' },
    });
    const m2 = mountField({
      binding,
      searchList: SearchListStub,
      searchViewTitle: 'Pick tags',
      searchViewWidth: 800,
      condition: ['Active', '=', true],
    });
    await nextTick();
    m2.click('[data-test="add-row"]');
    await nextTick();
    expect(m2.q('.dialog')?.getAttribute('data-open')).toBe('1');
    expect(m2.q('.search-list-stub')).toBeTruthy();
    expect((m2.q('.dialog-title')?.textContent || '').includes('Pick tags')).toBe(true);
    m2.click('[data-test="dialog-cancel"]');
    await nextTick();
    expect(m2.q('.dialog')?.getAttribute('data-open')).toBe('0');
    m2.unmount();
  });

  test('confirmAdd inserts new rows, skips duplicates, and unwraps record wrappers', async () => {
    const { binding, items, insertItem } = makeBinding({
      items: [{ Id: 'exist', Name: 'E' }],
      relationStore: { fullModelName: 'demo.Tag' },
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
      condition: [['Active', '=', true]],
    });
    await nextTick();
    m.click('[data-test="add-row"]');
    await nextTick();

    // Empty selection closes.
    searchExpose.selectedItems = [];
    m.click('[data-test="dialog-ok"]');
    await flushPromises();
    expect(insertItem.calls.length).toBe(0);

    m.click('[data-test="add-row"]');
    await nextTick();
    // Duplicate only.
    searchExpose.selectedItems = [{ Id: 'exist', Name: 'E' }];
    m.click('[data-test="dialog-ok"]');
    await flushPromises();
    expect(insertItem.calls.length).toBe(0);

    m.click('[data-test="add-row"]');
    await nextTick();
    searchExpose.selectedItems = [
      { kind: 'record', payload: { Id: 'new1', Name: 'N1' } },
      { type: 'record', record: { Id: 'new2', Name: 'N2' } },
      { value: [{ Id: 'ignored-wrap' }] },
      { Id: 'exist', Name: 'dup' },
    ];
    // selectedItems as ref-shaped
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
    expect(items.value.map((r: any) => r.Id)).toEqual(['exist', 'new1', 'new2']);
    expect(Object.prototype.propertyIsEnumerable.call(items.value[1], '__rowKey')).toBe(false);
    m.unmount();
  });

  test('onRemove deletes row via actions column', async () => {
    const { binding, removeItemAt } = makeBinding({
      items: [{ Id: 'a' }, { Id: 'b' }],
      relationStore: { fullModelName: 'demo.Tag' },
    });
    const m = mountField({ binding, searchList: SearchListStub });
    await nextTick();
    const del = m.q('[data-test="row-delete"]') as HTMLElement | null;
    expect(del).toBeTruthy();
    del!.click();
    expect(removeItemAt.calls.length).toBe(1);
    expect(removeItemAt.calls[0]?.[0]).toBe(0);
    m.unmount();
  });

  test('relationStore falls back to createStoreByModel via targetModel', async () => {
    restoreFactory = replaceStoreFactory('demo.FallbackTag', () => ({
      fullModelName: 'demo.FallbackTag',
      NameSearch: fnRecorder(async () => []),
    }));
    const { binding } = makeBinding({
      items: [],
      relationStore: undefined,
      meta: { type: 'ManyToMany' },
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

  test('createStoreByModel failure warns and leaves picker closed', async () => {
    const warn = console.warn;
    console.warn = () => {};
    try {
      const { binding } = makeBinding({
        items: [],
        relationStore: undefined,
        meta: { type: 'ManyToMany' },
      });
      const m = mountField({
        binding,
        searchList: SearchListStub,
        targetModel: 'demo.MissingModel.That.Fails',
      });
      await nextTick();
      m.click('[data-test="add-row"]');
      await nextTick();
      expect(msgWarn.calls.length).toBe(1);
      expect(m.q('.dialog')?.getAttribute('data-open')).toBe('0');
      m.unmount();
    } finally {
      console.warn = warn;
    }
  });

  test('list-cell display shows count summary instead of table', async () => {
    restoreSfc(FieldBase as any);
    installStubs('display');
    const { binding } = makeBinding({
      items: [{ Id: 'a' }, { Id: 'b' }],
      isEditMode: false,
    });
    (binding.env as any).isForm = false;
    const m = mountField({ binding });
    await nextTick();
    const summary = m.q('[data-testid="choy-m2m-summary"]');
    expect(summary).toBeTruthy();
    expect(summary?.textContent?.trim()).toBe('2 records');
    expect(m.q('.ov-table-stub')).toBeNull();
    m.unmount();

    const empty = makeBinding({ items: [], isEditMode: false });
    (empty.binding.env as any).isForm = false;
    const em = mountField({ binding: empty.binding });
    await nextTick();
    expect(em.q('[data-testid="choy-m2m-summary"]')?.textContent).toBe('—');
    expect(em.q('.ov-table-stub')).toBeNull();
    em.unmount();
  });
});
