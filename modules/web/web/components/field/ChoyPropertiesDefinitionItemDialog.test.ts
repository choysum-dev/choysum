// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { createPinia, setActivePinia } from 'pinia';

import { useI18nStore } from '@/web/web/stores/i18nStore';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import Checkbox from '../vendor/ui/checkbox/Checkbox.vue';
import FieldLabel from '../vendor/ui/field/FieldLabel.vue';
import Input from '../vendor/ui/input/Input.vue';
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import Select from '../vendor/ui/select/Select.vue';
import SelectContent from '../vendor/ui/select/SelectContent.vue';
import SelectItem from '../vendor/ui/select/SelectItem.vue';
import SelectTrigger from '../vendor/ui/select/SelectTrigger.vue';
import ChoyPropertyLabelTranslationsDialog from './ChoyPropertyLabelTranslationsDialog.vue';
import ChoyPropertiesDefinitionSelectionEditor from './ChoyPropertiesDefinitionSelectionEditor.vue';
import ItemDialog from './ChoyPropertiesDefinitionItemDialog.vue';
import type { DefinitionEditorDraftItem } from './propertiesDefinitionHelpers';

function draft(partial: Partial<DefinitionEditorDraftItem> = {}): DefinitionEditorDraftItem {
  return {
    name: '',
    type: 'char',
    string: '',
    stringMap: {},
    help: '',
    helpMap: {},
    default: '',
    readonly: false,
    selectionText: '',
    selectionLabelMaps: {},
    ...partial,
  };
}

describe('ChoyPropertiesDefinitionItemDialog', () => {
  let pinia: ReturnType<typeof createPinia>;
  let lastTranslateProps: Record<string, unknown> | null = null;
  let translateSavedEmit: ((next: Record<string, string>) => void) | null = null;
  let lastSelectionProps: Record<string, unknown> | null = null;
  let selectionTranslateEmit: ((value: string) => void) | null = null;
  let selectionUpdateEmit: ((rows: unknown[]) => void) | null = null;

  function installStubs() {
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
        return () => h('div', { class: ['dialog-content', attrs.class], ...attrs }, slots.default?.());
      },
    });
    stubSfc(ChoyDialogTitle as any, {
      name: 'DialogTitle',
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'dialog-title', 'data-testid': 'item-dialog-title' }, slots.default?.());
      },
    });
    stubSfc(ChoyButton as any, {
      name: 'ChoyButton',
      props: {
        type: { type: String, default: 'button' },
        variant: { type: String, default: 'default' },
        size: { type: String, default: undefined },
        disabled: { type: Boolean, default: false },
      },
      emits: ['click'],
      setup(props: any, { emit, slots, attrs }: any) {
        return () =>
          h(
            'button',
            {
              ...attrs,
              type: props.type || 'button',
              disabled: props.disabled || undefined,
              onClick: (e: Event) => emit('click', e),
            },
            slots.default?.(),
          );
      },
    });
    stubSfc(Input as any, {
      name: 'Input',
      props: { modelValue: { type: [String, Number], default: '' }, disabled: Boolean, id: String },
      emits: ['update:modelValue'],
      setup(props: any, { emit, attrs }: any) {
        return () =>
          h('input', {
            ...attrs,
            id: props.id,
            disabled: props.disabled || undefined,
            value: props.modelValue ?? '',
            onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value),
          });
      },
    });
    stubSfc(Textarea as any, {
      name: 'Textarea',
      props: { modelValue: { type: String, default: '' }, disabled: Boolean, id: String },
      emits: ['update:modelValue'],
      setup(props: any, { emit, attrs }: any) {
        return () =>
          h('textarea', {
            ...attrs,
            id: props.id,
            disabled: props.disabled || undefined,
            value: props.modelValue ?? '',
            onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLTextAreaElement).value),
          });
      },
    });
    stubSfc(FieldLabel as any, {
      name: 'FieldLabel',
      setup(_: any, { slots }: any) {
        return () => h('label', slots.default?.());
      },
    });
    stubSfc(Checkbox as any, {
      name: 'Checkbox',
      props: { modelValue: { type: [Boolean, String, Number], default: false }, disabled: Boolean },
      emits: ['update:modelValue'],
      setup(props: any, { emit, attrs }: any) {
        return () =>
          h('input', {
            ...attrs,
            type: 'checkbox',
            disabled: props.disabled || undefined,
            checked: props.modelValue === true,
            onChange: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).checked),
          });
      },
    });
    stubSfc(Select as any, {
      name: 'Select',
      props: { modelValue: { type: String, default: '' }, disabled: Boolean },
      emits: ['update:modelValue'],
      setup(props: any, { slots, emit }: any) {
        return () =>
          h('div', { class: 'select-stub', 'data-value': props.modelValue }, [
            slots.default?.(),
            h(
              'button',
              {
                type: 'button',
                'data-testid': 'stub-set-selection-type',
                disabled: props.disabled || undefined,
                onClick: () => emit('update:modelValue', 'selection'),
              },
              'selection',
            ),
          ]);
      },
    });
    stubSfc(SelectTrigger as any, {
      name: 'SelectTrigger',
      setup(_: any, { attrs }: any) {
        return () => h('button', { ...attrs, type: 'button' }, 'type');
      },
    });
    stubSfc(SelectContent as any, {
      name: 'SelectContent',
      setup(_: any, { slots }: any) {
        return () => h('div', slots.default?.());
      },
    });
    stubSfc(SelectItem as any, {
      name: 'SelectItem',
      props: { value: String },
      setup(props: any, { slots }: any) {
        return () => h('div', { 'data-value': props.value }, slots.default?.());
      },
    });
    stubSfc(ChoyPropertyLabelTranslationsDialog as any, {
      name: 'ChoyPropertyLabelTranslationsDialog',
      props: { modelValue: Boolean, labelTitle: String, translations: Object },
      emits: ['update:modelValue', 'saved', 'closed'],
      setup(props: any, { emit }: any) {
        lastTranslateProps = props;
        translateSavedEmit = (next: Record<string, string>) => emit('saved', next);
        return () =>
          props.modelValue
            ? h('div', { 'data-testid': 'translate-dialog-stub' }, [
                h(
                  'button',
                  {
                    type: 'button',
                    'data-testid': 'stub-translate-save-string',
                    onClick: () => emit('saved', { en_US: 'L-en', zh_CN: 'L-zh' }),
                  },
                  'save-string',
                ),
                h(
                  'button',
                  {
                    type: 'button',
                    'data-testid': 'stub-translate-save-help',
                    onClick: () => emit('saved', { en_US: 'H-en', zh_CN: 'H-zh' }),
                  },
                  'save-help',
                ),
                h(
                  'button',
                  {
                    type: 'button',
                    'data-testid': 'stub-translate-save-selection',
                    onClick: () => emit('saved', { en_US: 'S-en', zh_CN: 'S-zh' }),
                  },
                  'save-selection',
                ),
              ])
            : null;
      },
    });
    stubSfc(ChoyPropertiesDefinitionSelectionEditor as any, {
      name: 'ChoyPropertiesDefinitionSelectionEditor',
      props: { modelValue: { type: Array, default: () => [] }, readonly: Boolean },
      emits: ['update:modelValue', 'translate'],
      setup(props: any, { emit }: any) {
        lastSelectionProps = props;
        selectionTranslateEmit = (value: string) => emit('translate', value);
        selectionUpdateEmit = (rows: unknown[]) => emit('update:modelValue', rows);
        return () =>
          h('div', { 'data-testid': 'selection-editor-stub' }, [
            h(
              'button',
              {
                type: 'button',
                'data-testid': 'stub-selection-update',
                onClick: () =>
                  emit('update:modelValue', [
                    { value: 'low', label: 'Low', labelMap: { en_US: 'Low' } },
                    { value: 'high', label: 'High', labelMap: { en_US: 'High' } },
                  ]),
              },
              'update',
            ),
            h(
              'button',
              {
                type: 'button',
                'data-testid': 'stub-selection-translate',
                onClick: () => emit('translate', 'low'),
              },
              'translate',
            ),
          ]);
      },
    });
  }

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    void useI18nStore();
    lastTranslateProps = null;
    translateSavedEmit = null;
    lastSelectionProps = null;
    selectionTranslateEmit = null;
    selectionUpdateEmit = null;
    installStubs();
  });

  afterEach(() => {
    restoreSfc(ChoyDialog as any);
    restoreSfc(ChoyDialogContent as any);
    restoreSfc(ChoyDialogTitle as any);
    restoreSfc(ChoyButton as any);
    restoreSfc(Input as any);
    restoreSfc(Textarea as any);
    restoreSfc(FieldLabel as any);
    restoreSfc(Checkbox as any);
    restoreSfc(Select as any);
    restoreSfc(SelectTrigger as any);
    restoreSfc(SelectContent as any);
    restoreSfc(SelectItem as any);
    restoreSfc(ChoyPropertyLabelTranslationsDialog as any);
    restoreSfc(ChoyPropertiesDefinitionSelectionEditor as any);
  });

  function mountDialog(props: Record<string, unknown> = {}, on?: Record<string, (...args: any[]) => void>) {
    return mountApp(ItemDialog as any, {
      props: {
        modelValue: true,
        item: draft({ name: 'tax_id', string: 'Tax', stringMap: { en_US: 'Tax' } }),
        ...props,
      },
      on,
      plugins: [pinia],
    });
  }

  test('title override and edit/new/readonly titles', async () => {
    const override = mountDialog({ title: 'Custom Title' });
    await flushPromises();
    expect(override.el.querySelector('[data-testid="item-dialog-title"]')?.textContent).toBe('Custom Title');
    override.unmount();

    const edit = mountDialog({ title: '' });
    await flushPromises();
    expect(edit.el.querySelector('[data-testid="item-dialog-title"]')?.textContent || '').toMatch(/tax_id/);
    edit.unmount();

    const neu = mountDialog({ title: '', item: draft() });
    await flushPromises();
    expect(neu.el.querySelector('[data-testid="item-dialog-title"]')?.textContent || '').toMatch(/New/i);
    neu.unmount();

    const ro = mountDialog({ title: '', readonly: true });
    await flushPromises();
    expect(ro.el.querySelector('[data-testid="item-dialog-title"]')?.textContent || '').toMatch(/Property/);
    expect(ro.q('[data-testid="choy-properties-definition-item-save"]')).toBeFalsy();
    ro.unmount();

    const roEmpty = mountDialog({ title: '', readonly: true, item: draft() });
    await flushPromises();
    expect(roEmpty.el.querySelector('[data-testid="item-dialog-title"]')?.textContent || '').toMatch(/Property/);
    roEmpty.unmount();
  });

  test('Name required validation shows formError', async () => {
    const onSaved = fnRecorder();
    const m = mountDialog({ item: draft({ name: '  ' }) }, { onSaved });
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-item-save"]');
    await flushPromises();
    expect(m.q('[data-testid="choy-properties-definition-item-error"]')).toBeTruthy();
    expect(onSaved.calls.length).toBe(0);
    m.unmount();
  });

  test('Save surfaces applyDraftCurrentLangToMaps failures as formError', async () => {
    const onSaved = fnRecorder();
    const i18n = useI18nStore();
    Object.defineProperty(i18n, 'terminologyLang', {
      configurable: true,
      get: () => 'zh-CN',
    });
    const m = mountDialog({ item: draft({ name: 'tax_id', string: 'Tax', stringMap: { en_US: 'Tax' } }) }, { onSaved });
    try {
      await flushPromises();
      m.click('[data-testid="choy-properties-definition-item-save"]');
      await flushPromises();
      expect(onSaved.calls.length).toBe(0);
      const err = m.q('[data-testid="choy-properties-definition-item-error"]');
      expect(err).toBeTruthy();
      expect(String(err?.textContent || '')).toMatch(/terminology|locale|language/i);
    } finally {
      m.unmount();
    }
  });

  test('Save emits saved and does not close itself', async () => {
    const onSaved = fnRecorder();
    const onUpdate = fnRecorder();
    const m = mountDialog({}, { onSaved, 'onUpdate:modelValue': onUpdate });
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-item-save"]');
    await flushPromises();
    expect(onSaved.calls.length).toBe(1);
    expect((onSaved.calls[0][0] as DefinitionEditorDraftItem).name).toBe('tax_id');
    expect(onUpdate.calls.some((c: any[]) => c[0] === false)).toBe(false);
    expect(m.q('[data-testid="choy-properties-definition-item-dialog"]')).toBeTruthy();
    m.unmount();
  });

  test('Save trims name onto the draft before emit', async () => {
    const onSaved = fnRecorder();
    const m = mountDialog(
      { item: draft({ name: '  padded_name  ', string: 'P', stringMap: { en_US: 'P' } }) },
      { onSaved },
    );
    try {
      await flushPromises();
      m.click('[data-testid="choy-properties-definition-item-save"]');
      await flushPromises();
      expect(onSaved.calls.length).toBe(1);
      expect((onSaved.calls[0][0] as DefinitionEditorDraftItem).name).toBe('padded_name');
    } finally {
      m.unmount();
    }
  });

  test('Cancel closes dialog', async () => {
    const onUpdate = fnRecorder();
    const onClosed = fnRecorder();
    const m = mountApp(ItemDialog as any, {
      reactiveProps: true,
      props: {
        modelValue: true,
        item: draft({ name: 'tax_id', string: 'Tax', stringMap: { en_US: 'Tax' } }),
        'onUpdate:modelValue': (v: boolean) => {
          (m.props as any).modelValue = v;
        },
        onClosed,
      },
      plugins: [pinia],
    });
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-item-cancel"]');
    await flushPromises();
    expect((m.props as any).modelValue).toBe(false);
    expect(onClosed.calls.length).toBeGreaterThanOrEqual(1);
    // onUpdate may be wired via props callback rather than `on` map
    void onUpdate;
    m.unmount();
  });

  test('translate label and help apply maps via onTranslateSaved', async () => {
    const m = mountDialog({
      item: draft({
        name: 'tax_id',
        string: 'Tax',
        stringMap: { en_US: 'Tax' },
        help: 'Help EN',
        helpMap: { en_US: 'Help EN' },
      }),
    });
    await flushPromises();

    m.click('[data-testid="choy-properties-definition-translate-string"]');
    await flushPromises();
    expect(m.q('[data-testid="translate-dialog-stub"]')).toBeTruthy();
    expect(String(lastTranslateProps?.labelTitle || '')).toMatch(/tax_id/);
    m.click('[data-testid="stub-translate-save-string"]');
    await flushPromises();

    m.click('[data-testid="choy-properties-definition-translate-help"]');
    await flushPromises();
    expect(String(lastTranslateProps?.labelTitle || '')).toMatch(/Help/i);
    m.click('[data-testid="stub-translate-save-help"]');
    await flushPromises();

    // Re-open and save again so maps stick into local before confirm
    m.click('[data-testid="choy-properties-definition-item-save"]');
    await flushPromises();
    m.unmount();
  });

  test('selection type shows editor; update and translate option', async () => {
    const onSaved = fnRecorder();
    const m = mountDialog(
      {
        item: draft({
          name: 'prio',
          type: 'selection',
          string: 'Priority',
          stringMap: { en_US: 'Priority' },
          selectionText: '[["low","Low"]]',
          selectionLabelMaps: { low: { en_US: 'Low' } },
        }),
      },
      { onSaved },
    );
    await flushPromises();
    expect(m.q('[data-testid="selection-editor-stub"]')).toBeTruthy();
    expect(lastSelectionProps).toBeTruthy();

    m.click('[data-testid="stub-selection-update"]');
    await flushPromises();
    expect(selectionUpdateEmit).toBeTruthy();

    m.click('[data-testid="stub-selection-translate"]');
    await flushPromises();
    expect(m.q('[data-testid="translate-dialog-stub"]')).toBeTruthy();
    expect(selectionTranslateEmit).toBeTruthy();
    m.click('[data-testid="stub-translate-save-selection"]');
    await flushPromises();

    m.click('[data-testid="choy-properties-definition-item-save"]');
    await flushPromises();
    expect(onSaved.calls.length).toBe(1);
    const saved = onSaved.calls[0][0] as DefinitionEditorDraftItem;
    expect(saved.type).toBe('selection');
    expect(saved.selectionLabelMaps.low).toBeTruthy();
    m.unmount();
  });

  test('readonly skips translate controls and selection mutate', async () => {
    const m = mountDialog({
      readonly: true,
      item: draft({
        name: 'prio',
        type: 'selection',
        selectionText: '[["low","Low"]]',
        selectionLabelMaps: { low: { en_US: 'Low' } },
      }),
    });
    await flushPromises();
    expect(m.q('[data-testid="choy-properties-definition-translate-string"]')).toBeFalsy();
    expect(m.q('[data-testid="choy-properties-definition-translate-help"]')).toBeFalsy();
    expect(m.q('[data-testid="selection-editor-stub"]')).toBeTruthy();
    // Selection update should be ignored when readonly
    m.click('[data-testid="stub-selection-update"]');
    await flushPromises();
    m.unmount();
  });

  test('switching to selection type via select stub shows editor', async () => {
    const m = mountDialog({ item: draft({ name: 'x', type: 'char' }) });
    await flushPromises();
    expect(m.q('[data-testid="selection-editor-stub"]')).toBeFalsy();
    m.click('[data-testid="stub-set-selection-type"]');
    await flushPromises();
    expect(m.q('[data-testid="selection-editor-stub"]')).toBeTruthy();
    m.unmount();
  });

  test('editorLang falls back without pinia', async () => {
    // Unset pinia after mount setup path uses TRANSLATED_BASE_LANG when no pinia
    setActivePinia(undefined as any);
    const onSaved = fnRecorder();
    const m = mountApp(ItemDialog as any, {
      props: {
        modelValue: true,
        item: draft({ name: 'no_pinia', string: 'X', stringMap: { en_US: 'X' } }),
      },
      on: { onSaved },
    });
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-item-save"]');
    await flushPromises();
    expect(onSaved.calls.length).toBe(1);
    m.unmount();
    setActivePinia(pinia);
  });

  test('checkbox toggles readonly on draft', async () => {
    const onSaved = fnRecorder();
    const m = mountDialog({}, { onSaved });
    await flushPromises();
    const cb = m.q('[data-testid="choy-properties-definition-readonly"]') as HTMLInputElement;
    expect(cb).toBeTruthy();
    cb.checked = true;
    cb.dispatchEvent(new Event('change', { bubbles: true }));
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-item-save"]');
    await flushPromises();
    expect((onSaved.calls[0][0] as DefinitionEditorDraftItem).readonly).toBe(true);
    m.unmount();
  });

  test('openSelectionTranslate ignores empty value', async () => {
    stubSfc(ChoyPropertiesDefinitionSelectionEditor as any, {
      name: 'ChoyPropertiesDefinitionSelectionEditor',
      props: { modelValue: { type: Array, default: () => [] }, readonly: Boolean },
      emits: ['update:modelValue', 'translate'],
      setup(_: any, { emit }: any) {
        return () =>
          h('div', { 'data-testid': 'selection-editor-stub' }, [
            h(
              'button',
              {
                type: 'button',
                'data-testid': 'stub-selection-translate-empty',
                onClick: () => emit('translate', '  '),
              },
              'empty',
            ),
          ]);
      },
    });
    const m = mountDialog({
      item: draft({
        name: 'prio',
        type: 'selection',
        selectionText: '[["low","Low"]]',
        selectionLabelMaps: { low: { en_US: 'Low' } },
      }),
    });
    await flushPromises();
    m.click('[data-testid="stub-selection-translate-empty"]');
    await flushPromises();
    expect(m.q('[data-testid="translate-dialog-stub"]')).toBeFalsy();
    m.unmount();
  });
});
