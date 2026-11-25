<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDialog v-model:open="visible">
    <ChoyDialogContent
      class="choy-field-translations-dialog max-w-xl"
      :show-close-button="false"
      :dismissible="dismissible"
      @open-auto-focus.prevent
    >
      <ChoyDialogHeader class="text-start">
        <ChoyDialogTitle>{{ dialogTitle }}</ChoyDialogTitle>
      </ChoyDialogHeader>
      <div class="choy-field-translations-dialog__body min-h-[120px]" :aria-busy="loading || undefined">
        <ChoyFormView
          v-if="formStore"
          ref="formViewRef"
          class="choy-field-translations-dialog__form flex flex-col gap-3.5"
          :store="formStore"
          view-mode="create"
          v-bind="embeddedFormChrome"
          :initial-values="formInitialValues"
          :submit-handler="onSubmit"
          :cancel-handler="cancelHandler"
        >
          <template #default>
            <div
              v-for="code in fieldCodes"
              :key="code"
              class="choy-field-translations-dialog__row"
            >
              <ChoyVarcharField
                :store="formStore"
                :prop="code"
                orientation="horizontal"
                buffer-strategy="live"
                class="choy-field-translations-dialog__field"
              />
              <div
                v-if="code === 'en_US'"
                class="choy-field-translations-dialog__hint mt-1 text-xs leading-snug text-muted-foreground"
              >
                {{ _t('Base language (cannot be deleted)') }}
              </div>
            </div>
          </template>
        </ChoyFormView>
      </div>
      <ChoyDialogFooter>
        <ChoyButton type="button" variant="outline" :disabled="saving" @click="onFooterCancel">{{
          _t('Cancel')
        }}</ChoyButton>
        <ChoyActionButton type="button" :busy="saving" @execute="onFooterSave">
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
import { useI18nStore } from '@/web/web/stores/i18nStore';
import { createUiModelStore, type UiModelStore } from '@/web/web/stores/uiModelStore';
import type { ModelStore } from '@/web/web/stores/modelStore';

defineOptions({ name: 'ChoyFieldTranslationsDialog' });

export type ChoyTranslationRow = {
  code: string;
  label: string;
  value: string;
  initial: string;
  existed: boolean;
};

type TranslationRowMeta = Pick<ChoyTranslationRow, 'initial' | 'existed'> & {
  label: string;
};

type TranslationRow = ChoyTranslationRow;

const props = defineProps<{
  modelValue: boolean;
  store: ModelStore<any>;
  recordId: string;
  fieldName: string;
  fieldLabel?: string;
  maxLength?: number;
  /** Unsaved form value for the current UI language; applied on open. */
  draftValue?: string;
  /** Optional lang override (e.g. zh_CN); defaults to store lang / activeLanguage.code. */
  draftLang?: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  saved: [value: string | null];
  closed: [];
}>();

const { _t } = createTranslate('web', { scope: 'web/components/field/FieldTranslationsDialog' });
const languageStore = createModelStore('base.Language');

const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
const formStore = shallowRef<UiModelStore | null>(null);
const formInitialValues = shallowRef<Record<string, unknown>>({});
const fieldCodes = ref<string[]>([]);
const rowMeta = ref(new Map<string, TranslationRowMeta>());

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
});

function closeDialog(): void {
  visible.value = false;
}

const saving = ref(false);
const { embeddedFormChrome, cancelHandler, onFooterSave, onFooterCancel, dismissible } = useChoyFormDialogShell({
  onClose: closeDialog,
  formViewRef,
  saving,
});

const dialogTitle = computed(() => {
  const label = String(props.fieldLabel || props.fieldName || '').trim();
  return label ? _t('Translate: %s', label) : _t('Translate field');
});

const loading = ref(false);

/** Prefer language display name (Odoo-style); fall back to code. */
function formatLanguageLabel(name: unknown, code: string): string {
  const display = String(name ?? '').trim();
  return display || code;
}

function resolveDraftLang(): string {
  const fromProp = String(props.draftLang || '').trim();
  if (fromProp) return fromProp;
  try {
    return String(useI18nStore().activeLanguage.code || '').trim();
  } catch {
    return '';
  }
}

/**
 * Overlay the form draft onto the current UI language row only.
 * Keep `initial` as the server value so Save still detects the draft as dirty.
 */
function applyDraftValue(byCode: Map<string, TranslationRow>) {
  if (props.draftValue === undefined) return;
  const lang = resolveDraftLang();
  if (!lang) return;
  const row = byCode.get(lang);
  if (!row) return;
  row.value = props.draftValue == null ? '' : String(props.draftValue);
}

function clearForm(): void {
  formStore.value = null;
  formInitialValues.value = {};
  fieldCodes.value = [];
  rowMeta.value = new Map();
}

function openFormFromRows(ordered: TranslationRow[]): void {
  const initialValues: Record<string, string> = {};
  const meta = new Map<string, TranslationRowMeta>();
  const codes: string[] = [];
  const fields: Array<{ name: string; label: string; type: 'varchar'; size?: number }> = [];

  for (const row of ordered) {
    codes.push(row.code);
    initialValues[row.code] = row.value;
    meta.set(row.code, { initial: row.initial, existed: row.existed, label: row.label });
    fields.push({
      name: row.code,
      label: row.label,
      type: 'varchar',
      ...(props.maxLength != null ? { size: props.maxLength } : {}),
    });
  }

  fieldCodes.value = codes;
  rowMeta.value = meta;
  formInitialValues.value = { ...initialValues };
  formStore.value = createUiModelStore({
    storeId: `ui-model:field-translations:${props.recordId}:${props.fieldName}:${Date.now()}`,
    fields,
    initialValues,
  });
}

async function loadRows() {
  loading.value = true;
  try {
    const [langs, map] = await Promise.all([
      (languageStore as any).GetActiveLanguages() as Promise<Array<{ Code?: string; Name?: string }>>,
      (props.store as any).GetFieldTranslations(props.recordId, props.fieldName) as Promise<Record<string, string>>,
    ]);
    const current = map && typeof map === 'object' ? map : {};
    const list = Array.isArray(langs) ? langs : [];
    const byCode = new Map<string, TranslationRow>();

    for (const lang of list) {
      const code = String(lang?.Code || '').trim();
      if (!code) continue;
      const existed = Object.prototype.hasOwnProperty.call(current, code);
      byCode.set(code, {
        code,
        label: formatLanguageLabel(lang?.Name, code),
        value: existed ? String(current[code] ?? '') : '',
        initial: existed ? String(current[code] ?? '') : '',
        existed,
      });
    }

    // Keep base language visible even if inactive somehow.
    if (!byCode.has('en_US')) {
      const existed = Object.prototype.hasOwnProperty.call(current, 'en_US');
      byCode.set('en_US', {
        code: 'en_US',
        label: formatLanguageLabel('English (US)', 'en_US'),
        value: existed ? String(current.en_US ?? '') : '',
        initial: existed ? String(current.en_US ?? '') : '',
        existed,
      });
    }

    applyDraftValue(byCode);

    const ordered = Array.from(byCode.values());
    ordered.sort((a, b) => {
      if (a.code === 'en_US') return -1;
      if (b.code === 'en_US') return 1;
      return a.label.localeCompare(b.label);
    });
    openFormFromRows(ordered);
  } catch (err: any) {
    ChoyMessage.error(String(err?.message || err || _t('Failed to load translations')));
    clearForm();
  } finally {
    loading.value = false;
  }
}

function onOpened() {
  void loadRows();
}

watch(
  () => props.modelValue,
  (open, wasOpen) => {
    if (open && !wasOpen) void onOpened();
    if (!open && wasOpen) {
      clearForm();
      emit('closed');
    }
  },
  { immediate: true },
);

async function performSave(formData: Record<string, unknown>) {
  saving.value = true;
  try {
    const patch: Record<string, string | false> = {};
    for (const code of fieldCodes.value) {
      const meta = rowMeta.value.get(code);
      if (!meta) continue;
      const next = String(formData[code] ?? '');
      if (meta.existed && next === meta.initial) continue;
      if (!meta.existed && next === '') continue;
      if (meta.existed && next === '' && code !== 'en_US') {
        patch[code] = false;
        continue;
      }
      if (meta.existed && next === '' && code === 'en_US') {
        patch[code] = '';
        continue;
      }
      patch[code] = next;
    }

    if (Object.keys(patch).length) {
      await (props.store as any).UpdateFieldTranslations(props.recordId, props.fieldName, patch);
    }

    const refreshed = (await (props.store as any).Browse(props.recordId, [props.fieldName])) as Record<string, unknown>;
    const nextValue = refreshed?.[props.fieldName];
    emit('saved', nextValue == null ? null : String(nextValue));
    ChoyMessage.success(_t('Translations saved'));
    closeDialog();
  } catch (err: any) {
    ChoyMessage.error(String(err?.message || err || _t('Failed to save translations')));
  } finally {
    saving.value = false;
  }
}

async function onSubmit(ctx: { formData: Record<string, unknown> }) {
  await performSave(ctx.formData || {});
  return { handled: true, skipSuccessMessage: true };
}

defineExpose({
  resolveDraftLang,
  fieldCodes,
  rowMeta,
  formStore,
});
</script>