// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, reactive, ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { UseField } from '@/web/web/composables/useField';
import type { ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import OPropertiesField from './OPropertiesField.vue';
import {
  buildFullPropertiesMap,
  countSchemaMapIntersection,
  filterRenderablePropertyItems,
  normalizeSelectionOptions,
  propertiesFieldKey,
  propertyDatetimeFromPicker,
  propertyDatetimeToPicker,
  writePropertyValue,
} from './oproperties_helpers';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string, ...args: unknown[]) => (args.length ? `${msg}:${args.join(',')}` : msg),
    }),
  };
});

const useFieldMock = vi.hoisted(() => ({
  impl: null as null | ((...args: any[]) => any),
}));

vi.mock('@/web/web/composables/useField', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/composables/useField')>(
    '@/web/web/composables/useField'
  );
  return {
    ...actual,
    useField: (...args: any[]) => {
      if (useFieldMock.impl) return useFieldMock.impl(...args);
      return (actual as any).useField(...args);
    },
  };
});

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  const Control = (name: string, tag = 'input') =>
    defineComponent({
      name,
      props: {
        modelValue: { type: [String, Number, Boolean, Object], default: undefined },
        disabled: Boolean,
        type: String,
        clearable: Boolean,
        precision: Number,
        controls: Boolean,
        autosize: [Boolean, Object],
        valueFormat: String,
        id: String,
      },
      emits: ['update:modelValue'],
      setup(props, { emit, slots }) {
        return () =>
          h('div', { class: `stub-${name}` }, [
            h(tag, {
              class: `ctrl-${name}`,
              id: props.id,
              value: props.modelValue == null ? '' : String(props.modelValue),
              disabled: props.disabled || undefined,
              onInput: (e: Event) => {
                const raw = (e.target as HTMLInputElement).value;
                if (name === 'ElSwitch') emit('update:modelValue', raw === 'true' || raw === '1');
                else if (name === 'ElInputNumber')
                  emit('update:modelValue', raw === '' ? undefined : Number(raw));
                else emit('update:modelValue', raw);
              },
            }),
            slots.default?.(),
          ]);
      },
    });
  return {
    ...actual,
    ElInput: Control('ElInput', 'textarea'),
    ElInputNumber: Control('ElInputNumber'),
    ElSwitch: Control('ElSwitch'),
    ElSelect: Control('ElSelect'),
    ElOption: defineComponent({
      name: 'ElOption',
      props: { label: String, value: [String, Number] },
      setup: () => () => null,
    }),
    ElDatePicker: defineComponent({
      name: 'ElDatePicker',
      props: {
        modelValue: { type: [String, Number, Boolean, Object, Date], default: undefined },
        disabled: Boolean,
        type: String,
        valueFormat: String,
        id: String,
      },
      emits: ['update:modelValue'],
      setup(props, { emit }) {
        return () =>
          h('input', {
            class: 'ctrl-ElDatePicker',
            id: props.id,
            value:
              props.modelValue instanceof Date
                ? props.modelValue.toISOString()
                : props.modelValue == null
                  ? ''
                  : String(props.modelValue),
            disabled: props.disabled || undefined,
            'data-type': props.type,
            onInput: (e: Event) => {
              const raw = (e.target as HTMLInputElement).value;
              if (props.type === 'datetime') {
                emit('update:modelValue', raw ? new Date(raw) : null);
              } else {
                emit('update:modelValue', raw || null);
              }
            },
          });
      },
    }),
  };
});

function makeBinding(opts: {
  map?: Record<string, unknown> | null;
  record?: Record<string, unknown>;
  isForm?: boolean;
  prop?: string;
  store?: any;
  ResolveProperties?: ((...args: any[]) => Promise<any>) | false;
}): UseField & { __value: any; __store: any; __record: any } {
  const value = ref(opts.map === undefined ? {} : opts.map);
  const recordRef = ref(opts.record ?? { Id: '1', Properties: value.value });
  let store: any;
  if (Object.prototype.hasOwnProperty.call(opts, 'store')) {
    store = opts.store;
  } else if (opts.ResolveProperties === false) {
    store = {};
  } else {
    store = {
      ResolveProperties:
        opts.ResolveProperties ?? vi.fn(async () => [] as ResolvedPropertyItem[]),
    };
  }
  return {
    env: {
      isForm: opts.isForm !== false,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: opts.prop === undefined ? 'Properties' : opts.prop,
    meta: reactive({ type: 'properties' }) as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => recordRef.value) as any,
    registerFields: () => undefined,
    store: store as any,
    asView: () => ({ fieldValue: () => value }) as any,
    __value: value,
    __store: store,
    __record: recordRef,
  } as any;
}

const fieldBaseStub = defineComponent({
  name: 'OFieldBaseStub',
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
    renderMode: { type: String, default: 'auto' },
    showInlineError: { type: Boolean, default: false },
  },
  setup(props, { slots }) {
    const fieldValue = () => {
      const raw = (props.binding as UseField).fieldRef();
      const toView = props.toView as ((v: any) => any) | undefined;
      const fromView = props.fromView as ((v: any) => any) | undefined;
      return computed({
        get: () => (toView ? toView(raw.value) : raw.value),
        set: v => {
          raw.value = fromView ? fromView(v) : v;
        },
      });
    };
    return () =>
      h('div', { class: 'ob' }, [slots.edit?.({ fieldValue }), slots.display?.({ fieldValue })]);
  },
});

const schemaItems: ResolvedPropertyItem[] = [
  { name: 'active', type: 'boolean', string: 'Active', value: true },
  { name: 'code', type: 'char', string: 'Code', value: 'A1' },
  { name: 'qty', type: 'integer', string: 'Qty', value: 2 },
  { name: 'amount', type: 'float', string: 'Amount', value: 1.5 },
  { name: 'note', type: 'text', string: 'Note' },
  { name: 'day', type: 'date', string: 'Day', value: '2024-01-02T00:00:00Z' },
  {
    name: 'when',
    type: 'datetime',
    string: 'When',
    value: '2024-06-30T16:00:00.000Z',
  },
  {
    name: 'kind',
    type: 'selection',
    string: 'Kind',
    selection: [
      ['a', 'Alpha'],
      { value: 'b', label: 'Beta' },
    ],
  },
  { name: 'html_x', type: 'html', string: 'Bad' },
];

describe('oproperties_helpers', () => {
  it('resolves field keys for RPC and DOM ids', () => {
    expect(propertiesFieldKey('A', 'B')).toBe('A');
    expect(propertiesFieldKey('', 'B')).toBe('B');
    expect(propertiesFieldKey('', '', 'properties')).toBe('properties');
    expect(propertiesFieldKey(null, undefined, '')).toBe('');
  });

  it('filters unknown / empty items and normalizes selection options', () => {
    expect(filterRenderablePropertyItems(null as any)).toEqual({ renderable: [], skipped: [] });
    const { renderable, skipped } = filterRenderablePropertyItems([
      null as any,
      { name: '', type: 'char' },
      { name: 'ok', type: 'char' },
      { name: 'bad', type: 'html' },
      ...schemaItems,
    ]);
    expect(renderable.map(i => i.name)).toEqual([
      'ok',
      'active',
      'code',
      'qty',
      'amount',
      'note',
      'day',
      'when',
      'kind',
    ]);
    expect(skipped.map(i => i.name)).toEqual(['bad', 'html_x']);
    expect(normalizeSelectionOptions(undefined)).toEqual([]);
    expect(normalizeSelectionOptions([123, { label: 'x' }, ['only'], ['v'], { value: 'c' }])).toEqual([
      { value: 'only', label: 'only' },
      { value: 'v', label: 'v' },
      { value: 'c', label: 'c' },
    ]);
  });

  it('counts schema∩map and builds / writes full replace maps', () => {
    expect(countSchemaMapIntersection(['active', 'code'], { active: true, orphan: 1 })).toBe(1);
    expect(countSchemaMapIntersection([], { active: true })).toBe(0);
    const map = buildFullPropertiesMap(
      [
        null as any,
        { name: '', type: 'char' },
        { name: 'skip', type: 'html' },
        { name: 'fromPrev', type: 'char' },
        { name: 'fromValue', type: 'char', value: 'V' },
        { name: 'fromDefault', type: 'char', default: 'D' },
        { name: 'empty', type: 'char' },
      ],
      { fromPrev: 'P', orphan: 9 }
    );
    expect(map).toEqual({ fromPrev: 'P', fromValue: 'V', fromDefault: 'D' });
    expect(writePropertyValue([{ name: 'code', type: 'char' }], { code: 'A' }, 'code', 'B2')).toEqual({
      code: 'B2',
    });
    // Unknown types are not written; map stays the schema∩prev replace result.
    expect(
      writePropertyValue(
        [
          { name: 'fromPrev', type: 'char' },
          { name: 'html_x', type: 'html' },
        ],
        map,
        'html_x',
        '<x/>'
      )
    ).toEqual({ fromPrev: 'P' });

    // "__proto__" must become an own data key (align with BE properties write).
    const protoPrev = JSON.parse('{"__proto__":"from-prev"}');
    const protoMap = buildFullPropertiesMap([{ name: '__proto__', type: 'char' }], protoPrev);
    expect(Object.prototype.hasOwnProperty.call(protoMap, '__proto__')).toBe(true);
    expect(protoMap['__proto__']).toBe('from-prev');
    expect(Object.getPrototypeOf(protoMap)).toBe(null);
    const written = writePropertyValue([{ name: '__proto__', type: 'char' }], {}, '__proto__', 'safe');
    expect(Object.prototype.hasOwnProperty.call(written, '__proto__')).toBe(true);
    expect(written['__proto__']).toBe('safe');
  });

  it('converts datetime through the UTC wall-clock codec', () => {
    expect(propertyDatetimeToPicker(null)).toBeNull();
    expect(propertyDatetimeToPicker('')).toBeNull();
    expect(propertyDatetimeToPicker({ x: 1 })).toBeNull();
    const fromDate = propertyDatetimeToPicker(new Date('2024-06-30T16:00:00.000Z'), 'UTC');
    expect(fromDate).toBeInstanceOf(Date);
    const fromMs = propertyDatetimeToPicker(Date.parse('2024-06-30T16:00:00.000Z'), 'UTC');
    expect(fromMs).toBeInstanceOf(Date);
    const wall = propertyDatetimeToPicker('2024-06-30T16:00:00.000Z', 'America/New_York');
    expect(wall!.getHours()).toBe(12);
    expect(propertyDatetimeFromPicker(wall, 'America/New_York')).toBe('2024-06-30T16:00:00.000Z');
    expect(propertyDatetimeFromPicker(null)).toBeNull();
    expect(propertyDatetimeFromPicker('')).toBeNull();
    expect(propertyDatetimeFromPicker('2024-06-30T12:00:00.000Z', 'UTC')).toMatch(/Z$/);
    expect(propertyDatetimeFromPicker(new Date('invalid'), 'UTC')).toBeNull();
  });
});

describe('OPropertiesField', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  async function mountField(props: Record<string, unknown>) {
    const wrapper = mount(OPropertiesField as any, {
      props,
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await flushPromises();
    return wrapper;
  }

  it('renders resolve items and writes a full map on edit', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const ResolveProperties = vi.fn(async () => schemaItems);
    const binding = makeBinding({
      map: { active: true, code: 'A1', qty: 2, orphan: 'keep-until-replace' },
      ResolveProperties,
    });
    const wrapper = await mountField({ binding, renderMode: 'form' });
    try {
      expect(ResolveProperties).toHaveBeenCalled();
      expect(wrapper.find('[data-testid="o-properties-form"]').exists()).toBe(true);
      expect(wrapper.find('[data-name="code"]').exists()).toBe(true);
      expect(wrapper.find('[data-name="html_x"]').exists()).toBe(false);
      expect(warn).toHaveBeenCalledWith(expect.stringContaining("unsupported property type 'html'"));

      const codeInput = wrapper.find('[data-name="code"] textarea.ctrl-ElInput');
      await codeInput.setValue('B9');
      await nextTick();
      expect(binding.__value.value).toEqual({
        active: true,
        code: 'B9',
        qty: 2,
        amount: 1.5,
        day: '2024-01-02T00:00:00Z',
        when: '2024-06-30T16:00:00.000Z',
      });
      expect(binding.__value.value).not.toHaveProperty('orphan');
    } finally {
      wrapper.unmount();
      warn.mockRestore();
    }
  });

  it('covers every PP7 control type and form display values', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const ResolveProperties = vi.fn(async () => [
      { name: 'active', type: 'boolean', value: true },
      { name: 'qty', type: 'integer', value: 3 },
      { name: 'amount', type: 'float', value: 2.5 },
      { name: 'note', type: 'text', value: 'hi' },
      { name: 'day', type: 'date', value: '2024-01-02T00:00:00Z' },
      { name: 'when', type: 'datetime', value: '2024-06-30T16:00:00.000Z' },
      {
        name: 'kind',
        type: 'selection',
        value: 'a',
        selection: [
          ['a', 'Alpha'],
          { value: 'b', label: 'Beta' },
        ],
      },
      { name: 'code', type: 'char', value: 'X' },
      { name: 'flag', type: 'boolean', value: false },
      { name: 'unknownSel', type: 'selection', value: 'z', selection: [['a', 'A']] },
      { name: 'empty', type: 'char' },
      { name: 'fromDefault', type: 'char', default: 'D' },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({
      map: { active: true, qty: 3, amount: 2.5, note: 'hi', day: '2024-01-02T00:00:00Z', when: '2024-06-30T16:00:00.000Z', kind: 'a', code: 'X', flag: false },
      ResolveProperties,
    });
    const wrapper = await mountField({ binding, renderMode: 'form' });
    try {
      expect(wrapper.find('[data-type="boolean"]').exists()).toBe(true);
      expect(wrapper.find('[data-type="integer"]').exists()).toBe(true);
      expect(wrapper.find('[data-type="float"]').exists()).toBe(true);
      expect(wrapper.find('[data-type="text"]').exists()).toBe(true);
      expect(wrapper.find('[data-type="date"] .ctrl-ElDatePicker').attributes('data-type')).toBe('date');
      expect(wrapper.find('[data-type="datetime"] .ctrl-ElDatePicker').attributes('data-type')).toBe(
        'datetime'
      );
      expect(wrapper.find('[data-type="selection"]').exists()).toBe(true);
      expect(wrapper.find('[data-type="char"]').exists()).toBe(true);

      // Display slot values (Yes / No / selection label / fallback).
      expect(wrapper.text()).toContain('Yes');
      expect(wrapper.text()).toContain('No');
      expect(wrapper.text()).toContain('Alpha');
      expect(wrapper.text()).toContain('z');

      await wrapper.find('[data-type="boolean"] .ctrl-ElSwitch').setValue('false');
      await wrapper.find('[data-type="integer"] .ctrl-ElInputNumber').setValue('9');
      await wrapper.find('[data-type="float"] .ctrl-ElInputNumber').setValue('3.25');
      await wrapper.find('[data-type="text"] textarea.ctrl-ElInput').setValue('bye');
      await wrapper.find('[data-type="date"] .ctrl-ElDatePicker').setValue('2024-02-03T00:00:00Z');
      await wrapper
        .find('[data-type="datetime"] .ctrl-ElDatePicker')
        .setValue('2024-06-30T12:00:00.000Z');
      await wrapper.find('[data-type="selection"] .ctrl-ElSelect').setValue('b');
      await wrapper.find('[data-type="char"] textarea.ctrl-ElInput').setValue('Y');
      await nextTick();
      expect(binding.__value.value.active).toBe(false);
      expect(binding.__value.value.qty).toBe(9);
      expect(binding.__value.value.amount).toBe(3.25);
      expect(binding.__value.value.note).toBe('bye');
      expect(binding.__value.value.day).toBe('2024-02-03T00:00:00Z');
      expect(String(binding.__value.value.when)).toMatch(/Z$/);
      expect(binding.__value.value.kind).toBe('b');
      expect(binding.__value.value.code).toBe('Y');
    } finally {
      wrapper.unmount();
      warn.mockRestore();
    }
  });

  it('list summary and empty intersection / inline mode', async () => {
    const ResolveProperties = vi.fn(async () => [
      { name: 'active', type: 'boolean' },
      { name: 'code', type: 'char' },
    ] as ResolvedPropertyItem[]);
    const withHits = makeBinding({
      isForm: false,
      map: { active: true, orphan: 'x', code: 'Z' },
      ResolveProperties,
    });
    const table = await mountField({ binding: withHits, renderMode: 'table' });
    try {
      expect(table.find('[data-testid="o-properties-summary"]').text()).toContain('2');
      expect(table.find('[data-testid="o-properties-form"]').exists()).toBe(false);
    } finally {
      table.unmount();
    }

    const empty = makeBinding({
      isForm: false,
      map: { orphan: 1 },
      ResolveProperties: vi.fn(async () => [{ name: 'active', type: 'boolean' }] as ResolvedPropertyItem[]),
    });
    const emptyWrap = await mountField({ binding: empty, renderMode: 'inline' });
    try {
      expect(emptyWrap.find('[data-testid="o-properties-summary"]').text()).toBe('');
    } finally {
      emptyWrap.unmount();
    }
  });

  it('form display empty schema and auto mode from env.isForm', async () => {
    const ResolveProperties = vi.fn(async () => []);
    const binding = makeBinding({ map: {}, ResolveProperties, isForm: true });
    const wrapper = await mountField({ binding, renderMode: 'auto' });
    try {
      expect(wrapper.find('[data-testid="o-properties-empty"]').exists()).toBe(true);
      expect(wrapper.findAll('.o-properties-item')).toHaveLength(0);
    } finally {
      wrapper.unmount();
    }

    const listEnv = makeBinding({
      map: {},
      isForm: false,
      ResolveProperties: vi.fn(async () => []),
    });
    const listWrap = await mountField({ binding: listEnv, renderMode: 'auto' });
    try {
      expect(listWrap.find('[data-testid="o-properties-summary"]').exists()).toBe(true);
    } finally {
      listWrap.unmount();
    }
  });

  it('normalizes non-object field values via toView/fromView', async () => {
    const ResolveProperties = vi.fn(async () => [{ name: 'code', type: 'char' }] as ResolvedPropertyItem[]);
    const binding = makeBinding({ map: null, ResolveProperties });
    const wrapper = await mountField({ binding, renderMode: 'form' });
    try {
      expect(wrapper.find('[data-name="code"]').exists()).toBe(true);
      binding.__value.value = ['not', 'a', 'map'] as any;
      await nextTick();
      await flushPromises();
      const codeInput = wrapper.find('[data-name="code"] textarea.ctrl-ElInput');
      await codeInput.setValue('Z');
      await nextTick();
      expect(binding.__value.value).toEqual({ code: 'Z' });
    } finally {
      wrapper.unmount();
    }
  });

  it('re-resolves when containerId changes without trimming the in-memory map', async () => {
    const ResolveProperties = vi
      .fn()
      .mockResolvedValueOnce([{ name: 'a', type: 'char', value: '1' }] as ResolvedPropertyItem[])
      .mockResolvedValueOnce([{ name: 'b', type: 'char', value: '2' }] as ResolvedPropertyItem[]);
    const binding = makeBinding({
      map: { a: '1', orphan: 'keep' },
      ResolveProperties,
    });
    const wrapper = await mountField({ binding, renderMode: 'form', containerId: 'p1' });
    try {
      expect(wrapper.find('[data-name="a"]').exists()).toBe(true);
      expect(binding.__value.value).toEqual({ a: '1', orphan: 'keep' });

      await wrapper.setProps({ containerId: 'p2' });
      await flushPromises();
      expect(ResolveProperties).toHaveBeenLastCalledWith(
        expect.objectContaining({ Properties: { a: '1', orphan: 'keep' } }),
        'Properties',
        { containerId: 'p2' }
      );
      expect(wrapper.find('[data-name="b"]').exists()).toBe(true);
      expect(binding.__value.value).toEqual({ a: '1', orphan: 'keep' });
    } finally {
      wrapper.unmount();
    }
  });

  it('discards stale ResolveProperties responses including errors', async () => {
    let resolveOlder!: (v: ResolvedPropertyItem[]) => void;
    let resolveNewer!: (v: ResolvedPropertyItem[]) => void;
    const ResolveProperties = vi.fn(async () => [] as ResolvedPropertyItem[]);
    ResolveProperties.mockImplementationOnce(
      () =>
        new Promise<ResolvedPropertyItem[]>(resolve => {
          resolveOlder = resolve;
        })
    );
    ResolveProperties.mockImplementationOnce(
      () =>
        new Promise<ResolvedPropertyItem[]>(resolve => {
          resolveNewer = resolve;
        })
    );
    const binding = makeBinding({ map: {}, ResolveProperties });
    const wrapper = await mountField({ binding, renderMode: 'form', containerId: 'c1' });
    try {
      await wrapper.setProps({ containerId: 'c2' });
      await flushPromises();

      resolveNewer([{ name: 'newer', type: 'char', string: 'Newer' }]);
      await flushPromises();
      expect(wrapper.find('[data-name="newer"]').exists()).toBe(true);

      resolveOlder([{ name: 'older', type: 'char', string: 'Older' }]);
      await flushPromises();
      expect(wrapper.find('[data-name="newer"]').exists()).toBe(true);
      expect(wrapper.find('[data-name="older"]').exists()).toBe(false);
    } finally {
      wrapper.unmount();
    }
  });

  it('discards a stale ResolveProperties rejection that loses the race', async () => {
    let rejectFirst!: (e: Error) => void;
    let resolveSecond!: (v: ResolvedPropertyItem[]) => void;
    const ResolveProperties = vi.fn(async () => [] as ResolvedPropertyItem[]);
    ResolveProperties.mockImplementationOnce(
      () =>
        new Promise<ResolvedPropertyItem[]>((_resolve, reject) => {
          rejectFirst = reject;
        })
    );
    ResolveProperties.mockImplementationOnce(
      () =>
        new Promise<ResolvedPropertyItem[]>(resolve => {
          resolveSecond = resolve;
        })
    );
    const binding = makeBinding({ map: {}, ResolveProperties });
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const wrapper = await mountField({ binding, renderMode: 'form', containerId: 'a' });
    try {
      await wrapper.setProps({ containerId: 'b' });
      await flushPromises();
      resolveSecond([{ name: 'ok', type: 'char' }]);
      await flushPromises();
      expect(wrapper.find('[data-name="ok"]').exists()).toBe(true);
      warn.mockClear();
      rejectFirst(new Error('lost race'));
      await flushPromises();
      expect(wrapper.find('[data-name="ok"]').exists()).toBe(true);
      expect(warn).not.toHaveBeenCalled();
    } finally {
      wrapper.unmount();
      warn.mockRestore();
    }
  });

  it('clears items when ResolveProperties is missing, fails, or returns non-arrays', async () => {
    const noRpc = makeBinding({ ResolveProperties: false, map: {} });
    const noRpcWrap = await mountField({ binding: noRpc, renderMode: 'form' });
    try {
      expect(noRpcWrap.find('[data-testid="o-properties-empty"]').exists()).toBe(true);
    } finally {
      noRpcWrap.unmount();
    }

    const noProp = makeBinding({
      prop: '',
      ResolveProperties: vi.fn(async () => [{ name: 'x', type: 'char' }]),
    });
    const noPropWrap = await mountField({ binding: noProp, renderMode: 'form' });
    try {
      expect(noPropWrap.find('[data-testid="o-properties-empty"]').exists()).toBe(true);
    } finally {
      noPropWrap.unmount();
    }

    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const failing = makeBinding({
      ResolveProperties: vi.fn(async () => {
        throw new Error('rpc down');
      }),
    });
    const failWrap = await mountField({ binding: failing, renderMode: 'form' });
    try {
      expect(failWrap.find('[data-testid="o-properties-empty"]').exists()).toBe(true);
      expect(warn).toHaveBeenCalledWith('[OPropertiesField] ResolveProperties failed', expect.any(Error));
    } finally {
      failWrap.unmount();
      warn.mockRestore();
    }

    const nonArray = makeBinding({
      ResolveProperties: vi.fn(async () => ({ not: 'array' }) as any),
    });
    const nonArrayWrap = await mountField({ binding: nonArray, renderMode: 'form' });
    try {
      expect(nonArrayWrap.find('[data-testid="o-properties-empty"]').exists()).toBe(true);
    } finally {
      nonArrayWrap.unmount();
    }
  });

  it('associates labels with control ids and prefers props.prop in controlId', async () => {
    const ResolveProperties = vi.fn(async () => [
      { name: 'code', type: 'char', string: 'Code', value: 'A' },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({
      map: { code: 'A' },
      prop: '',
      ResolveProperties,
    });
    const wrapper = await mountField({
      binding,
      prop: 'AltProp',
      renderMode: 'form',
    });
    try {
      const label = wrapper.find('label.o-properties-item__label');
      expect(label.attributes('for')).toBe('o-properties-AltProp-code');
      expect(wrapper.find('#o-properties-AltProp-code').exists()).toBe(true);
    } finally {
      wrapper.unmount();
    }
  });

  it('uses item.value / item.default when map key is absent', async () => {
    const ResolveProperties = vi.fn(async () => [
      { name: 'fromValue', type: 'char', value: 'V' },
      { name: 'fromDefault', type: 'char', default: 'D' },
      { name: 'missing', type: 'char' },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({ map: {}, ResolveProperties });
    const wrapper = await mountField({ binding, renderMode: 'form' });
    try {
      expect((wrapper.find('[data-name="fromValue"] textarea.ctrl-ElInput').element as HTMLTextAreaElement).value).toBe(
        'V'
      );
      expect((wrapper.find('[data-name="fromDefault"] textarea.ctrl-ElInput').element as HTMLTextAreaElement).value).toBe(
        'D'
      );
      expect((wrapper.find('[data-name="missing"] textarea.ctrl-ElInput').element as HTMLTextAreaElement).value).toBe('');
    } finally {
      wrapper.unmount();
    }
  });

  it('clears schema when store is missing and honors readonly / default controlId', async () => {
    const noStore = makeBinding({ store: undefined, map: {} });
    const noStoreWrap = await mountField({ binding: noStore, renderMode: 'form' });
    try {
      expect(noStoreWrap.find('[data-testid="o-properties-empty"]').exists()).toBe(true);
    } finally {
      noStoreWrap.unmount();
    }

    const ResolveProperties = vi.fn(async () => [
      { name: 'code', type: 'char', string: 'Code', value: 'A', readonly: true },
      { name: 'qty', type: 'integer', value: 'not-a-number' as any },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({
      map: { code: 'A' },
      prop: undefined as any,
      ResolveProperties,
    });
    // Force empty binding.prop so controlId falls through to props.prop then 'properties'.
    (binding as any).prop = '';
    const wrapper = await mountField({ binding, renderMode: 'form', prop: '' });
    try {
      // Empty field name → no resolve → empty UI; still exercises !fieldName clear path.
      expect(wrapper.find('[data-testid="o-properties-empty"]').exists()).toBe(true);
    } finally {
      wrapper.unmount();
    }

    const ro = makeBinding({
      map: { code: 'A', qty: 'x' as any },
      ResolveProperties,
    });
    const roWrap = await mountField({ binding: ro, renderMode: 'form' });
    try {
      expect(roWrap.find('[data-name="code"] textarea.ctrl-ElInput').attributes('disabled')).toBeDefined();
      expect((roWrap.find('[data-name="qty"] .ctrl-ElInputNumber').element as HTMLInputElement).value).toBe('');
      // Clear datetime via picker null.
      const dtBinding = makeBinding({
        map: { when: '2024-06-30T16:00:00.000Z' },
        ResolveProperties: vi.fn(async () => [
          { name: 'when', type: 'datetime', value: '2024-06-30T16:00:00.000Z' },
        ] as ResolvedPropertyItem[]),
      });
      const dtWrap = await mountField({ binding: dtBinding, renderMode: 'form' });
      try {
        await dtWrap.find('[data-type="datetime"] .ctrl-ElDatePicker').setValue('');
        await nextTick();
        expect(dtBinding.__value.value.when).toBeNull();
      } finally {
        dtWrap.unmount();
      }
    } finally {
      roWrap.unmount();
    }
  });

  it('bootstraps via useField and falls back to props.store / empty record', async () => {
    const ResolveProperties = vi.fn(async (_payload: any, fieldName: string) => {
      expect(fieldName).toBe('Properties');
      return [{ name: 'code', type: 'char', value: 'Z' }] as ResolvedPropertyItem[];
    });
    const binding = makeBinding({ map: { code: 'Z' }, ResolveProperties });
    // Force store lookup onto props.store and record onto {}.
    (binding as any).store = undefined;
    (binding as any).recordRef = undefined;
    useFieldMock.impl = () => binding;
    try {
      const wrapper = await mountField({
        store: { ResolveProperties } as any,
        prop: 'Properties',
        renderMode: 'form',
      });
      try {
        expect(useFieldMock.impl).toBeTruthy();
        expect(ResolveProperties).toHaveBeenCalledWith(
          expect.objectContaining({ Properties: { code: 'Z' } }),
          'Properties',
          undefined
        );
        expect(wrapper.find('[data-name="code"]').exists()).toBe(true);
        expect(wrapper.find('#o-properties-Properties-code').exists()).toBe(true);
      } finally {
        wrapper.unmount();
      }
    } finally {
      useFieldMock.impl = null;
    }
  });
});
