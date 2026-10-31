<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDialog v-model:open="visible">
    <ChoyDialogContent
      class="choy-properties-definition-item-dialog flex max-h-[min(85vh,44rem)] w-[min(90vw,32rem)] max-w-[min(90vw,32rem)] flex-col gap-4 overflow-hidden"
      data-testid="choy-properties-definition-item-dialog"
      @open-auto-focus.prevent
    >
      <ChoyDialogTitle class="shrink-0">{{ dialogTitle }}</ChoyDialogTitle>
      <div class="min-h-0 overflow-y-auto">
        <ChoyFormView
          v-if="formStore"
          ref="formViewRef"
          class="choy-properties-definition-item-form !gap-0"
          :store="formStore"
          :view-mode="formViewMode"
          embedded
          :show-header="false"
          :show-actions="false"
          :show-messages="false"
          :resolve-record-id-from-route="false"
          :initial-values="formInitialValues"
          :submit-handler="onFormSubmit"
          :cancel-handler="onFormCancel"
          data-testid="choy-properties-definition-item-form"
        >
          <template #default="{ formData }">
            <span class="hidden">{{ trackFormData(formData) }}</span>
            <div
              class="flex flex-col gap-4"
              :data-testid="
                readonly
                  ? 'choy-properties-definition-item-display'
                  : 'choy-properties-definition-item-edit'
              "
            >
              <section
                class="grid gap-2"
                data-testid="choy-properties-definition-section-identity"
              >
                <div class="text-sm font-medium text-foreground">
                  {{ _t('Identity') }}
                </div>
                <ChoyVarcharField
                  :store="formStore"
                  prop="Name"
                  orientation="horizontal"
                  buffer-strategy="live"
                  :required="!readonly"
                  data-testid="choy-properties-definition-name"
                />
                <ChoySelectionField
                  :store="formStore"
                  prop="Type"
                  orientation="horizontal"
                  :clearable="false"
                  data-testid="choy-properties-definition-type"
                />
              </section>

              <section
                class="grid gap-2"
                data-testid="choy-properties-definition-section-presentation"
              >
                <div class="text-sm font-medium text-foreground">
                  {{ _t('Presentation') }}
                </div>
                <div class="flex min-w-0 items-center gap-1">
                  <ChoyVarcharField
                    class="min-w-0 flex-1"
                    :store="formStore"
                    prop="Label"
                    orientation="horizontal"
                    buffer-strategy="live"
                    data-testid="choy-properties-definition-string"
                  />
                  <ChoyButton
                    v-if="!readonly"
                    type="button"
                    size="icon"
                    variant="ghost"
                    class="size-8 shrink-0 text-muted-foreground hover:text-foreground"
                    :aria-label="_t('Translate label')"
                    :title="_t('Translate label')"
                    data-testid="choy-properties-definition-translate-string"
                    @click="openStringTranslate"
                  >
                    <Languages class="size-4" aria-hidden="true" />
                  </ChoyButton>
                </div>
                <div class="flex min-w-0 items-start gap-1">
                  <ChoyTextField
                    class="min-w-0 flex-1"
                    :store="formStore"
                    prop="Help"
                    orientation="horizontal"
                    buffer-strategy="live"
                    :rows="2"
                    data-testid="choy-properties-definition-help"
                  />
                  <ChoyButton
                    v-if="!readonly"
                    type="button"
                    size="icon"
                    variant="ghost"
                    class="mt-0.5 size-8 shrink-0 text-muted-foreground hover:text-foreground"
                    :aria-label="_t('Translate help')"
                    :title="_t('Translate help')"
                    data-testid="choy-properties-definition-translate-help"
                    @click="openHelpTranslate"
                  >
                    <Languages class="size-4" aria-hidden="true" />
                  </ChoyButton>
                </div>
              </section>

              <section
                class="grid gap-2"
                data-testid="choy-properties-definition-section-constraints"
              >
                <div class="text-sm font-medium text-foreground">
                  {{ _t('Constraints') }}
                </div>
                <div class="flex min-w-0 items-center gap-3">
                  <ChoyVarcharField
                    class="min-w-0 flex-1"
                    :store="formStore"
                    prop="Default"
                    orientation="horizontal"
                    buffer-strategy="live"
                    data-testid="choy-properties-definition-default"
                  />
                  <ChoyBooleanField
                    class="shrink-0"
                    :store="formStore"
                    prop="Readonly"
                    widget="checkbox"
                    render-mode="form"
                    orientation="horizontal"
                    hide-label
                    :checkbox-label="_t('Readonly')"
                    buffer-strategy="live"
                    data-testid="choy-properties-definition-readonly"
                  />
                </div>
              </section>

              <section
                v-if="String(formData.Type || '') === 'selection'"
                class="flex flex-col gap-2"
                data-testid="choy-properties-definition-section-selection"
              >
                <div class="shrink-0 text-sm font-medium text-foreground">
                  {{ _t('Selection') }}
                </div>
                <div
                  class="min-h-0 max-h-[min(40vh,18rem)] overflow-y-auto"
                  data-testid="choy-properties-definition-selection-scroll"
                >
                  <ChoyPropertiesDefinitionSelectionEditor
                    :model-value="selectionRows"
                    :readonly="readonly"
                    @update:model-value="onSelectionRowsUpdate"
                    @translate="openSelectionTranslate"
                  />
                </div>
              </section>
            </div>
            <p
              v-if="formError || commitError"
              class="mt-2 text-sm text-destructive"
              data-testid="choy-properties-definition-item-error"
            >
              {{ formError || commitError }}
            </p>
          </template>
        </ChoyFormView>
      </div>
      <div
        class="dialog-footer flex shrink-0 flex-col-reverse gap-2 sm:flex-row sm:justify-end"
        data-slot="dialog-footer"
        data-testid="choy-properties-definition-item-footer"
      >
        <template v-if="readonly">
          <ChoyButton
            v-if="canEdit"
            type="button"
            size="default"
            variant="default"
            data-testid="choy-properties-definition-item-edit"
            @click="onRequestEdit"
          >
            {{ _t('Edit') }}
          </ChoyButton>
          <ChoyButton
            type="button"
            size="default"
            variant="outline"
            data-testid="choy-properties-definition-item-cancel"
            @click="onFooterCancel"
          >
            {{ _t('Close') }}
          </ChoyButton>
        </template>
        <template v-else>
          <ChoyButton
            type="button"
            size="default"
            variant="outline"
            data-testid="choy-properties-definition-item-cancel"
            @click="onFooterCancel"
          >
            {{ _t('Cancel') }}
          </ChoyButton>
          <ChoyButton
            type="button"
            size="default"
            variant="default"
            data-testid="choy-properties-definition-item-save"
            @click="onApply"
          >
            {{ _t('Apply') }}
          </ChoyButton>
        </template>
      </div>
    </ChoyDialogContent>
  </ChoyDialog>

  <ChoyPropertyLabelTranslationsDialog
    v-model="translateOpen"
    :label-title="translateTitle"
    :translations="translateMap"
    @saved="onTranslateSaved"
  />
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue';
import { getActivePinia } from 'pinia';
import { Languages } from 'lucide-vue-next';
import { TRANSLATED_BASE_LANG, unwrapPropertyLabel } from '@/core/service/orm/model/properties_label_i18n';
import { createTranslate } from '@/web/web/i18n';
import { confirmChoyAction } from '@/web/web/composables/confirmChoyAction';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import { createLocalFormStore, type LocalFormStore } from '@/web/web/stores/localFormStore';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyFormView from '@/web/web/components/view/ChoyFormView.vue';
import type { FormSubmitHandler } from '@/web/web/components/view/formViewTypes';
import type { ViewMode } from '@/web/web/components/view/viewScopeKeys';
import ChoyVarcharField from '@/web/web/components/field/ChoyVarcharField.vue';
import ChoyTextField from '@/web/web/components/field/ChoyTextField.vue';
import ChoyBooleanField from '@/web/web/components/field/ChoyBooleanField.vue';
import ChoySelectionField from '@/web/web/components/field/ChoySelectionField.vue';
import ChoyPropertyLabelTranslationsDialog from './ChoyPropertyLabelTranslationsDialog.vue';
import ChoyPropertiesDefinitionSelectionEditor from './ChoyPropertiesDefinitionSelectionEditor.vue';
import {
  PROPERTY_DEFINITION_V1_TYPE_OPTIONS,
  applyDraftCurrentLangToMaps,
  cloneDraftItem,
  selectionRowsFromDraft,
  syncDraftLabelsForLang,
  writeSelectionRowsToDraft,
  type DefinitionEditorDraftItem,
  type SelectionDraftRow,
} from './propertiesDefinitionHelpers';

defineOptions({ name: 'ChoyPropertiesDefinitionItemDialog' });

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyPropertiesDefinitionItemDialog' });

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    /** Draft row to edit; cloned into local state while open. */
    item?: DefinitionEditorDraftItem | null;
    readonly?: boolean;
    /**
     * When readonly, show an Edit control that asks the parent to reopen the
     * same row in mutate mode (parent must set readonly=false).
     */
    canEdit?: boolean;
    /** Dialog title override (defaults from item name). */
    title?: string;
    /** Parent commitDrafts error shown while dialog stays open after a failed save. */
    commitError?: string;
  }>(),
  {
    item: null,
    readonly: false,
    canEdit: false,
    title: '',
    commitError: '',
  },
);

const emit = defineEmits<{
  'update:modelValue': [boolean];
  saved: [item: DefinitionEditorDraftItem];
  /** Parent should clear readonly so this dialog switches to edit. */
  edit: [];
  closed: [];
}>();

const formViewRef = ref<{
  submit?: () => Promise<unknown>;
  cancel?: () => void | Promise<void>;
} | null>(null);
const local = ref<DefinitionEditorDraftItem | null>(null);
const formStore = shallowRef<LocalFormStore | null>(null);
const formInitialValues = shallowRef<Record<string, unknown>>({});
const formDataRef = shallowRef<Record<string, unknown>>({});
const formError = ref('');
/** Selection editor writes only to `local`; snapshot so Cancel can dirty-check it. */
const selectionBaselineJson = ref('');

const translateOpen = ref(false);
const translateTitle = ref('');
const translateMap = ref<Record<string, string>>({});
type TranslateTarget = { kind: 'string' } | { kind: 'help' } | { kind: 'selection'; value: string };
const translateTarget = ref<TranslateTarget | null>(null);

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
});

const formViewMode = computed<ViewMode>(() => (props.readonly ? 'display' : 'create'));

const dialogTitle = computed(() => {
  const override = String(props.title || '').trim();
  if (override) return override;
  const name = String(local.value?.name || props.item?.name || formDataRef.value?.Name || '').trim();
  if (props.readonly) {
    return name ? _t('Property: %s', name) : _t('Property');
  }
  return name ? _t('Edit property: %s', name) : _t('New property');
});

const selectionRows = computed((): SelectionDraftRow[] => {
  const item = local.value;
  if (!item || item.type !== 'selection') return [];
  return selectionRowsFromDraft(item);
});

function trackFormData(data: Record<string, unknown>): string {
  formDataRef.value = data;
  return '';
}

function editorLang(): string {
  if (!getActivePinia()) return TRANSLATED_BASE_LANG;
  try {
    const lang = String(useI18nStore().terminologyLang || '').trim();
    return lang || TRANSLATED_BASE_LANG;
  } catch {
    return TRANSLATED_BASE_LANG;
  }
}

function buildTypeSelection() {
  return PROPERTY_DEFINITION_V1_TYPE_OPTIONS.map(t => ({ value: t, label: t }));
}

function selectionSnapshot(item: DefinitionEditorDraftItem | null | undefined): string {
  if (!item) return '';
  return JSON.stringify({
    text: String(item.selectionText || ''),
    maps: item.selectionLabelMaps || {},
  });
}

function isSelectionDirty(): boolean {
  return selectionSnapshot(local.value) !== selectionBaselineJson.value;
}

function openFormFromDraft(item: DefinitionEditorDraftItem) {
  const initial = {
    Name: item.name || '',
    Type: item.type || 'char',
    Label: item.string || '',
    Help: item.help || '',
    Default: item.default || '',
    Readonly: item.readonly === true,
  };
  formInitialValues.value = initial;
  formStore.value = createLocalFormStore({
    storeId: `local-form:property-definition-item:${Date.now()}`,
    fields: [
      { name: 'Name', label: _t('Name'), type: 'varchar' },
      {
        name: 'Type',
        label: _t('Type'),
        type: 'selection',
        selection: buildTypeSelection(),
      },
      { name: 'Label', label: _t('Label'), type: 'varchar' },
      { name: 'Help', label: _t('Help'), type: 'text' },
      { name: 'Default', label: _t('Default'), type: 'varchar' },
      { name: 'Readonly', label: _t('Readonly'), type: 'boolean' },
    ],
    initialValues: initial,
  });
  formDataRef.value = { ...initial };
  selectionBaselineJson.value = selectionSnapshot(item);
}

function clearForm() {
  formStore.value = null;
  formInitialValues.value = {};
  formDataRef.value = {};
  selectionBaselineJson.value = '';
}

function onSelectionRowsUpdate(rows: SelectionDraftRow[]) {
  const item = local.value;
  if (!item || props.readonly) return;
  writeSelectionRowsToDraft(item, rows, editorLang());
}

function openStringTranslate() {
  const item = local.value;
  if (!item || props.readonly) return;
  mergeFormIntoDraft(item);
  applyDraftCurrentLangToMaps(item, editorLang());
  translateTarget.value = { kind: 'string' };
  translateTitle.value = item.name || _t('Label');
  translateMap.value = { ...item.stringMap };
  translateOpen.value = true;
}

function openHelpTranslate() {
  const item = local.value;
  if (!item || props.readonly) return;
  mergeFormIntoDraft(item);
  applyDraftCurrentLangToMaps(item, editorLang());
  translateTarget.value = { kind: 'help' };
  translateTitle.value = item.name ? `${item.name} · ${_t('Help')}` : _t('Help');
  translateMap.value = { ...item.helpMap };
  translateOpen.value = true;
}

function openSelectionTranslate(value: string) {
  const item = local.value;
  const opt = String(value || '').trim();
  if (!item || props.readonly || !opt) return;
  mergeFormIntoDraft(item);
  applyDraftCurrentLangToMaps(item, editorLang());
  translateTarget.value = { kind: 'selection', value: opt };
  translateTitle.value = `${item.name || _t('Label')}.${opt}`;
  translateMap.value = { ...(item.selectionLabelMaps[opt] || {}) };
  translateOpen.value = true;
}

function onTranslateSaved(next: Record<string, string>) {
  const target = translateTarget.value;
  const item = local.value;
  if (!target || !item) return;
  if (target.kind === 'string') {
    item.stringMap = { ...next };
    syncDraftLabelsForLang(item, editorLang());
    formDataRef.value.Label = item.string;
  } else if (target.kind === 'help') {
    item.helpMap = { ...next };
    syncDraftLabelsForLang(item, editorLang());
    formDataRef.value.Help = item.help;
  } else {
    const pendingRows = selectionRowsFromDraft(item);
    item.selectionLabelMaps = {
      ...item.selectionLabelMaps,
      [target.value]: { ...next },
    };
    const lang = editorLang();
    const freshLabel = unwrapPropertyLabel(next, lang);
    const restored = pendingRows.map(row =>
      String(row.value || '').trim() === target.value ? { ...row, label: freshLabel } : row,
    );
    writeSelectionRowsToDraft(item, restored, lang);
  }
  translateTarget.value = null;
}

function mergeFormIntoDraft(item: DefinitionEditorDraftItem) {
  const data = formDataRef.value || {};
  item.name = String(data.Name ?? item.name ?? '').trim();
  item.type = String(data.Type ?? item.type ?? 'char');
  item.string = String(data.Label ?? item.string ?? '');
  item.help = String(data.Help ?? item.help ?? '');
  item.default = String(data.Default ?? item.default ?? '');
  item.readonly = data.Readonly === true;
}

/** FormView cancelHandler: close after dirty confirmation (or when footer skips FormView). */
function onFormCancel() {
  visible.value = false;
}

async function confirmDiscardUnsaved(): Promise<boolean> {
  try {
    await confirmChoyAction(_t('You have unsaved changes. Discard them?'), _t('Unsaved changes'), {
      confirmText: _t('Discard'),
      cancelText: _t('Cancel'),
      destructive: true,
    });
    return true;
  } catch {
    return false;
  }
}

/**
 * Footer Cancel/Close. Selection edits live on `local` (outside FormView draft),
 * so they need their own dirty check. Form-field dirtiness still goes through
 * FormView.cancel. Readonly Close has nothing to discard.
 */
async function onFooterCancel() {
  if (props.readonly) {
    onFormCancel();
    return;
  }
  // Selection-only (or selection+form) dirty: one confirm, then close — avoid a
  // second FormView prompt when the form draft is also dirty.
  if (isSelectionDirty()) {
    if (!(await confirmDiscardUnsaved())) return;
    onFormCancel();
    return;
  }
  if (formViewRef.value?.cancel) {
    await formViewRef.value.cancel();
    return;
  }
  onFormCancel();
}

function onRequestEdit() {
  if (!props.canEdit || !props.readonly) return;
  emit('edit');
}

async function onApply() {
  if (props.readonly) return;
  try {
    if (formViewRef.value?.submit) {
      await formViewRef.value.submit();
      return;
    }
    await onFormSubmit({
      formData: formDataRef.value as any,
      mode: 'create',
      data: formDataRef.value as any,
      defaultSubmit: async () => null,
    });
  } catch (e) {
    formError.value = e instanceof Error ? e.message : String(e);
  }
}

const onFormSubmit: FormSubmitHandler<any> = async (_ctx) => {
  formError.value = '';
  const item = local.value;
  if (!item || props.readonly) {
    return { handled: true, skipSuccessMessage: true };
  }
  mergeFormIntoDraft(item);
  const name = String(item.name || '').trim();
  if (!name) {
    formError.value = _t('Name is required');
    return { handled: true, skipSuccessMessage: true };
  }
  item.name = name;
  try {
    applyDraftCurrentLangToMaps(item, editorLang());
  } catch (e) {
    formError.value = e instanceof Error ? e.message : String(e);
    return { handled: true, skipSuccessMessage: true };
  }
  emit('saved', cloneDraftItem(item));
  return { handled: true, skipSuccessMessage: true };
};

watch(
  () => props.modelValue,
  (open, wasOpen) => {
    if (open && !wasOpen) {
      formError.value = '';
      local.value = cloneDraftItem(props.item);
      syncDraftLabelsForLang(local.value, editorLang());
      openFormFromDraft(local.value);
    }
    if (!open && wasOpen) {
      local.value = null;
      clearForm();
      translateOpen.value = false;
      translateTarget.value = null;
      emit('closed');
    }
  },
  { immediate: true },
);

// Prefer store getField so in-place Type edits on the reactive draft are tracked
// (formDataRef is a shallowRef that may keep a stable object identity).
watch(
  () => formStore.value?.getField('Type'),
  type => {
    const item = local.value;
    if (!item || props.readonly || type == null) return;
    item.type = String(type);
  },
);
</script>

<style scoped>
/*
  Dialog form uses horizontal fields: pin label column width so Name/Type/Label/…
  share one gutter. Selection table is outside these field rows.
*/
.choy-properties-definition-item-form
  :deep([data-slot='field'][data-orientation='horizontal'] > [data-slot='field-label']) {
  width: 5.5rem;
  min-width: 5.5rem;
  flex: 0 0 5.5rem;
}

.choy-properties-definition-item-form
  :deep([data-slot='field'][data-orientation='horizontal']) {
  align-items: flex-start;
}

/* Embedded dialog: drop FormView page padding (p-5) — DialogContent already pads. */
.choy-properties-definition-item-form :deep([data-anchor='choy.form-view']) {
  padding: 0;
  overflow: visible;
}
</style>
