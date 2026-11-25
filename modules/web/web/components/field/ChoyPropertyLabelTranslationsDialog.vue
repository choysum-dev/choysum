<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDialog v-model:open="visible">
    <ChoyDialogContent
      class="choy-property-label-translations-dialog max-w-xl"
      :show-close-button="false"
      :dismissible="dismissible"
      @open-auto-focus.prevent
    >
      <ChoyDialogHeader class="text-start">
        <ChoyDialogTitle>{{ dialogTitle }}</ChoyDialogTitle>
      </ChoyDialogHeader>
      <div class="min-h-[120px]" :aria-busy="loading || undefined">
        <ChoyFormView
          v-if="formStore"
          ref="formViewRef"
          class="flex flex-col gap-3.5"
          :store="formStore"
          view-mode="create"
          v-bind="embeddedFormChrome"
          :initial-values="formInitialValues"
          :submit-handler="onSubmit"
          :cancel-handler="cancelHandler"
        >
          <template #default>
            <div v-for="code in fieldCodes" :key="code">
              <ChoyVarcharField
                :store="formStore"
                :prop="code"
                orientation="horizontal"
                buffer-strategy="live"
                data-testid="choy-property-label-translation-input"
              />
              <div
                v-if="code === TRANSLATED_BASE_LANG"
                class="mt-1 text-xs leading-snug text-muted-foreground"
              >
                {{ _t('Base language (cannot be deleted)') }}
              </div>
            </div>
          </template>
        </ChoyFormView>
      </div>
      <ChoyDialogFooter>
        <ChoyButton type="button" variant="outline" @click="onFooterCancel">{{ _t('Cancel') }}</ChoyButton>
        <ChoyActionButton
          type="button"
          data-testid="choy-property-label-translations-save"
          :disabled="fieldCodes.length === 0"
          :busy="loading"
          :requires-connectivity="false"
          @execute="onFooterSave"
        >
          {{ _t('Save translations') }}
        </ChoyActionButton>
      </ChoyDialogFooter>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue';
import {
  ChoyDialog,
  ChoyDialogContent,
  ChoyDialogFooter,
  ChoyDialogHeader,
  ChoyDialogTitle,
} from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/button/ChoyButton.vue';
import ChoyActionButton from '@/web/web/components/button/ChoyActionButton.vue';
import ChoyFormView from '@/web/web/components/view/ChoyFormView.vue';
import ChoyVarcharField from '@/web/web/components/field/ChoyVarcharField.vue';
import { useChoyFormDialogShell, type ChoyFormDialogViewApi } from '@/web/web/composables/useChoyFormDialogShell';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { createTranslate } from '@/web/web/i18n';
import { createModelStore } from '@/web/web/stores/registry';
import { TRANSLATED_BASE_LANG } from '@/core/service/orm/model/properties_label_i18n';
import { createUiModelStore, type UiModelStore } from '@/web/web/stores/uiModelStore';

defineOptions({ name: 'ChoyPropertyLabelTranslationsDialog' });

type TranslationRow = {
  code: string;
  label: string;
  value: string;
};

const props = defineProps<{
  modelValue: boolean;
  /** Dialog title suffix (property name / option value). */
  labelTitle?: string;
  /** Current lang map being edited (in-memory; not a DB translate field). */
  translations?: Record<string, string> | null;
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  saved: [translations: Record<string, string>];
  closed: [];
}>();

const { _t } = createTranslate('web', { scope: 'web/components/field/PropertyLabelTranslationsDialog' });

const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
const formStore = shallowRef<UiModelStore | null>(null);
const formInitialValues = shallowRef<Record<string, unknown>>({});
const fieldCodes = ref<string[]>([]);
let formStoreSeq = 0;

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
});

function closeDialog(): void {
  visible.value = false;
}

const { embeddedFormChrome, cancelHandler, onFooterSave, onFooterCancel, dismissible } = useChoyFormDialogShell({
  onClose: closeDialog,
  formViewRef,
});

const dialogTitle = computed(() => {
  const label = String(props.labelTitle || '').trim();
  return label ? _t('Translate: %s', label) : _t('Translate label');
});

const loading = ref(false);
/** Bumped on each load / close so a late GetActiveLanguages cannot overwrite a newer open. */
let loadGeneration = 0;

function formatLanguageLabel(name: unknown, code: string): string {
  const display = String(name ?? '').trim();
  return display || code;
}

function clearForm(): void {
  formStore.value = null;
  formInitialValues.value = {};
  fieldCodes.value = [];
}

function openFormFromRows(ordered: TranslationRow[]): void {
  const initialValues: Record<string, string> = {};
  const codes: string[] = [];
  const fields: Array<{ name: string; label: string; type: 'varchar' }> = [];

  for (const row of ordered) {
    codes.push(row.code);
    initialValues[row.code] = row.value;
    fields.push({ name: row.code, label: row.label, type: 'varchar' });
  }

  fieldCodes.value = codes;
  formInitialValues.value = { ...initialValues };
  formStore.value = createUiModelStore({
    storeId: `ui-model:property-label-translations:${++formStoreSeq}`,
    fields,
    initialValues,
  });
}

async function loadRows() {
  const gen = ++loadGeneration;
  loading.value = true;
  try {
    const current = props.translations && typeof props.translations === 'object' ? { ...props.translations } : {};
    let list: Array<{ Code?: string; Name?: string }> = [];
    try {
      const languageStore = createModelStore('base.Language');
      list = (await (languageStore as any).GetActiveLanguages()) as Array<{ Code?: string; Name?: string }>;
    } catch {
      list = [];
    }
    if (gen !== loadGeneration) return;
    const byCode = new Map<string, TranslationRow>();
    for (const lang of Array.isArray(list) ? list : []) {
      const code = String(lang?.Code || '').trim();
      if (!code) continue;
      byCode.set(code, {
        code,
        label: formatLanguageLabel(lang?.Name, code),
        value: Object.prototype.hasOwnProperty.call(current, code) ? String(current[code] ?? '') : '',
      });
    }
    if (!byCode.has(TRANSLATED_BASE_LANG)) {
      byCode.set(TRANSLATED_BASE_LANG, {
        code: TRANSLATED_BASE_LANG,
        label: formatLanguageLabel('English (US)', TRANSLATED_BASE_LANG),
        value: Object.prototype.hasOwnProperty.call(current, TRANSLATED_BASE_LANG)
          ? String(current[TRANSLATED_BASE_LANG] ?? '')
          : '',
      });
    }
    // Include any extra keys already present on the map.
    for (const [code, value] of Object.entries(current)) {
      if (byCode.has(code)) continue;
      byCode.set(code, { code, label: code, value: String(value ?? '') });
    }
    const ordered = Array.from(byCode.values());
    ordered.sort((a, b) => {
      if (a.code === TRANSLATED_BASE_LANG) return -1;
      if (b.code === TRANSLATED_BASE_LANG) return 1;
      return a.label.localeCompare(b.label);
    });
    openFormFromRows(ordered);
  } catch (err: any) {
    if (gen !== loadGeneration) return;
    ChoyMessage.error(String(err?.message || err || _t('Failed to load translations')));
    clearForm();
  } finally {
    if (gen === loadGeneration) loading.value = false;
  }
}

watch(
  () => props.modelValue,
  (open, wasOpen) => {
    if (open && !wasOpen) void loadRows();
    if (!open && wasOpen) {
      loadGeneration += 1;
      clearForm();
      loading.value = false;
      emit('closed');
    }
  },
  { immediate: true },
);

function buildSavedMap(formData: Record<string, unknown>): Record<string, string> | null {
  if (loading.value || fieldCodes.value.length === 0) {
    return null;
  }
  const next: Record<string, string> = {};
  for (const code of fieldCodes.value) {
    const value = String(formData[code] ?? '');
    if (code === TRANSLATED_BASE_LANG || value !== '') {
      next[code] = value;
    }
  }
  if (!Object.prototype.hasOwnProperty.call(next, TRANSLATED_BASE_LANG)) {
    next[TRANSLATED_BASE_LANG] = '';
  }
  return next;
}

function handleSave(formData?: Record<string, unknown>) {
  const data = formData ?? formStore.value?.getValues() ?? {};
  const next = buildSavedMap(data);
  if (next == null) {
    closeDialog();
    return;
  }
  emit('saved', next);
  closeDialog();
}

async function onSubmit(ctx: { formData: Record<string, unknown> }) {
  handleSave(ctx.formData || {});
  return { handled: true, skipSuccessMessage: true };
}

defineExpose({
  handleSave,
  fieldCodes,
  formStore,
});
</script>