<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-dialog
    v-model="visible"
    :title="dialogTitle"
    width="560px"
    append-to-body
    destroy-on-close
    class="o-field-translations-dialog"
    @opened="onOpened"
    @closed="emit('closed')"
  >
    <div v-loading="loading" class="o-field-translations-dialog__body">
      <el-form label-position="left" label-width="168px" @submit.prevent>
        <el-form-item v-for="row in rows" :key="row.code" :label="row.label">
          <el-input
            v-model="row.value"
            :maxlength="maxLength ?? undefined"
            :show-word-limit="maxLength != null"
            clearable
          />
          <div v-if="row.code === 'en_US'" class="o-field-translations-dialog__hint">
            {{ _t('Base language (cannot be deleted)') }}
          </div>
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="visible = false">{{ _t('Cancel') }}</el-button>
      <el-button type="primary" :loading="saving" native-type="button" @click="handleSave">
        {{ _t('Save translations') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElMessage } from 'element-plus';
import { createTranslate } from '@/web/web/i18n';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';

defineOptions({ name: 'OFieldTranslationsDialog' });

type TranslationRow = {
  code: string;
  label: string;
  value: string;
  initial: string;
  existed: boolean;
};

const props = defineProps<{
  modelValue: boolean;
  store: WebModelStore<any>;
  recordId: string;
  fieldName: string;
  fieldLabel?: string;
  maxLength?: number;
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  saved: [value: string | null];
  closed: [];
}>();

const { _t } = createTranslate('web', { scope: 'web/components/field/OFieldTranslationsDialog' });
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

    const ordered = Array.from(byCode.values());
    ordered.sort((a, b) => {
      if (a.code === 'en_US') return -1;
      if (b.code === 'en_US') return 1;
      return a.label.localeCompare(b.label);
    });
    rows.value = ordered;
  } catch (err: any) {
    ElMessage.error(String(err?.message || err || _t('Failed to load translations')));
    rows.value = [];
  } finally {
    loading.value = false;
  }
}

function onOpened() {
  void loadRows();
}

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
    ElMessage.success(_t('Translations saved'));
    visible.value = false;
  } catch (err: any) {
    ElMessage.error(String(err?.message || err || _t('Failed to save translations')));
  } finally {
    saving.value = false;
  }
}
</script>

<style scoped>
.o-field-translations-dialog__body {
  min-height: 120px;
}
.o-field-translations-dialog__body :deep(.el-form-item) {
  margin-bottom: 14px;
  align-items: flex-start;
}
.o-field-translations-dialog__body :deep(.el-form-item__label) {
  line-height: 32px;
  color: var(--el-text-color-regular);
  justify-content: flex-start;
  text-align: left;
  padding-right: 12px;
}
.o-field-translations-dialog__body :deep(.el-form-item__content) {
  flex: 1 1 auto;
  min-width: 0;
}
.o-field-translations-dialog__hint {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--el-text-color-secondary);
}
</style>
