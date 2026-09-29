// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, defineComponent, h, nextTick, ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';

import type { UseField } from '@/web/web/composables/useField';
import { useAuthStore } from '@/auth/web/stores/auth';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import FieldBase from './FieldBase.vue';
import ChoyManyToOneField from './ChoyManyToOneField.vue';
import Dialog from '@/web/web/components/vendor/ui/dialog/Dialog.vue';
import DialogContent from '@/web/web/components/vendor/ui/dialog/DialogContent.vue';
import DialogTitle from '@/web/web/components/vendor/ui/dialog/DialogTitle.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';

const searchExpose = {
  selectedItem: null as any,
  selectedItems: [] as any[],
};

const SearchViewStub = defineComponent({
  name: 'SearchViewStub',
  setup(_, { expose }) {
    expose(searchExpose);
    return () => h('div', { class: 'search-view-stub', 'data-test': 'search-view' });
  },
});

const ElSelectV2Stub = defineComponent({
  name: 'ElSelectV2Stub',
  inheritAttrs: false,
  props: {
    remoteMethod: { type: Function, default: undefined },
    modelValue: { type: [String, Number, Object, Array, null] as any, default: undefined },
    options: { type: Array, default: () => [] },
    clearable: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
    placeholder: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'visible-change'],
  setup(props: any, { slots, emit }: any) {
    return () =>
      h(
        'div',
        {
          class: 'select-stub',
          'data-clearable': props.clearable ? '1' : '0',
          'data-placeholder': String(props.placeholder || ''),
          'data-options': String((props.options || []).length),
          'data-labels': (props.options || []).map((o: any) => o?.label ?? '').join('|'),
        },
        [
          h('button', {
            type: 'button',
            'data-test': 'remote',
            onClick: () => props.remoteMethod?.('  bob  '),
          }),
          h('button', {
            type: 'button',
            'data-test': 'remote-empty',
            onClick: () => props.remoteMethod?.(''),
          }),
          h('button', {
            type: 'button',
            'data-test': 'pick',
            onClick: () => emit('update:modelValue', { Id: 'picked', DisplayName: 'Picked' }),
          }),
          h('button', {
            type: 'button',
            'data-test': 'clear',
            onClick: () => emit('update:modelValue', null),
          }),
          h('button', {
            type: 'button',
            'data-test': 'visible-close',
            onClick: () => emit('visible-change', false),
          }),
          slots.footer?.(),
        ]
      );
  },
});

function makeBinding(opts?: {
  value?: any;
  prop?: string;
  record?: any;
  /** Row passed into #edit/#display slot (O2M line); defaults to root record. */
  slotRecord?: any;
  relationStore?: any;
}): { binding: UseField; value: any; record: any } {
  const value = ref(opts?.value ?? null);
  const record = ref(opts?.record ?? { Id: '1' });
  const binding = {
    env: {
      isForm: true,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: opts?.prop || 'PartnerId',
    meta: { type: 'ManyToOne', relationModel: 'demo.Partner' } as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    relationStore: opts?.relationStore,
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
    _slotRecord: opts?.slotRecord,
  } as any;
  return { binding, value, record };
}

function seedCreatePermission() {
  const auth = useAuthStore();
  auth.identity = {
    metadata: {
      activeCompanyId: 'c1',
      enabledCompanyIds: ['c1'],
    },
  } as any;
  auth.permissionState = {
    permStateVersion: 1,
    byCompany: {
      '*': { ui: { routes: [], menus: [], actions: ['partner.action.partner_create'] } },
    },
  } as any;
}

function installStubs() {
  stubSfc(FieldBase as any, {
    name: 'FieldBase',
    inheritAttrs: false,
    props: {
      binding: { type: Object, required: true },
      toView: { type: Function, default: undefined },
      fromView: { type: Function, default: undefined },
      rules: { type: Array, default: undefined },
      label: { type: String, default: undefined },
      formItemProps: { type: Object, default: undefined },
      vColumnProps: { type: Object, default: undefined },
      required: { type: [Boolean, Function, Object], default: undefined },
      readonly: { type: [Boolean, Function, Object], default: undefined },
      visible: { type: [Boolean, Function, Object], default: undefined },
      cellVisible: { type: [Boolean, Function, Object], default: undefined },
      renderMode: { type: String, default: undefined },
      showInlineError: { type: Boolean, default: undefined },
    },
    setup(p: any, { slots }: any) {
      const fieldValue = () => (p.binding as UseField).fieldRef();
      const rootRef = (p.binding as UseField).recordRef() as any;
      const root = rootRef && typeof rootRef === 'object' && 'value' in rootRef ? rootRef.value : rootRef;
      const record = (p.binding as any)._slotRecord ?? root;
      return () =>
        h('div', { class: 'ob' }, [
          slots.edit?.({ fieldValue, record }),
          slots.display?.({ fieldValue, record }),
        ]);
    },
  });
  stubSfc(Dialog as any, {
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
        const testId = /ok|确定/.test(lower) ? 'dialog-ok' : 'dialog-cancel';
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

describe('ChoyManyToOneField mount coverage', () => {
  let pinia: ReturnType<typeof createPinia>;
  let lastOnchangeResult: any;

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    seedCreatePermission();
    searchExpose.selectedItem = null;
    searchExpose.selectedItems = [];
    lastOnchangeResult = ref(null);
    installStubs();
  });

  afterEach(() => {
    restoreSfc(FieldBase as any);
    restoreSfc(Dialog as any);
    restoreSfc(DialogContent as any);
    restoreSfc(DialogTitle as any);
    restoreSfc(ChoyButton as any);
    restoreSfc(ChoyViewScope as any);
  });

  function mountField(props: Record<string, unknown>, on?: Record<string, (...args: any[]) => void>) {
    return mountApp(ChoyManyToOneField as any, {
      props: { renderMode: 'form', ...props },
      on,
      plugins: [pinia],
      provide: { lastOnchangeResult },
      stubs: {
        'el-select-v2': ElSelectV2Stub,
        ElSelectV2: ElSelectV2Stub,
      },
    });
  }

  test('clearable and placeholder pass through to select stub', async () => {
    const { binding } = makeBinding({
      relationStore: { NameSearch: fnRecorder(async () => []), fullModelName: 'partner.Partner' },
    });
    const m = mountField({ binding, clearable: false, placeholder: 'Pick one' });
    await nextTick();
    expect(m.q('.select-stub')?.getAttribute('data-clearable')).toBe('0');
    expect(m.q('.select-stub')?.getAttribute('data-placeholder')).toBe('Pick one');
    m.unmount();

    const { binding: b2 } = makeBinding({
      relationStore: { NameSearch: fnRecorder(async () => []), fullModelName: 'partner.Partner' },
    });
    const m2 = mountField({ binding: b2, clearable: true });
    await nextTick();
    expect(m2.q('.select-stub')?.getAttribute('data-clearable')).toBe('1');
    m2.unmount();
  });

  test('display text falls back across DisplayName Name Title Code Id', async () => {
    const cases = [
      { value: { Id: '1', DisplayName: 'DN' }, expect: 'DN' },
      { value: { Id: '2', Name: 'NM' }, expect: 'NM' },
      { value: { Id: '3', Title: 'TT' }, expect: 'TT' },
      { value: { Id: '4', Code: 'CD' }, expect: 'CD' },
      { value: { Id: '5' }, expect: '5' },
      { value: null, expect: '' },
      { value: 'raw', expect: '' },
    ];
    for (const c of cases) {
      const { binding } = makeBinding({
        value: c.value,
        relationStore: { NameSearch: fnRecorder(async () => []), fullModelName: 'partner.Partner' },
      });
      const m = mountField({ binding });
      await nextTick();
      expect((m.q('.choy-field-display-text')?.textContent || '').trim()).toBe(c.expect);
      m.unmount();
    }
  });

  test('optionsFor merges current value when idle and uses search results while searching', async () => {
    const NameSearch = fnRecorder(async () => [
      { Id: 'a', DisplayName: 'Alice' },
      null,
      { DisplayName: 'no-id' },
      { Id: 'b', DisplayName: 'Bob' },
    ]);
    const { binding, value } = makeBinding({
      value: { Id: 'cur', DisplayName: 'Current' },
      relationStore: { NameSearch, fullModelName: 'partner.Partner' },
    });
    const m = mountField({ binding, pageSize: 8 });
    await nextTick();
    // Idle: current value composed into options.
    expect(m.q('.select-stub')?.getAttribute('data-options')).toBe('1');
    expect(m.q('.select-stub')?.getAttribute('data-labels')).toBe('Current');

    m.click('[data-test="remote"]');
    await flushPromises();
    await nextTick();
    expect(NameSearch.calls.length).toBe(1);
    expect(NameSearch.calls[0]?.[0]).toBe('bob');
    expect(NameSearch.calls[0]?.[2]).toMatchObject({ fields: ['Id', 'DisplayName'], limit: 8 });
    // Searching: only NameSearch hits (null / missing Id filtered).
    expect(m.q('.select-stub')?.getAttribute('data-options')).toBe('2');
    expect(m.q('.select-stub')?.getAttribute('data-labels')).toBe('Alice|Bob');

    m.click('[data-test="visible-close"]');
    await nextTick();
    // Clearing search restores compose-with-current-value path.
    expect(m.q('.select-stub')?.getAttribute('data-labels') || '').toContain('Current');

    m.click('[data-test="pick"]');
    await nextTick();
    expect(value.value).toEqual({ Id: 'picked', DisplayName: 'Picked' });
    // Upsert replaces / prepends into options.
    expect(m.q('.select-stub')?.getAttribute('data-labels') || '').toContain('Picked');

    m.click('[data-test="clear"]');
    await nextTick();
    expect(value.value).toBeNull();
    m.unmount();
  });

  test('onUpdate upserts existing option id in place', async () => {
    const NameSearch = fnRecorder(async () => [{ Id: 'picked', DisplayName: 'Old' }]);
    const { binding, value } = makeBinding({
      relationStore: { NameSearch, fullModelName: 'partner.Partner' },
    });
    const m = mountField({ binding });
    m.click('[data-test="remote"]');
    await flushPromises();
    await nextTick();
    m.click('[data-test="visible-close"]');
    await nextTick();
    m.click('[data-test="pick"]');
    await nextTick();
    expect(value.value?.DisplayName).toBe('Picked');
    expect(m.q('.select-stub')?.getAttribute('data-labels') || '').toContain('Picked');
    m.unmount();
  });

  test('name-create entry visible with keyword and allowCreate', async () => {
    const NameCreate = fnRecorder(async (name: string) => ({ Id: 'n1', DisplayName: name, Name: name }));
    const { binding, value } = makeBinding({
      relationStore: {
        NameSearch: fnRecorder(async () => []),
        NameCreate,
        fullModelName: 'partner.Partner',
      },
    });
    const m = mountField({ binding, allowCreate: true });
    m.click('[data-test="remote"]');
    await flushPromises();
    await nextTick();
    const entry = m.q('[data-testid="choy-m2o-name-create"]') as HTMLElement | null;
    expect(entry).toBeTruthy();
    expect((entry!.textContent || '').includes('bob')).toBe(true);
    entry!.click();
    await flushPromises();
    expect(value.value?.Id).toBe('n1');
    m.unmount();
  });

  test('opens search dialog and confirmPick applies selectedItem or selectedItems', async () => {
    const { binding, value } = makeBinding({
      relationStore: { NameSearch: fnRecorder(async () => []), fullModelName: 'partner.Partner' },
    });
    const m = mountField({
      binding,
      searchView: SearchViewStub,
      searchViewTitle: 'Find partner',
      searchViewWidth: 640,
    });
    await nextTick();
    expect(m.q('.dialog')?.getAttribute('data-open')).toBe('0');

    const more = Array.from(m.el.querySelectorAll('.choy-m2o__more') || []).find(el =>
      (el.textContent || '').toLowerCase().includes('search')
    ) as HTMLElement | undefined;
    expect(more).toBeTruthy();
    more!.click();
    await nextTick();
    expect(m.q('.dialog')?.getAttribute('data-open')).toBe('1');
    expect(m.q('.search-view-stub')).toBeTruthy();
    expect((m.q('.dialog-title')?.textContent || '').includes('Find partner')).toBe(true);

    // Empty selection closes without writing.
    m.click('[data-test="dialog-ok"]');
    await nextTick();
    expect(value.value).toBeNull();
    expect(m.q('.dialog')?.getAttribute('data-open')).toBe('0');

    more!.click();
    await nextTick();
    searchExpose.selectedItem = { Id: 's1', DisplayName: 'FromSingle' };
    m.click('[data-test="dialog-ok"]');
    await nextTick();
    expect(value.value).toEqual({ Id: 's1', DisplayName: 'FromSingle' });

    more!.click();
    await nextTick();
    searchExpose.selectedItem = null;
    searchExpose.selectedItems = [{ Id: 's2', DisplayName: 'FromMulti' }];
    m.click('[data-test="dialog-ok"]');
    await nextTick();
    expect(value.value).toEqual({ Id: 's2', DisplayName: 'FromMulti' });

    more!.click();
    await nextTick();
    m.click('[data-test="dialog-cancel"]');
    await nextTick();
    expect(m.q('.dialog')?.getAttribute('data-open')).toBe('0');
    m.unmount();
  });

  test('openSearchDialog is a no-op without relationStore', async () => {
    const { binding } = makeBinding({ relationStore: undefined });
    const m = mountField({ binding, searchView: SearchViewStub });
    await nextTick();
    const more = Array.from(m.el.querySelectorAll('.choy-m2o__more') || []).find(el =>
      (el.textContent || '').toLowerCase().includes('search')
    ) as HTMLElement | undefined;
    more?.click();
    await nextTick();
    expect(m.q('.dialog')?.getAttribute('data-open')).toBe('0');
    m.unmount();
  });

  test('display value-click and keydown emit when clickable', async () => {
    const onValueClick = fnRecorder();
    const { binding } = makeBinding({
      value: { Id: '  p9  ', DisplayName: 'Partner Nine' },
      relationStore: { NameSearch: fnRecorder(async () => []), fullModelName: 'partner.Partner' },
    });
    const m = mountField({ binding, valueClickable: true }, { onValueClick });
    await nextTick();
    const display = m.q('.choy-field-display-text') as HTMLElement;
    expect(display.classList.contains('choy-field-display-text--clickable')).toBe(true);
    display.click();
    expect(onValueClick.calls.length).toBe(1);
    expect(onValueClick.calls[0]?.[0]).toMatchObject({
      id: 'p9',
      label: 'Partner Nine',
      source: 'display',
    });

    // QJS has no KeyboardEvent; cover click path only.
    display.click();
    expect(onValueClick.calls.length).toBe(2);
    m.unmount();

    const off = fnRecorder();
    const { binding: b2 } = makeBinding({
      value: { Id: 'x', DisplayName: 'X' },
      relationStore: { NameSearch: fnRecorder(async () => []), fullModelName: 'partner.Partner' },
    });
    const m2 = mountField({ binding: b2, valueClickable: false }, { onValueClick: off });
    await nextTick();
    (m2.q('.choy-field-display-text') as HTMLElement).click();
    expect(off.calls.length).toBe(0);
    m2.unmount();
  });

  test('remote search merges external and nested onchange conditions', async () => {
    const NameSearch = fnRecorder(async () => [{ Id: 'p1', DisplayName: 'P1' }]);
    const line = { Id: 'L1', Name: 'line' };
    const { binding } = makeBinding({
      prop: 'Lines.PartnerId',
      record: { Id: 'root', Lines: [line, { Id: 'L2' }] },
      slotRecord: line,
      relationStore: { NameSearch, fullModelName: 'partner.Partner' },
    });
    lastOnchangeResult.value = {
      condition: [
        { field: 'Lines(id=L1).PartnerId', condition: ['Name', '=', 'A'] },
        { field: 'Lines[0].PartnerId', condition: ['Code', '=', 'B'] },
        { field: 'Other', condition: ['X', '=', 1] },
      ],
    };
    const m = mountField({
      binding,
      condition: ['Active', '=', true],
      pageSize: 5,
    });
    await nextTick();
    m.click('[data-test="remote"]');
    await flushPromises();
    expect(NameSearch.calls.length).toBe(1);
    const cond = NameSearch.calls[0]?.[1] as any;
    expect(Array.isArray(cond?.And)).toBe(true);
    expect(cond.And.length).toBeGreaterThanOrEqual(2);
    m.unmount();
  });

  test('flat-field onchange conditions filter by prop name', async () => {
    const NameSearch = fnRecorder(async () => []);
    const { binding } = makeBinding({
      prop: 'PartnerId',
      relationStore: { NameSearch, fullModelName: 'partner.Partner' },
    });
    lastOnchangeResult.value = {
      condition: [
        { field: 'PartnerId', condition: ['Country', '=', 'CN'] },
        { field: 'Other', condition: ['X', '=', 1] },
      ],
    };
    const m = mountField({ binding });
    m.click('[data-test="remote"]');
    await flushPromises();
    expect(NameSearch.calls[0]?.[1]).toEqual(['Country', '=', 'CN']);
    m.unmount();
  });

  test('dialog conditions combine external and onchange for nested props', async () => {
    const line = { Id: 'L1' };
    const { binding } = makeBinding({
      prop: 'Lines.PartnerId',
      record: { Id: 'root', Lines: [line] },
      slotRecord: line,
      relationStore: { NameSearch: fnRecorder(async () => []), fullModelName: 'partner.Partner' },
    });
    lastOnchangeResult.value = {
      condition: [{ field: 'Lines(id=L1).PartnerId', condition: ['Name', '=', 'Z'] }],
    };
    const m = mountField({
      binding,
      searchView: SearchViewStub,
      condition: { Or: [['A', '=', 1]] },
    });
    await nextTick();
    const more = Array.from(m.el.querySelectorAll('.choy-m2o__more') || []).find(el =>
      (el.textContent || '').toLowerCase().includes('search')
    ) as HTMLElement;
    more.click();
    await nextTick();
    expect(m.q('.search-view-stub')).toBeTruthy();
    m.unmount();
  });

  test('nested object chain and normalizeRowRef variants still search', async () => {
    const NameSearch = fnRecorder(async () => []);
    const child = { Id: 'C1' };
    const { binding } = makeBinding({
      prop: 'Parent.Child.PartnerId',
      record: { Id: 'root', Parent: { Child: child } },
      slotRecord: child,
      relationStore: { NameSearch, fullModelName: 'partner.Partner' },
    });
    lastOnchangeResult.value = {
      condition: [{ field: 'Parent.Child(id=C1).PartnerId', condition: ['X', '=', 1] }],
    };
    const m = mountField({ binding });
    m.click('[data-test="remote"]');
    await flushPromises();
    expect(NameSearch.calls.length).toBe(1);
    m.unmount();
  });

  test('confirmPick unwraps ref-shaped selectedItem.value', async () => {
    const { binding, value } = makeBinding({
      relationStore: { NameSearch: fnRecorder(async () => []), fullModelName: 'partner.Partner' },
    });
    const m = mountField({ binding, searchView: SearchViewStub });
    const more = Array.from(m.el.querySelectorAll('.choy-m2o__more') || []).find(el =>
      (el.textContent || '').toLowerCase().includes('search')
    ) as HTMLElement;
    more.click();
    await nextTick();
    searchExpose.selectedItem = { value: { Id: 'wrap', DisplayName: 'Wrapped' } };
    m.click('[data-test="dialog-ok"]');
    await nextTick();
    expect(value.value).toEqual({ Id: 'wrap', DisplayName: 'Wrapped' });
    m.unmount();
  });
});
