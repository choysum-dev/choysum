// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, inject, provide, reactive } from 'vue';
import { createPinia, setActivePinia } from 'pinia';

import { resolveConfirmChoy } from '@/web/web/composables/confirmChoyAction';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyFormView from '@/web/web/components/view/ChoyFormView.vue';
import ChoyVarcharField from '@/web/web/components/field/ChoyVarcharField.vue';
import ChoyTextField from '@/web/web/components/field/ChoyTextField.vue';
import ChoyBooleanField from '@/web/web/components/field/ChoyBooleanField.vue';
import ChoySelectionField from '@/web/web/components/field/ChoySelectionField.vue';
import ChoyPropertyLabelTranslationsDialog from './ChoyPropertyLabelTranslationsDialog.vue';
import ChoyPropertiesDefinitionSelectionEditor from './ChoyPropertiesDefinitionSelectionEditor.vue';
import ItemDialog from './ChoyPropertiesDefinitionItemDialog.vue';
import type { DefinitionEditorDraftItem } from './propertiesDefinitionHelpers';
import type { LocalFormStore } from '@/web/web/stores/localFormStore';

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
  let lastSelectionProps: Record<string, unknown> | null = null;

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

    stubSfc(ChoyFormView as any, {
      name: 'ChoyFormView',
      props: {
        store: Object,
        initialValues: Object,
        viewMode: String,
        embedded: Boolean,
        showHeader: Boolean,
        showActions: Boolean,
        submitHandler: Function,
        cancelHandler: Function,
      },
      setup(props: any, { slots, attrs, expose }: any) {
        const formData = reactive({ ...(props.initialValues || {}) });
        const syncStoreField = (path: string, value: unknown) => {
          (formData as any)[path] = value;
          const store = props.store as LocalFormStore | null | undefined;
          store?.setField?.(path, value);
        };
        provide('form-root', {
          get mode() {
            return props.viewMode === 'display' ? 'display' : 'create';
          },
          get draft() {
            return formData;
          },
          setField(path: string, value: unknown) {
            syncStoreField(path, value);
          },
          getField(path: string) {
            return (formData as any)[path];
          },
          onChange() {},
        });
        expose({
          submit: () =>
            props.submitHandler?.({
              formData,
              mode: 'create',
              data: formData,
              defaultSubmit: async () => null,
            }),
          // Mirror FormView.cancel: dirty confirm is skipped in this stub; host
          // cancelHandler closes the dialog (same post-confirm contract).
          cancel: async () => {
            await props.cancelHandler?.();
          },
        });
        return () =>
          h('div', { ...attrs, 'data-testid': 'choy-properties-definition-item-form' }, [
            slots.default?.({ formData, viewMode: props.viewMode }),
          ]);
      },
    });

    function fieldStub(name: string, tag: 'input' | 'textarea' | 'checkbox' | 'select') {
      return {
        name,
        props: {
          store: Object,
          prop: String,
          readonly: { type: Boolean, default: false },
        },
        setup(props: any, { attrs }: any) {
          const form = inject<any>('form-root');
          const testId = attrs['data-testid'] || `field-${props.prop}`;
          return () => {
            const mode = form?.mode;
            if (mode === 'display' || props.readonly) {
              const v = form?.getField?.(props.prop);
              const text = v === true ? 'Yes' : v === false ? 'No' : String(v ?? '');
              return h('div', { 'data-testid': testId, 'data-readonly': '1' }, text);
            }
            if (tag === 'checkbox') {
              return h('input', {
                type: 'checkbox',
                'data-testid': testId,
                checked: form?.getField?.(props.prop) === true,
                onChange: (e: Event) => form?.setField?.(props.prop, (e.target as HTMLInputElement).checked),
              });
            }
            if (tag === 'select') {
              return h('div', { 'data-testid': testId }, [
                h(
                  'button',
                  {
                    type: 'button',
                    'data-testid': 'stub-set-selection-type',
                    onClick: () => form?.setField?.(props.prop, 'selection'),
                  },
                  String(form?.getField?.(props.prop) ?? ''),
                ),
              ]);
            }
            return h(tag, {
              'data-testid': testId,
              value: String(form?.getField?.(props.prop) ?? ''),
              onInput: (e: Event) =>
                form?.setField?.(props.prop, (e.target as HTMLInputElement | HTMLTextAreaElement).value),
            });
          };
        },
      };
    }

    stubSfc(ChoyVarcharField as any, fieldStub('ChoyVarcharField', 'input') as any);
    stubSfc(ChoyTextField as any, fieldStub('ChoyTextField', 'textarea') as any);
    stubSfc(ChoyBooleanField as any, fieldStub('ChoyBooleanField', 'checkbox') as any);
    stubSfc(ChoySelectionField as any, fieldStub('ChoySelectionField', 'select') as any);

    stubSfc(ChoyPropertyLabelTranslationsDialog as any, {
      name: 'ChoyPropertyLabelTranslationsDialog',
      props: { modelValue: Boolean, labelTitle: String, translations: Object },
      emits: ['update:modelValue', 'saved', 'closed'],
      setup(props: any, { emit }: any) {
        lastTranslateProps = props;
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
    lastSelectionProps = null;
    installStubs();
  });

  afterEach(() => {
    restoreSfc(ChoyDialog as any);
    restoreSfc(ChoyDialogContent as any);
    restoreSfc(ChoyDialogTitle as any);
    restoreSfc(ChoyButton as any);
    restoreSfc(ChoyFormView as any);
    restoreSfc(ChoyVarcharField as any);
    restoreSfc(ChoyTextField as any);
    restoreSfc(ChoyBooleanField as any);
    restoreSfc(ChoySelectionField as any);
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
    expect(edit.q('[data-testid="choy-properties-definition-item-save"]')).toBeTruthy();
    expect(edit.q('[data-testid="choy-properties-definition-item-footer"]')).toBeTruthy();
    edit.unmount();

    const neu = mountDialog({ title: '', item: draft() });
    await flushPromises();
    expect(neu.el.querySelector('[data-testid="item-dialog-title"]')?.textContent || '').toMatch(/New/i);
    neu.unmount();

    const ro = mountDialog({ title: '', readonly: true });
    await flushPromises();
    expect(ro.el.querySelector('[data-testid="item-dialog-title"]')?.textContent || '').toMatch(/Property/);
    expect(ro.q('[data-testid="choy-properties-definition-item-save"]')).toBeFalsy();
    expect(ro.q('[data-testid="choy-properties-definition-item-edit"]')).toBeFalsy();
    expect(ro.q('[data-testid="choy-properties-definition-item-cancel"]')).toBeTruthy();
    ro.unmount();

    const roEdit = mountDialog({ title: '', readonly: true, canEdit: true });
    await flushPromises();
    expect(roEdit.q('[data-testid="choy-properties-definition-item-edit"]')).toBeTruthy();
    roEdit.unmount();
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

  test('Apply emits saved via FormView submit', async () => {
    const onSaved = fnRecorder();
    const onUpdate = fnRecorder();
    const m = mountDialog({}, { onSaved, 'onUpdate:modelValue': onUpdate });
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-item-save"]');
    await flushPromises();
    expect(onSaved.calls.length).toBe(1);
    expect((onSaved.calls[0][0] as DefinitionEditorDraftItem).name).toBe('tax_id');
    expect(onUpdate.calls.some((c: any[]) => c[0] === false)).toBe(false);
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

  test('Cancel footer closes dialog', async () => {
    const onUpdate = fnRecorder();
    const m = mountDialog({}, { 'onUpdate:modelValue': onUpdate });
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-item-cancel"]');
    await flushPromises();
    expect(onUpdate.calls.some((c: any[]) => c[0] === false)).toBe(true);
    m.unmount();
  });

  test('Cancel with selection-only edits confirms before close', async () => {
    const onUpdate = fnRecorder();
    const m = mountDialog(
      {
        item: draft({
          name: 'prio',
          type: 'selection',
          string: 'P',
          stringMap: { en_US: 'P' },
          selectionText: '[["low","Low"]]',
          selectionLabelMaps: { low: { en_US: 'Low' } },
        }),
      },
      { 'onUpdate:modelValue': onUpdate },
    );
    try {
      await flushPromises();
      m.click('[data-testid="stub-selection-update"]');
      await flushPromises();
      queueMicrotask(() => resolveConfirmChoy('confirm'));
      m.click('[data-testid="choy-properties-definition-item-cancel"]');
      await flushPromises();
      expect(onUpdate.calls.some((c: any[]) => c[0] === false)).toBe(true);
    } finally {
      m.unmount();
    }
  });

  test('Cancel with selection edits declines keeps dialog open', async () => {
    const onUpdate = fnRecorder();
    const m = mountDialog(
      {
        item: draft({
          name: 'prio',
          type: 'selection',
          string: 'P',
          stringMap: { en_US: 'P' },
          selectionText: '[["low","Low"]]',
          selectionLabelMaps: { low: { en_US: 'Low' } },
        }),
      },
      { 'onUpdate:modelValue': onUpdate },
    );
    try {
      await flushPromises();
      m.click('[data-testid="stub-selection-update"]');
      await flushPromises();
      queueMicrotask(() => resolveConfirmChoy('cancel'));
      m.click('[data-testid="choy-properties-definition-item-cancel"]');
      await flushPromises();
      expect(onUpdate.calls.some((c: any[]) => c[0] === false)).toBe(false);
      expect(m.q('[data-testid="choy-properties-definition-item-form"]')).toBeTruthy();
    } finally {
      m.unmount();
    }
  });

  test('Cancel without FormView.cancel closes directly when clean', async () => {
    stubSfc(ChoyFormView as any, {
      name: 'ChoyFormView',
      props: {
        store: Object,
        initialValues: Object,
        viewMode: String,
        submitHandler: Function,
        cancelHandler: Function,
      },
      setup(props: any, { slots, attrs, expose }: any) {
        const formData = reactive({ ...(props.initialValues || {}) });
        provide('form-root', {
          get mode() {
            return 'create';
          },
          get draft() {
            return formData;
          },
          setField(path: string, value: unknown) {
            (formData as any)[path] = value;
            (props.store as LocalFormStore | undefined)?.setField?.(path, value);
          },
          getField(path: string) {
            return (formData as any)[path];
          },
          onChange() {},
        });
        expose({});
        return () =>
          h('div', { ...attrs, 'data-testid': 'choy-properties-definition-item-form' }, [
            slots.default?.({ formData, viewMode: props.viewMode }),
          ]);
      },
    });
    const onUpdate = fnRecorder();
    const m = mountDialog({}, { 'onUpdate:modelValue': onUpdate });
    try {
      await flushPromises();
      m.click('[data-testid="choy-properties-definition-item-cancel"]');
      await flushPromises();
      expect(onUpdate.calls.some((c: any[]) => c[0] === false)).toBe(true);
    } finally {
      m.unmount();
      installStubs();
    }
  });

  test('readonly Close closes without requiring FormView.cancel', async () => {
    const onUpdate = fnRecorder();
    const m = mountDialog({ readonly: true }, { 'onUpdate:modelValue': onUpdate });
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-item-cancel"]');
    await flushPromises();
    expect(onUpdate.calls.some((c: any[]) => c[0] === false)).toBe(true);
    m.unmount();
  });

  test('onApply/onRequestEdit early-return guards via setupState', async () => {
    const onEdit = fnRecorder();
    const m = mountDialog({ readonly: true, canEdit: false }, { onEdit });
    try {
      await flushPromises();
      const api = m.setupState();
      await api.onApply();
      api.onRequestEdit();
      expect(onEdit.calls.length).toBe(0);
      expect(m.q('[data-testid="choy-properties-definition-item-error"]')).toBeFalsy();
    } finally {
      m.unmount();
    }
  });

  test('onApply surfaces thrown submit errors as formError', async () => {
    stubSfc(ChoyFormView as any, {
      name: 'ChoyFormView',
      props: {
        store: Object,
        initialValues: Object,
        viewMode: String,
        submitHandler: Function,
        cancelHandler: Function,
      },
      setup(props: any, { slots, attrs, expose }: any) {
        const formData = reactive({ ...(props.initialValues || {}) });
        provide('form-root', {
          get mode() {
            return 'create';
          },
          get draft() {
            return formData;
          },
          setField() {},
          getField(path: string) {
            return (formData as any)[path];
          },
          onChange() {},
        });
        expose({
          submit: async () => {
            throw new Error('submit boom');
          },
        });
        return () =>
          h('div', { ...attrs, 'data-testid': 'choy-properties-definition-item-form' }, [
            slots.default?.({ formData, viewMode: props.viewMode }),
          ]);
      },
    });
    const m = mountDialog({});
    try {
      await flushPromises();
      m.click('[data-testid="choy-properties-definition-item-save"]');
      await flushPromises();
      expect(m.q('[data-testid="choy-properties-definition-item-error"]')?.textContent || '').toMatch(
        /submit boom/,
      );
    } finally {
      m.unmount();
      installStubs();
    }
  });

  test('Apply falls back to onFormSubmit when FormView has no submit()', async () => {
    stubSfc(ChoyFormView as any, {
      name: 'ChoyFormView',
      props: {
        store: Object,
        initialValues: Object,
        viewMode: String,
        submitHandler: Function,
        cancelHandler: Function,
      },
      setup(props: any, { slots, attrs, expose }: any) {
        const formData = reactive({ ...(props.initialValues || {}) });
        provide('form-root', {
          get mode() {
            return 'create';
          },
          get draft() {
            return formData;
          },
          setField(path: string, value: unknown) {
            (formData as any)[path] = value;
          },
          getField(path: string) {
            return (formData as any)[path];
          },
          onChange() {},
        });
        // No submit/cancel expose — exercises onApply fallback.
        expose({});
        return () =>
          h('div', { ...attrs, 'data-testid': 'choy-properties-definition-item-form' }, [
            slots.default?.({ formData, viewMode: props.viewMode }),
          ]);
      },
    });
    const onSaved = fnRecorder();
    const m = mountDialog({}, { onSaved });
    try {
      await flushPromises();
      m.click('[data-testid="choy-properties-definition-item-save"]');
      await flushPromises();
      expect(onSaved.calls.length).toBe(1);
    } finally {
      m.unmount();
      installStubs();
    }
  });

  test('submitHandler is a no-op when dialog is readonly', async () => {
    const onSaved = fnRecorder();
    let submitHandler: ((ctx: any) => Promise<unknown>) | undefined;
    stubSfc(ChoyFormView as any, {
      name: 'ChoyFormView',
      props: {
        store: Object,
        initialValues: Object,
        viewMode: String,
        submitHandler: Function,
        cancelHandler: Function,
      },
      setup(props: any, { slots, attrs, expose }: any) {
        const formData = reactive({ ...(props.initialValues || {}) });
        submitHandler = props.submitHandler;
        provide('form-root', {
          get mode() {
            return 'display';
          },
          get draft() {
            return formData;
          },
          setField() {},
          getField(path: string) {
            return (formData as any)[path];
          },
          onChange() {},
        });
        expose({
          submit: () =>
            props.submitHandler?.({
              formData,
              mode: 'display',
              data: formData,
              defaultSubmit: async () => null,
            }),
        });
        return () =>
          h('div', { ...attrs, 'data-testid': 'choy-properties-definition-item-form' }, [
            slots.default?.({ formData, viewMode: props.viewMode }),
          ]);
      },
    });
    const m = mountDialog({ readonly: true }, { onSaved });
    try {
      await flushPromises();
      expect(submitHandler).toBeTruthy();
      const result = await submitHandler!({
        formData: {},
        mode: 'display',
        data: {},
        defaultSubmit: async () => null,
      });
      expect(result).toEqual({ handled: true, skipSuccessMessage: true });
      expect(onSaved.calls.length).toBe(0);
    } finally {
      m.unmount();
      installStubs();
    }
  });

  test('readonly Edit emits edit when canEdit', async () => {
    const onEdit = fnRecorder();
    const m = mountDialog({ readonly: true, canEdit: true }, { onEdit });
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-item-edit"]');
    await flushPromises();
    expect(onEdit.calls.length).toBe(1);
    m.unmount();
  });

  test('Edit is hidden when canEdit is false', async () => {
    const onEdit = fnRecorder();
    const m = mountDialog({ readonly: true, canEdit: false }, { onEdit });
    await flushPromises();
    expect(m.q('[data-testid="choy-properties-definition-item-edit"]')).toBeFalsy();
    expect(onEdit.calls.length).toBe(0);
    m.unmount();
  });

  test('translate label and help apply maps via onTranslateSaved', async () => {
    const onSaved = fnRecorder();
    const m = mountDialog(
      {
        item: draft({
          name: 'tax_id',
          string: 'Tax',
          stringMap: { en_US: 'Tax' },
          help: 'H',
          helpMap: { en_US: 'H' },
        }),
      },
      { onSaved },
    );
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-translate-string"]');
    await flushPromises();
    m.click('[data-testid="stub-translate-save-string"]');
    await flushPromises();
    expect(String((m.q('[data-testid="choy-properties-definition-string"]') as HTMLInputElement)?.value || '')).toBe(
      'L-en',
    );

    m.click('[data-testid="choy-properties-definition-translate-help"]');
    await flushPromises();
    m.click('[data-testid="stub-translate-save-help"]');
    await flushPromises();

    m.click('[data-testid="choy-properties-definition-item-save"]');
    await flushPromises();
    expect(onSaved.calls.length).toBe(1);
    const saved = onSaved.calls[0][0] as DefinitionEditorDraftItem;
    expect(saved.stringMap).toEqual({ en_US: 'L-en', zh_CN: 'L-zh' });
    expect(saved.helpMap).toEqual({ en_US: 'H-en', zh_CN: 'H-zh' });
    m.unmount();
  });

  test('selection type shows editor; update and translate option', async () => {
    const onSaved = fnRecorder();
    const m = mountDialog(
      {
        item: draft({
          name: 'prio',
          type: 'selection',
          string: 'P',
          stringMap: { en_US: 'P' },
          selectionText: '[["low","Low"]]',
          selectionLabelMaps: { low: { en_US: 'Low' } },
        }),
      },
      { onSaved },
    );
    await flushPromises();
    expect(m.q('[data-testid="choy-properties-definition-section-selection"]')).toBeTruthy();
    expect(m.q('[data-testid="choy-properties-definition-selection-scroll"]')).toBeTruthy();
    expect(m.q('[data-testid="selection-editor-stub"]')).toBeTruthy();
    m.click('[data-testid="stub-selection-update"]');
    await flushPromises();
    m.click('[data-testid="stub-selection-translate"]');
    await flushPromises();
    expect(lastTranslateProps).toBeTruthy();
    m.click('[data-testid="stub-translate-save-selection"]');
    await flushPromises();
    m.click('[data-testid="choy-properties-definition-item-save"]');
    await flushPromises();
    expect(onSaved.calls.length).toBe(1);
    const saved = onSaved.calls[0][0] as DefinitionEditorDraftItem;
    expect(saved.selectionLabelMaps.low).toEqual({ en_US: 'S-en', zh_CN: 'S-zh' });
    m.unmount();
  });

  test('readonly uses FormView display mode (not edit inputs) and skips translate', async () => {
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
    expect(m.q('[data-testid="choy-properties-definition-item-display"]')).toBeTruthy();
    expect(m.q('[data-testid="choy-properties-definition-name"]')?.getAttribute('data-readonly')).toBe('1');
    expect(m.q('input')).toBeFalsy();
    expect(m.q('[data-testid="choy-properties-definition-translate-string"]')).toBeFalsy();
    expect(m.q('[data-testid="selection-editor-stub"]')).toBeTruthy();
    expect(lastSelectionProps?.readonly).toBe(true);
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
      setup(_props: any, { emit }: any) {
        return () =>
          h('div', { 'data-testid': 'selection-editor-stub' }, [
            h(
              'button',
              {
                type: 'button',
                'data-testid': 'stub-selection-translate-empty',
                onClick: () => emit('translate', '  '),
              },
              'translate-empty',
            ),
          ]);
      },
    });
    const m = mountDialog({
      item: draft({
        name: 'prio',
        type: 'selection',
        string: 'P',
        stringMap: { en_US: 'P' },
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
