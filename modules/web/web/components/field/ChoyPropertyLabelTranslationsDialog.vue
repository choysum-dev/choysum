<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDialog v-model:open="visible">
    <ChoyDialogContent class="choy-property-label-translations-dialog max-w-xl" @open-auto-focus.prevent>
      <ChoyDialogTitle>{{ dialogTitle }}</ChoyDialogTitle>
      <div class="min-h-[120px]" :aria-busy="loading || undefined">
        <form class="flex flex-col gap-3.5" @submit.prevent>
          <div
            v-for="row in rows"
            :key="row.code"
            class="grid grid-cols-[168px_1fr] items-start gap-3"
          >
            <label class="leading-8 text-foreground">{{ row.label }}</label>
            <div>
              <ChoyInput
                v-model="row.value"
                class="box-border min-h-8 w-full min-w-0"
                data-testid="choy-property-label-translation-input"
              />
              <div
                v-if="row.code === 'en_US'"
                class="mt-1 text-xs leading-snug text-muted-foreground"
              >
                {{ _t('Base language (cannot be deleted)') }}
              </div>
            </div>
          </div>
        </form>
      </div>
      <div class="mt-4 flex justify-end gap-2">
        <ChoyButton type="button" variant="outline" @click="visible = false">{{ _t('Cancel') }}</ChoyButton>
        <ChoyButton type="button" data-testid="choy-property-label-translations-save" @click="handleSave">
          {{ _t('Save translations') }}
        </ChoyButton>
      </div>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyInput from '@/web/web/components/vendor/ui/input/Input.vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { createTranslate } from '@/web/web/i18n';
import { createStoreByModel } from '@/web/web/stores/registry';
import { TRANSLATED_BASE_LANG } from '@/core/service/orm/repository/projection/translated_field_codec';

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

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
});

const dialogTitle = computed(() => {
  const label = String(props.labelTitle || '').trim();
  return label ? _t('Translate: %s', label) : _t('Translate label');
});

const loading = ref(false);
const rows = ref<TranslationRow[]>([]);

function formatLanguageLabel(name: unknown, code: string): string {
  const display = String(name ?? '').trim();
  return display || code;
}

async function loadRows() {
  loading.value = true;
  try {
    const current = props.translations && typeof props.translations === 'object' ? { ...props.translations } : {};
    let list: Array<{ Code?: string; Name?: string }> = [];
    try {
      const languageStore = createStoreByModel('base.Language');
      list = (await (languageStore as any).GetActiveLanguages()) as Array<{ Code?: string; Name?: string }>;
    } catch {
      list = [];
    }
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
    rows.value = ordered;
  } catch (err: any) {
    ChoyMessage.error(String(err?.message || err || _t('Failed to load translations')));
    rows.value = [];
  } finally {
    loading.value = false;
  }
}

watch(
  () => props.modelValue,
  (open, wasOpen) => {
    if (open && !wasOpen) void loadRows();
    if (!open && wasOpen) emit('closed');
  },
  { immediate: true },
);

function handleSave() {
  const next: Record<string, string> = {};
  for (const row of rows.value) {
    const value = String(row.value ?? '');
    if (row.code === TRANSLATED_BASE_LANG || value !== '') {
      next[row.code] = value;
    }
  }
  if (!Object.prototype.hasOwnProperty.call(next, TRANSLATED_BASE_LANG)) {
    next[TRANSLATED_BASE_LANG] = '';
  }
  emit('saved', next);
  visible.value = false;
}
</script>
