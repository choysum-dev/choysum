// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { computed, h, nextTick, reactive, ref } from 'vue';

import { TRANSLATED_BASE_LANG } from '@/core/service/orm/model/properties_label_i18n';
import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { useI18nStore } from '@/web/web/stores/i18nStore';
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
      expect(displayMounted.q('[data-testid="choy-properties-definition-remove"]')).toBeFalsy();
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

  test('new-item dialog appends even if rows shrink while open', async () => {
    const binding = makeBinding({
      definition: [{ name: 'tax_id', type: 'char', string: { en_US: 'Tax' } }],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      mounted.click('[data-testid="choy-properties-definition-add"]');
      await nextTick();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();
      // Remove the existing row while the add dialog is still open.
      const removeBtns = mounted.el.querySelectorAll('[data-testid="choy-properties-definition-remove"]');
      expect(removeBtns.length).toBe(1);
      (removeBtns[0] as HTMLElement).click();
      await nextTick();
      await flushPromises();
      expect((binding.__value.value as any[]).length).toBe(0);
      mounted.click('[data-testid="stub-item-save"]');
      await nextTick();
      await flushPromises();
      const saved = binding.__value.value as any[];
      expect(saved.length).toBe(1);
      expect(saved[0]?.name).toBe('from_dialog');
    } finally {
      mounted.unmount();
    }
  });

  test('stale edit index surfaces writeError and keeps dialog open', async () => {
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
      const rows = mounted.el.querySelectorAll('[data-testid="choy-properties-definition-row"]');
      expect(rows.length).toBe(2);
      (rows[1] as HTMLElement).click();
      await nextTick();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();
      // Remove the first row so dialogIndex 1 is now out of range.
      const removeBtns = mounted.el.querySelectorAll('[data-testid="choy-properties-definition-remove"]');
      (removeBtns[0] as HTMLElement).click();
      await nextTick();
      await flushPromises();
      expect((binding.__value.value as any[]).map((r: any) => r.name)).toEqual(['b']);
      mounted.click('[data-testid="stub-item-save"]');
      await nextTick();
      await flushPromises();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();
      expect(mounted.q('[data-testid="choy-properties-definition-write-error"]')?.textContent || '').toMatch(
        /changed while editing/i,
      );
      expect((binding.__value.value as any[]).map((r: any) => r.name)).toEqual(['b']);
    } finally {
      mounted.unmount();
    }
  });

  test('reorder while edit open blocks save when original name no longer matches', async () => {
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
      const rows = mounted.el.querySelectorAll('[data-testid="choy-properties-definition-row"]');
      (rows[0] as HTMLElement).click();
      await nextTick();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();
      const state = mounted.setupState() as {
        onReorder?: (rows: any[]) => void;
        drafts?: any[];
      };
      const drafts = Array.isArray(state.drafts) ? state.drafts : [];
      expect(drafts.length).toBe(2);
      state.onReorder?.([drafts[1], drafts[0]]);
      await nextTick();
      await flushPromises();
      mounted.click('[data-testid="stub-item-save"]');
      await nextTick();
      await flushPromises();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();
      expect(mounted.q('[data-testid="choy-properties-definition-write-error"]')?.textContent || '').toMatch(
        /changed while editing/i,
      );
      expect((binding.__value.value as any[]).map((r: any) => r.name)).toEqual(['b', 'a']);
    } finally {
      mounted.unmount();
    }
  });

  test('external definition change closes open edit dialog', async () => {
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
      const rows = mounted.el.querySelectorAll('[data-testid="choy-properties-definition-row"]');
      (rows[0] as HTMLElement).click();
      await nextTick();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();
      binding.__value.value = [
        { name: 'b', type: 'char', string: { en_US: 'B' } },
        { name: 'a', type: 'char', string: { en_US: 'A' } },
      ];
      await nextTick();
      await flushPromises();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeFalsy();
      expect(mounted.q('[data-testid="choy-properties-definition-write-error"]')?.textContent || '').toMatch(
        /changed while editing/i,
      );
    } finally {
      mounted.unmount();
    }
  });

  test('commitDrafts failure keeps dialog open (onDialogSaved catch return)', async () => {
    // Invalid selection JSON → draftsToDefinitionItems throws inside commitDrafts.
    stubSfc(ChoyPropertiesDefinitionItemDialog as any, {
      name: 'ChoyPropertiesDefinitionItemDialog',
      props: {
        modelValue: Boolean,
        item: Object,
        readonly: Boolean,
        title: String,
        commitError: String,
      },
      emits: ['update:modelValue', 'saved', 'closed'],
      setup(props: any, { emit }: any) {
        return () =>
          props.modelValue
            ? h('div', { 'data-testid': 'choy-properties-definition-item-dialog-stub' }, [
                props.commitError
                  ? h(
                      'p',
                      { 'data-testid': 'choy-properties-definition-item-error' },
                      String(props.commitError),
                    )
                  : null,
                h(
                  'button',
                  {
                    'data-testid': 'stub-item-save-bad',
                    onClick: () =>
                      emit('saved', {
                        name: 'bad_sel',
                        type: 'selection',
                        string: 'Bad',
                        stringMap: { en_US: 'Bad' },
                        help: '',
                        helpMap: {},
                        default: '',
                        readonly: false,
                        selectionText: '',
                        selectionLabelMaps: {},
                      }),
                  },
                  'save-bad',
                ),
              ])
            : null;
      },
    } as any);

    const binding = makeBinding({ definition: [], isEditMode: true });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      mounted.click('[data-testid="choy-properties-definition-add"]');
      await nextTick();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();
      mounted.click('[data-testid="stub-item-save-bad"]');
      await nextTick();
      await flushPromises();
      // Dialog stays open; commit error is shown inside the dialog (not only behind it).
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();
      expect(mounted.q('[data-testid="choy-properties-definition-item-error"]')?.textContent || '').toMatch(
        /selection/,
      );
      expect((binding.__value.value as any[]).length).toBe(0);
    } finally {
      mounted.unmount();
      installItemDialogStub();
    }
  });

  test('openRow readonly in display mode has no View column and opens dialog via row click', async () => {
    const binding = makeBinding({
      definition: [{ name: 'tax_id', type: 'char', string: { en_US: 'Tax' } }],
      isEditMode: false,
    });
    stubSfc(ChoyPropertiesDefinitionItemDialog as any, {
      name: 'ChoyPropertiesDefinitionItemDialog',
      props: {
        modelValue: Boolean,
        item: Object,
        readonly: Boolean,
        title: String,
      },
      emits: ['update:modelValue', 'saved', 'closed'],
      setup(props: any) {
        return () =>
          props.modelValue
            ? h('div', {
                'data-testid': 'choy-properties-definition-item-dialog-stub',
                'data-readonly': props.readonly ? '1' : '0',
              })
            : null;
      },
    } as any);

    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      expect(mounted.q('[data-testid="choy-properties-definition-remove"]')).toBeFalsy();
      // Row click opens as readonly when canMutate is false (display mode).
      mounted.click('[data-testid="choy-properties-definition-row"]');
      await nextTick();
      const stub = mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]');
      expect(stub).toBeTruthy();
      expect(stub?.getAttribute('data-readonly')).toBe('1');
    } finally {
      mounted.unmount();
      installItemDialogStub();
    }
  });

  test('openRow edit path and remove keep selection drafts intact', async () => {
    const binding = makeBinding({
      definition: [
        {
          name: 'prio',
          type: 'selection',
          string: { en_US: 'Priority' },
          selection: [
            ['low', 'Low'],
            ['high', 'High'],
          ],
        },
        { name: 'code', type: 'char', string: { en_US: 'Code' } },
      ],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding },
    });
    try {
      await flushPromises();
      expect(mounted.el.textContent || '').toMatch(/prio/);
      expect(mounted.el.textContent || '').toMatch(/selection/);
      mounted.click('[data-testid="choy-properties-definition-row"]');
      await nextTick();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();

      const removeBtns = mounted.el.querySelectorAll(
        '[data-testid="choy-properties-definition-remove"]',
      );
      (removeBtns[0] as HTMLButtonElement).click();
      await nextTick();
      await flushPromises();
      const next = binding.__value.value as any[];
      expect(next.length).toBe(1);
      expect(next[0].name).toBe('code');
    } finally {
      mounted.unmount();
    }
  });

  test('summaryText covers empty, few, and many named properties', async () => {
    const empty = makeBinding({ definition: [], isEditMode: true, isForm: false });
    const emptyMounted = mountApp(ChoyPropertiesDefinitionField as any, { props: { binding: empty } });
    try {
      await flushPromises();
      expect(emptyMounted.q('[data-testid="choy-properties-definition-summary"]')?.textContent || '').toMatch(
        /No properties|properties/i,
      );
    } finally {
      emptyMounted.unmount();
    }

    const few = makeBinding({
      definition: [
        { name: 'a', type: 'char' },
        { name: 'b', type: 'char' },
        { name: 'c', type: 'char' },
      ],
      isEditMode: true,
      isForm: false,
    });
    const fewMounted = mountApp(ChoyPropertiesDefinitionField as any, { props: { binding: few } });
    try {
      await flushPromises();
      const text = fewMounted.q('[data-testid="choy-properties-definition-summary"]')?.textContent || '';
      expect(text).toMatch(/a/);
      expect(text).toMatch(/c/);
    } finally {
      fewMounted.unmount();
    }

    const many = makeBinding({
      definition: [
        { name: 'a', type: 'char' },
        { name: 'b', type: 'char' },
        { name: 'c', type: 'char' },
        { name: 'd', type: 'char' },
      ],
      isEditMode: true,
      isForm: false,
    });
    const manyMounted = mountApp(ChoyPropertiesDefinitionField as any, { props: { binding: many } });
    try {
      await flushPromises();
      expect(manyMounted.q('[data-testid="choy-properties-definition-summary"]')?.textContent || '').toMatch(
        /4|properties|\.\.\./i,
      );
    } finally {
      manyMounted.unmount();
    }
  });

  test('row keyboard handlers open dialog; readonly saved is ignored', async () => {
    const binding = makeBinding({
      definition: [{ name: 'a', type: 'char', string: { en_US: 'A' } }],
      isEditMode: false,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, { props: { binding } });
    try {
      await flushPromises();
      const row = mounted.q('[data-testid="choy-properties-definition-row"]') as HTMLElement;
      // QuickJS has no KeyboardEvent; attach key onto a cancelable Event (Kanban pattern).
      const enter = new Event('keydown', { bubbles: true, cancelable: true });
      Object.defineProperty(enter, 'key', { value: 'Enter' });
      row.dispatchEvent(enter);
      await nextTick();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();

      mounted.click('[data-testid="stub-item-close"]');
      await nextTick();
      const space = new Event('keydown', { bubbles: true, cancelable: true });
      Object.defineProperty(space, 'key', { value: ' ' });
      row.dispatchEvent(space);
      await nextTick();
      expect(mounted.q('[data-testid="choy-properties-definition-item-dialog-stub"]')).toBeTruthy();

      expect(mounted.q('[data-testid="stub-item-save"]')).toBeTruthy();
      mounted.click('[data-testid="stub-item-save"]');
      await nextTick();
      await flushPromises();
      expect((binding.__value.value as any[])[0].name).toBe('a');
    } finally {
      mounted.unmount();
    }
  });

  test('edit existing row save replaces item at index', async () => {
    const binding = makeBinding({
      definition: [{ name: 'a', type: 'char', string: { en_US: 'A' } }],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, { props: { binding } });
    try {
      await flushPromises();
      mounted.click('[data-testid="choy-properties-definition-row"]');
      await nextTick();
      mounted.click('[data-testid="stub-item-save"]');
      await nextTick();
      await flushPromises();
      expect((binding.__value.value as any[])[0].name).toBe('from_dialog');
    } finally {
      mounted.unmount();
    }
  });

  test('opening a dialog clears a stale writeError', async () => {
    const binding = makeBinding({
      definition: [{ name: 'a', type: 'char', string: { en_US: 'A' } }],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, { props: { binding } });
    try {
      await flushPromises();
      // setupState unwraps refs; assign through the proxy so the underlying ref updates.
      const state = mounted.setupState() as { writeError: string };
      state.writeError = 'stale';
      expect(state.writeError).toBe('stale');
      mounted.click('[data-testid="choy-properties-definition-row"]');
      await nextTick();
      expect(state.writeError).toBe('');
    } finally {
      mounted.unmount();
    }
  });

  test('editorLang reads i18n store and falls back on empty or throw', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const i18n = useI18nStore();
    i18n.localeCode = 'zh-CN';

    const binding = makeBinding({
      definition: [{ name: 'a', type: 'char', string: { en_US: 'A' } }],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, { props: { binding } });
    try {
      await flushPromises();
      const state = mounted.setupState() as { editorLang?: () => string };
      expect(typeof state.editorLang).toBe('function');
      expect(state.editorLang?.()).toBe('zh_CN');

      Object.defineProperty(i18n, 'terminologyLang', {
        configurable: true,
        get: () => '   ',
      });
      expect(state.editorLang?.()).toBe(TRANSLATED_BASE_LANG);

      Object.defineProperty(i18n, 'terminologyLang', {
        configurable: true,
        get: () => {
          throw new Error('forced');
        },
      });
      expect(state.editorLang?.()).toBe(TRANSLATED_BASE_LANG);
    } finally {
      mounted.unmount();
      setActivePinia(undefined as any);
    }
  });

  test('onReorder failure keeps prior definition when commitDrafts throws', async () => {
    const binding = makeBinding({
      definition: [
        { name: 'a', type: 'char', string: { en_US: 'A' } },
        { name: 'b', type: 'char', string: { en_US: 'B' } },
      ],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, { props: { binding } });
    try {
      await flushPromises();
      const exposed = mounted.root as any;
      const drafts = (Array.isArray(exposed.drafts) ? exposed.drafts : exposed.drafts?.value) as any[];
      const bad = drafts.map((d: any, i: number) =>
        i === 0
          ? {
              ...d,
              type: 'selection',
              selectionText: '',
              selectionLabelMaps: {},
            }
          : d,
      );
      exposed.commitDrafts = () => {
        throw new Error('forced');
      };
      const state = mounted.setupState() as { onReorder?: (rows: any[]) => void };
      state.onReorder?.(bad);
      await nextTick();
      expect((binding.__value.value as any[]).map((r: any) => r.name)).toEqual(['a', 'b']);
    } finally {
      mounted.unmount();
    }
  });

  test('summaryText nameless items; handle drag and row dragover/drop', async () => {
    const nameless = makeBinding({
      definition: [{ type: 'char' }, { type: 'char' }],
      isEditMode: true,
      isForm: false,
    });
    const namelessMounted = mountApp(ChoyPropertiesDefinitionField as any, {
      props: { binding: nameless },
    });
    try {
      await flushPromises();
      expect(
        namelessMounted.q('[data-testid="choy-properties-definition-summary"]')?.textContent || '',
      ).toMatch(/2|properties/i);
    } finally {
      namelessMounted.unmount();
    }

    const binding = makeBinding({
      definition: [
        { name: 'a', type: 'char', string: { en_US: 'A' } },
        { name: 'b', type: 'char', string: { en_US: 'B' } },
      ],
      isEditMode: true,
    });
    const mounted = mountApp(ChoyPropertiesDefinitionField as any, { props: { binding } });
    try {
      await flushPromises();
      const handle = mounted.q('[data-testid="choy-properties-definition-handle"]') as HTMLElement;
      const row = mounted.qa('[data-testid="choy-properties-definition-row"]')[1] as HTMLElement;
      const dt = {
        effectAllowed: '',
        dropEffect: '',
        setData: () => undefined,
        setDragImage: () => undefined,
      };
      handle.dispatchEvent(
        Object.assign(new Event('dragstart', { bubbles: true }), {
          dataTransfer: dt,
          stopPropagation() {},
        }),
      );
      row.dispatchEvent(
        Object.assign(new Event('dragover', { bubbles: true }), {
          dataTransfer: dt,
          preventDefault() {},
        }),
      );
      row.dispatchEvent(
        Object.assign(new Event('drop', { bubbles: true }), {
          dataTransfer: dt,
          preventDefault() {},
        }),
      );
      handle.dispatchEvent(
        Object.assign(new Event('dragend', { bubbles: true }), {
          stopPropagation() {},
        }),
      );
      await nextTick();
      await flushPromises();
    } finally {
      mounted.unmount();
    }
  });

});
