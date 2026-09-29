<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Dialog v-model:open="visible">
    <DialogContent class="choy-field-translations-dialog max-w-xl" @open-auto-focus.prevent>
      <DialogTitle>{{ dialogTitle }}</DialogTitle>
      <div class="choy-field-translations-dialog__body" :aria-busy="loading || undefined">
        <form class="choy-field-translations-dialog__form" @submit.prevent>
          <div v-for="row in rows" :key="row.code" class="choy-field-translations-dialog__row">
            <label class="choy-field-translations-dialog__label">{{ row.label }}</label>
            <div class="choy-field-translations-dialog__control">
              <input
                v-model="row.value"
                class="choy-field-translations-dialog__input"
                :maxlength="maxLength ?? undefined"
              />
              <div v-if="row.code === 'en_US'" class="choy-field-translations-dialog__hint">
                {{ _t('Base language (cannot be deleted)') }}
              </div>
            </div>
          </div>
        </form>
      </div>
      <div class="dialog-footer">
        <ChoyButton type="button" variant="outline" @click="visible = false">{{ _t('Cancel') }}</ChoyButton>
        <ChoyButton type="button" :disabled="saving" @click="handleSave">
          {{ _t('Save translations') }}
        </ChoyButton>
      </div>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import Dialog from '@/web/web/components/vendor/ui/dialog/Dialog.vue';
import DialogContent from '@/web/web/components/vendor/ui/dialog/DialogContent.vue';
import DialogTitle from '@/web/web/components/vendor/ui/dialog/DialogTitle.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { createTranslate } from '@/web/web/i18n';
import { createStoreByModel } from '@/web/web/stores/registry';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import type { WebModelStore } from '@/web/web/stores/modelStore';

defineOptions({ name: 'ChoyFieldTranslationsDialog' });

export type ChoyTranslationRow = {
  code: string;
  label: string;
  value: string;
  initial: string;
  existed: boolean;
};

type TranslationRow = ChoyTranslationRow;

const props = defineProps<{
  modelValue: boolean;
  store: WebModelStore<any>;
  recordId: string;
  fieldName: string;
  fieldLabel?: string;
  maxLength?: number;
  /** Unsaved form value for the current UI language; applied on open. */
  draftValue?: string;
  /** Optional lang override (e.g. zh_CN); defaults to i18n terminologyLang. */
  draftLang?: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  saved: [value: string | null];
  closed: [];
}>();

const { _t } = createTranslate('web', { scope: 'web/components/field/FieldTranslationsDialog' });
const languageStore = createStoreByModel('base.Language');

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
});

const dialogTitle = computed(() => {
  const label = String(props.fieldLabel || props.fieldName || '').trim();
  return label ? _t('Translate: %s', label) : _t('Translate field');
});

const loading = ref(false);
const saving = ref(false);
const rows = ref<TranslationRow[]>([]);

/** Prefer language display name (Odoo-style); fall back to code. */
function formatLanguageLabel(name: unknown, code: string): string {
  const display = String(name ?? '').trim();
  return display || code;
}

function resolveDraftLang(): string {
  const fromProp = String(props.draftLang || '').trim();
  if (fromProp) return fromProp;
  try {
    return String(useI18nStore().terminologyLang || '').trim();
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
    rows.value = ordered;
  } catch (err: any) {
    ChoyMessage.error(String(err?.message || err || _t('Failed to load translations')));
    rows.value = [];
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
    if (!open && wasOpen) emit('closed');
  },
  { immediate: true },
);

async function handleSave() {
  saving.value = true;
  try {
    const patch: Record<string, string | false> = {};
    for (const row of rows.value) {
      const next = row.value;
      if (row.existed && next === row.initial) continue;
      if (!row.existed && next === '') continue;
      if (row.existed && next === '' && row.code !== 'en_US') {
        // Clearing a non-base translation removes the key (D12 uses false for delete).
        // Keep empty string for en_US as an explicit empty value.
        patch[row.code] = false;
        continue;
      }
      if (row.existed && next === '' && row.code === 'en_US') {
        patch[row.code] = '';
        continue;
      }
      patch[row.code] = next;
    }

    if (Object.keys(patch).length) {
      await (props.store as any).UpdateFieldTranslations(props.recordId, props.fieldName, patch);
    }

    const refreshed = (await (props.store as any).Browse(props.recordId, [props.fieldName])) as Record<string, unknown>;
    const nextValue = refreshed?.[props.fieldName];
    emit('saved', nextValue == null ? null : String(nextValue));
    ChoyMessage.success(_t('Translations saved'));
    visible.value = false;
  } catch (err: any) {
    ChoyMessage.error(String(err?.message || err || _t('Failed to save translations')));
  } finally {
    saving.value = false;
  }
}
</script>

<style scoped>
.choy-field-translations-dialog__body {
  min-height: 120px;
}
.choy-field-translations-dialog__form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.choy-field-translations-dialog__row {
  display: grid;
  grid-template-columns: 168px 1fr;
  gap: 12px;
  align-items: start;
}
.choy-field-translations-dialog__label {
  line-height: 32px;
  color: var(--el-text-color-regular);
}
.choy-field-translations-dialog__input {
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
}
.choy-field-translations-dialog__hint {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--el-text-color-secondary);
}
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
