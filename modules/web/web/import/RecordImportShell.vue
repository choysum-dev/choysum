<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ImportPanel
    v-model="open"
    :model="model"
    :company-id="companyId"
    :column-mapping="columnMapping"
    :upload-hint="uploadHint"
    @imported="emit('imported')"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ImportPanel from './ImportPanel.vue';
import { useRecordImportScope } from '@/web/web/composables/useRecordImportScope';
import type { RecordIoConfig } from '@/web/web/composables/recordIoTypes';

defineOptions({ name: 'RecordImportShell' });

const props = defineProps<{
  /** Direct model; optional when `config.model` is set. */
  model?: string;
  companyId?: string;
  columnMapping?: Record<string, string>;
  uploadHint?: string;
  /** Optional full config; direct props override when provided. */
  config?: RecordIoConfig;
}>();

const emit = defineEmits<{
  (e: 'imported'): void;
}>();

const open = defineModel<boolean>('open', { default: false });

const scopeConfig = computed<RecordIoConfig>(() => props.config ?? {
  model: String(props.model ?? '').trim(),
  import: {
    enabled: true,
    columnMapping: props.columnMapping,
    uploadHint: props.uploadHint,
  },
});

const scope = useRecordImportScope({ config: scopeConfig });

const model = computed(() => {
  const direct = String(props.model ?? '').trim();
  return direct || scope.model.value;
});
const companyId = computed(() => props.companyId ?? scope.companyId.value);
const columnMapping = computed(() => props.columnMapping ?? scope.columnMapping.value);
const uploadHint = computed(() => props.uploadHint ?? scope.uploadHint.value);
</script>
