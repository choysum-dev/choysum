// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, nextTick, reactive, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyFieldBase from './ChoyFieldBase.vue';
import ChoyPropertiesDefinitionField from './ChoyPropertiesDefinitionField.vue';
import ChoyPropertiesDefinitionItemDialog from './ChoyPropertiesDefinitionItemDialog.vue';
import ChoyPropertyLabelTranslationsDialog from './ChoyPropertyLabelTranslationsDialog.vue';

function makeBinding(opts: {
  definition?: unknown[] | null;
  isEditMode?: boolean;
  isForm?: boolean;
}): UseField & { __value: any } {
  const value = ref(opts.definition === undefined ? [] : opts.definition);
  return {
    env: {
      isForm: opts.isForm !== false,
      isEditMode: opts.isEditMode !== false,
      viewMode: opts.isEditMode === false ? 'display' : 'edit',
      fieldPrefix: null,
    },
    prop: 'Definition',
    meta: reactive({ type: 'jsonobject' }) as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => ({ Id: '1', Definition: value.value })) as any,
    registerFields: () => undefined,
    store: {} as any,
    asView: () => ({ fieldValue: () => value }) as any,
    __value: value,
  } as any;
}

function installFieldBaseStub() {
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
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
      orientation: { type: String, default: undefined },
      labelForControl: { type: Boolean, default: undefined },
    },
    setup(props: any, { slots }: any) {
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
      const slotProps = {
        fieldValue,
        inputId: 'def-input',
        labelId: 'def-label',
        describedBy: undefined,
        invalid: false,
      };
      return () => {
        const mode = props.binding?.env?.isEditMode ? 'edit' : 'display';
        const slot = mode === 'edit' ? slots.edit : slots.display;
        return h('div', { 'data-testid': 'choy-field-base-stub' }, slot?.(slotProps) ?? []);
      };
    },
  } as any);
}

function installChoyButtonStub() {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    props: {
      disabled: Boolean,
      loading: Boolean,
      type: String,
      size: String,
      variant: String,
    },
    emits: ['click'],
    inheritAttrs: false,
    setup(props: any, { emit, slots, attrs }: any) {
      return () =>
        h(
          'button',
          {
            ...attrs,
            type: 'button',
            disabled: props.disabled || undefined,
            onClick: (e: Event) => emit('click', e),
          },
          slots.default?.()
        );
    },
  } as any);
}

function installItemDialogStub() {
  stubSfc(ChoyPropertiesDefinitionItemDialog as any, {
    name: 'ChoyPropertiesDefinitionItemDialog',
    props: {
      modelValue: Boolean,
      item: Object,
      readonly: Boolean,
      title: String,
    },
    emits: ['update:modelValue', 'saved', 'closed'],
    setup(props: any, { emit }: any) {
      return () =>
        props.modelValue
          ? h('div', { 'data-testid': 'choy-properties-definition-item-dialog-stub' }, [
              h(
                'button',
                {
                  'data-testid': 'stub-item-save',
                  onClick: () =>
                    emit('saved', {
                      name: 'from_dialog',
                      type: 'char',
                      string: 'From Dialog',
                      stringMap: { en_US: 'From Dialog' },
                      help: '',
                      helpMap: {},
                      default: '',
                      readonly: false,
                      selectionText: '',
                      selectionLabelMaps: {},
                    }),
                },
                'save',
              ),
              h(
                'button',
                {
                  'data-testid': 'stub-item-close',
                  onClick: () => emit('update:modelValue', false),
                },
                'close',
              ),
            ])
          : null;
    },
  } as any);
}

function installTranslateDialogStub() {
  stubSfc(ChoyPropertyLabelTranslationsDialog as any, {
    name: 'ChoyPropertyLabelTranslationsDialog',
    props: { modelValue: Boolean, labelTitle: String, translations: Object },
    emits: ['update:modelValue', 'saved', 'closed'],
    setup() {
      return () => null;
    },
  } as any);
}

describe('ChoyPropertiesDefinitionField', () => {
  beforeEach(() => {
    installFieldBaseStub();
    installChoyButtonStub();
    installItemDialogStub();
    installTranslateDialogStub();
  });

  afterEach(() => {
    restoreSfc(ChoyFieldBase as any);
    restoreSfc(ChoyButton as any);
    restoreSfc(ChoyPropertiesDefinitionItemDialog as any);
    restoreSfc(ChoyPropertyLabelTranslationsDialog as any);
  });

  test('edit mode shows rows and Add; display mode hides Add', async () => {
    const binding = makeBinding({
      definition: [
        { name: 'tax_id', type: 'char', string: { en_US: 'Tax' } },
        { name: 'prio', type: 'selection', string: { en_US: 'Priority' }, selection: [['low', 'Low']] },
      ],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      expect(mounted.q('[data-testid="choy-properties-definition-form"]')).toBeTruthy();
      expect(mounted.q('[data-testid="choy-properties-definition-add"]')).toBeTruthy();
      expect(mounted.el.querySelectorAll('[data-testid="choy-properties-definition-row"]').length).toBe(2);
      expect(mounted.el.textContent || '').toMatch(/tax_id/);
      expect(mounted.el.textContent || '').toMatch(/Tax/);
    } finally {
      mounted.unmount();
    }

    const displayBinding = makeBinding({
      definition: [{ name: 'tax_id', type: 'char', string: { en_US: 'Tax' } }],
      isEditMode: false,
    });
    const displayMounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding: displayBinding },
    });
    try {
      await flushPromises();
      expect(displayMounted.q('[data-testid="choy-properties-definition-add"]')).toBeFalsy();
      expect(displayMounted.q('[data-testid="choy-properties-definition-display"]')).toBeTruthy();
    } finally {
      displayMounted.unmount();
    }
  });

  test('Remove updates draft Definition array on the field', async () => {
    const binding = makeBinding({
      definition: [
        { name: 'a', type: 'char', string: { en_US: 'A' } },
        { name: 'b', type: 'char', string: { en_US: 'B' } },
      ],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      const removeBtns = mounted.el.querySelectorAll('[data-testid="choy-properties-definition-remove"]');
      expect(removeBtns.length).toBe(2);
      (removeBtns[0] as HTMLButtonElement).click();
      await nextTick();
      await flushPromises();
      const next = binding.__value.value as any[];
      expect(next.length).toBe(1);
      expect(next[0].name).toBe('b');
    } finally {
      mounted.unmount();
    }
  });

  test('shows 1-based index column for definition rows', async () => {
    const binding = makeBinding({
      definition: [
        { name: 'a', type: 'char', string: { en_US: 'A' } },
        { name: 'b', type: 'char', string: { en_US: 'B' } },
      ],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      expect(mounted.q('[data-testid="choy-properties-definition-index-header"]')?.textContent).toBe('#');
      const indexes = mounted.qa('[data-testid="choy-properties-definition-index"]');
      expect(indexes.map(el => (el.textContent || '').trim())).toEqual(['1', '2']);
    } finally {
      mounted.unmount();
    }
  });

  test('handle drag reorders Definition rows', async () => {
    const binding = makeBinding({
      definition: [
        { name: 'a', type: 'char', string: { en_US: 'A' } },
        { name: 'b', type: 'char', string: { en_US: 'B' } },
        { name: 'c', type: 'char', string: { en_US: 'C' } },
      ],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      expect(mounted.qa('[data-testid="choy-properties-definition-handle"]').length).toBe(3);
      const exposed = mounted.root as any;
      const draftsRaw = exposed.drafts;
      const drafts = (Array.isArray(draftsRaw) ? draftsRaw : draftsRaw?.value) as any[];
      expect(drafts?.length).toBe(3);
      exposed.commitDrafts([drafts[2], drafts[0], drafts[1]]);
      await nextTick();
      await flushPromises();
      expect((binding.__value.value as any[]).map((r: any) => r.name)).toEqual(['c', 'a', 'b']);
    } finally {
      mounted.unmount();
    }
  });

  test('Add opens dialog; Confirm writes new item into Definition', async () => {
    const binding = makeBinding({ definition: [], isEditMode: true });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      expect(mounted.q('[data-testid="choy-properties-definition-empty"]')).toBeTruthy();
      mounted.click('[data-testid="choy-properties-definition-add"]');
      await nextTick();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();
      mounted.click('[data-testid="stub-item-save"]');
      await nextTick();
      await flushPromises();
      const next = binding.__value.value as any[];
      expect(next.length).toBe(1);
      expect(next[0].name).toBe('from_dialog');
      // Single-lang storage may collapse to a plain string.
      const label = next[0].string;
      if (typeof label === 'string') {
        expect(label).toBe('From Dialog');
      } else {
        expect(label).toEqual({ en_US: 'From Dialog' });
      }
    } finally {
      mounted.unmount();
    }
  });

  test('edit mode does not put translate control on the summary table', async () => {
    const binding = makeBinding({
      definition: [{ name: 'tax_id', type: 'char', string: { en_US: 'Tax', zh_CN: '税号' } }],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      expect(mounted.q('[data-testid="choy-properties-definition-translate"]')).toBeFalsy();
      expect(mounted.q('[data-testid="choy-properties-definition-translate-string"]')).toBeFalsy();
    } finally {
      mounted.unmount();
    }
  });

  test('table-like mode renders summary text', async () => {
    const binding = makeBinding({
      definition: [
        { name: 'a', type: 'char' },
        { name: 'b', type: 'char' },
      ],
      isEditMode: true,
      isForm: false,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      const summary = mounted.q('[data-testid="choy-properties-definition-summary"]');
      expect(summary).toBeTruthy();
      expect(summary!.textContent || '').toMatch(/a/);
      expect(summary!.textContent || '').toMatch(/b/);
    } finally {
      mounted.unmount();
    }
  });
});
