<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDialog v-model:open="visible">
    <ChoyDialogContent
      class="choy-properties-definition-item-dialog flex max-h-[min(85vh,40rem)] w-[min(90vw,40rem)] max-w-[min(90vw,40rem)] flex-col gap-3 overflow-hidden"
      data-testid="choy-properties-definition-item-dialog"
      @open-auto-focus.prevent
    >
      <ChoyDialogTitle class="shrink-0">{{ dialogTitle }}</ChoyDialogTitle>
      <div v-if="local" class="min-h-0 flex-1 overflow-auto">
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="grid gap-1.5">
            <FieldLabel :for="nameInputId">{{ _t('Name') }}</FieldLabel>
            <Input
              :id="nameInputId"
              v-model="local.name"
              class="w-full"
              :disabled="readonly"
              data-testid="choy-properties-definition-name"
            />
          </div>
          <div class="grid gap-1.5">
            <FieldLabel :for="typeInputId">{{ _t('Type') }}</FieldLabel>
            <Select v-model="local.type" :disabled="readonly">
              <SelectTrigger
                :id="typeInputId"
                class="w-full"
                data-testid="choy-properties-definition-type"
              />
              <SelectContent>
                <SelectItem v-for="t in typeOptions" :key="t" :value="t">
                  {{ t }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="grid gap-1.5 sm:col-span-2">
            <FieldLabel :for="labelInputId">{{ _t('Label') }}</FieldLabel>
            <div class="flex min-w-0 items-center gap-1">
              <Input
                :id="labelInputId"
                v-model="local.string"
                class="min-w-0 flex-1"
                :disabled="readonly"
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
          </div>
          <div class="grid gap-1.5 sm:col-span-2">
            <FieldLabel :for="helpInputId">{{ _t('Help') }}</FieldLabel>
            <div class="flex min-w-0 items-start gap-1">
              <Textarea
                :id="helpInputId"
                v-model="local.help"
                class="min-h-[4.5em] min-w-0 flex-1"
                :disabled="readonly"
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
          </div>
          <div class="grid gap-1.5 sm:col-span-2">
            <FieldLabel :for="defaultInputId">{{ _t('Default') }}</FieldLabel>
            <Input
              :id="defaultInputId"
              v-model="local.default"
              class="w-full"
              :disabled="readonly"
              data-testid="choy-properties-definition-default"
            />
          </div>
          <label class="flex items-center gap-2 text-sm sm:col-span-2">
            <Checkbox
              :model-value="local.readonly"
              :disabled="readonly"
              data-testid="choy-properties-definition-readonly"
              @update:model-value="local.readonly = $event === true"
            />
            {{ _t('Readonly') }}
          </label>
          <div v-if="local.type === 'selection'" class="grid gap-1.5 sm:col-span-2">
            <FieldLabel>{{ _t('Selection') }}</FieldLabel>
            <ChoyPropertiesDefinitionSelectionEditor
              :model-value="selectionRows"
              :readonly="readonly"
              @update:model-value="onSelectionRowsUpdate"
              @translate="openSelectionTranslate"
            />
          </div>
        </div>
        <p
          v-if="formError || commitError"
          class="mt-3 text-sm text-destructive"
          data-testid="choy-properties-definition-item-error"
        >
          {{ formError || commitError }}
        </p>
      </div>
      <div class="flex shrink-0 justify-end gap-2">
        <ChoyButton type="button" variant="outline" data-testid="choy-properties-definition-item-cancel" @click="onCancel">
          {{ _t('Cancel') }}
        </ChoyButton>
        <ChoyButton
          v-if="!readonly"
          type="button"
          data-testid="choy-properties-definition-item-save"
          @click="onConfirm"
        >
          {{ _t('Save') }}
        </ChoyButton>
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
import { computed, ref, watch } from 'vue';
import { getActivePinia } from 'pinia';
import { Languages } from 'lucide-vue-next';
import { TRANSLATED_BASE_LANG } from '@/core/service/orm/model/properties_label_i18n';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';
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
import {
  PROPERTY_DEFINITION_V1_TYPE_OPTIONS,
  applyDraftCurrentLangToMaps,
  cloneDraftItem,
  emptyDraftItem,
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
    /** Dialog title override (defaults from item name). */
    title?: string;
    /** Parent commitDrafts error shown while dialog stays open after a failed save. */
    commitError?: string;
  }>(),
  {
    item: null,
    readonly: false,
    title: '',
    commitError: '',
  },
);

const emit = defineEmits<{
  'update:modelValue': [boolean];
  saved: [item: DefinitionEditorDraftItem];
  closed: [];
}>();

const typeOptions = PROPERTY_DEFINITION_V1_TYPE_OPTIONS;
const local = ref<DefinitionEditorDraftItem | null>(null);
const formError = ref('');
const nameInputId = 'choy-properties-definition-name';
const typeInputId = 'choy-properties-definition-type';
const labelInputId = 'choy-properties-definition-label';
const helpInputId = 'choy-properties-definition-help';
const defaultInputId = 'choy-properties-definition-default';

const translateOpen = ref(false);
const translateTitle = ref('');
const translateMap = ref<Record<string, string>>({});
type TranslateTarget = { kind: 'string' } | { kind: 'help' } | { kind: 'selection'; value: string };
const translateTarget = ref<TranslateTarget | null>(null);

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
});

const dialogTitle = computed(() => {
  const override = String(props.title || '').trim();
  if (override) return override;
  const name = String(local.value?.name || props.item?.name || '').trim();
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

function editorLang(): string {
  if (!getActivePinia()) return TRANSLATED_BASE_LANG;
  try {
    const lang = String(useI18nStore().terminologyLang || '').trim();
    return lang || TRANSLATED_BASE_LANG;
  } catch {
    return TRANSLATED_BASE_LANG;
  }
}

function onSelectionRowsUpdate(rows: SelectionDraftRow[]) {
  const item = local.value;
  if (!item || props.readonly) return;
  writeSelectionRowsToDraft(item, rows, editorLang());
}

function openStringTranslate() {
  const item = local.value;
  if (!item || props.readonly) return;
  applyDraftCurrentLangToMaps(item, editorLang());
  translateTarget.value = { kind: 'string' };
  translateTitle.value = item.name || _t('Label');
  translateMap.value = { ...item.stringMap };
  translateOpen.value = true;
}

function openHelpTranslate() {
  const item = local.value;
  if (!item || props.readonly) return;
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
  } else if (target.kind === 'help') {
    item.helpMap = { ...next };
    syncDraftLabelsForLang(item, editorLang());
  } else {
    item.selectionLabelMaps = {
      ...item.selectionLabelMaps,
      [target.value]: { ...next },
    };
    syncDraftLabelsForLang(item, editorLang());
  }
  translateTarget.value = null;
}

function onCancel() {
  visible.value = false;
}

function onConfirm() {
  formError.value = '';
  const item = local.value;
  if (!item || props.readonly) return;
  const name = String(item.name || '').trim();
  if (!name) {
    formError.value = _t('Name is required');
    return;
  }
  try {
    applyDraftCurrentLangToMaps(item, editorLang());
  } catch (e) {
    formError.value = e instanceof Error ? e.message : String(e);
    return;
  }
  // Parent closes on successful commit; keep open when commitDrafts fails so edits are not lost.
  emit('saved', cloneDraftItem(item));
}

watch(
  () => props.modelValue,
  (open, wasOpen) => {
    if (open && !wasOpen) {
      formError.value = '';
      local.value = cloneDraftItem(props.item);
      syncDraftLabelsForLang(local.value, editorLang());
    }
    if (!open && wasOpen) {
      local.value = null;
      translateOpen.value = false;
      translateTarget.value = null;
      emit('closed');
    }
  },
  { immediate: true },
);
</script>
