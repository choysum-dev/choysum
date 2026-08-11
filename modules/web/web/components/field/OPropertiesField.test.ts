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
      },
      emits: ['update:modelValue'],
      setup(props, { emit, slots }) {
        return () =>
          h(
            'div',
            { class: `stub-${name}` },
            [
              h(tag, {
                class: `ctrl-${name}`,
                value: props.modelValue == null ? '' : String(props.modelValue),
                disabled: props.disabled || undefined,
                onInput: (e: Event) => {
                  const raw = (e.target as HTMLInputElement).value;
                  if (name === 'ElSwitch') emit('update:modelValue', raw === 'true' || raw === '1');
                  else if (name === 'ElInputNumber') emit('update:modelValue', raw === '' ? undefined : Number(raw));
                  else emit('update:modelValue', raw);
                },
              }),
              slots.default?.(),
            ]
          );
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
    ElDatePicker: Control('ElDatePicker'),
  };
});

function makeBinding(opts: {
  map?: Record<string, unknown>;
  record?: Record<string, unknown>;
  isForm?: boolean;
  ResolveProperties?: (...args: any[]) => Promise<ResolvedPropertyItem[]>;
}): UseField & { __value: any; __store: any } {
  const value = ref(opts.map ?? {});
  const recordRef = ref(opts.record ?? { Id: '1', Properties: value.value });
  const ResolveProperties =
    opts.ResolveProperties ??
    vi.fn(async () => [] as ResolvedPropertyItem[]);
  const store = { ResolveProperties };
  return {
    env: {
      isForm: opts.isForm !== false,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: 'Properties',
    meta: reactive({ type: 'properties' }) as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => recordRef.value) as any,
    registerFields: () => undefined,
    store: store as any,
    asView: () => ({ fieldValue: () => value }) as any,
    __value: value,
    __store: store,
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
      h('div', { class: 'ob' }, [
        slots.edit?.({ fieldValue }),
        slots.display?.({ fieldValue }),
      ]);
  },
});

const schemaItems: ResolvedPropertyItem[] = [
  { name: 'active', type: 'boolean', string: 'Active', value: true },
  { name: 'code', type: 'char', string: 'Code', value: 'A1' },
  { name: 'qty', type: 'integer', string: 'Qty', value: 2 },
  { name: 'note', type: 'text', string: 'Note' },
  { name: 'kind', type: 'selection', string: 'Kind', selection: [['a', 'Alpha'], { value: 'b', label: 'Beta' }] },
  { name: 'html_x', type: 'html', string: 'Bad' },
];

describe('oproperties_helpers', () => {
  it('filters unknown types and normalizes selection options', () => {
    const { renderable, skipped } = filterRenderablePropertyItems(schemaItems);
    expect(renderable.map(i => i.name)).toEqual(['active', 'code', 'qty', 'note', 'kind']);
    expect(skipped.map(i => i.name)).toEqual(['html_x']);
    expect(normalizeSelectionOptions(schemaItems[4]!.selection)).toEqual([
      { value: 'a', label: 'Alpha' },
      { value: 'b', label: 'Beta' },
    ]);
  });

  it('counts schema∩map and writes a full replace map', () => {
    expect(countSchemaMapIntersection(['active', 'code'], { active: true, orphan: 1 })).toBe(1);
    expect(countSchemaMapIntersection([], { active: true })).toBe(0);
    const map = buildFullPropertiesMap(schemaItems, { active: false, code: 'Z', orphan: 9 });
    expect(map).toEqual({ active: false, code: 'Z', qty: 2 });
    expect(writePropertyValue(schemaItems, map, 'code', 'B2')).toEqual({
      active: false,
      code: 'B2',
      qty: 2,
    });
    expect(writePropertyValue(schemaItems, map, 'html_x', '<x/>')).toEqual(map);
  });
});

describe('OPropertiesField', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders resolve items and writes a full map on edit', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const ResolveProperties = vi.fn(async () => schemaItems);
    const binding = makeBinding({
      map: { active: true, code: 'A1', qty: 2, orphan: 'keep-until-replace' },
      ResolveProperties,
    });
    const wrapper = mount(OPropertiesField as any, {
      props: { binding, renderMode: 'form' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    try {
      await flushPromises();
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
      });
      expect(binding.__value.value).not.toHaveProperty('orphan');
    } finally {
      wrapper.unmount();
      warn.mockRestore();
    }
  });

  it('list summary counts only schema∩map', async () => {
    const ResolveProperties = vi.fn(async () => [
      { name: 'active', type: 'boolean' },
      { name: 'code', type: 'char' },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({
      isForm: false,
      map: { active: true, orphan: 'x', code: 'Z' },
      ResolveProperties,
    });
    const wrapper = mount(OPropertiesField as any, {
      props: { binding, renderMode: 'table' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    try {
      await flushPromises();
      const text = wrapper.find('[data-testid="o-properties-summary"]').text();
      expect(text).toContain('2');
      expect(wrapper.find('[data-testid="o-properties-form"]').exists()).toBe(false);
    } finally {
      wrapper.unmount();
    }
  });

  it('empty schema renders without error', async () => {
    const ResolveProperties = vi.fn(async () => []);
    const binding = makeBinding({ map: {}, ResolveProperties });
    const wrapper = mount(OPropertiesField as any, {
      props: { binding, renderMode: 'form' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    try {
      await flushPromises();
      expect(wrapper.find('[data-testid="o-properties-empty"]').exists()).toBe(true);
      expect(wrapper.findAll('.o-properties-item')).toHaveLength(0);
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
    const wrapper = mount(OPropertiesField as any, {
      props: { binding, renderMode: 'form', containerId: 'p1' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    try {
      await flushPromises();
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
      // PP4 B1: memory map is not clipped on re-resolve.
      expect(binding.__value.value).toEqual({ a: '1', orphan: 'keep' });
    } finally {
      wrapper.unmount();
    }
  });
});
