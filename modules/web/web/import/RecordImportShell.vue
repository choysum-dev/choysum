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
  /** Fully qualified model name for ImportHub. */
  model?: string;
  companyId?: string;
  columnMapping?: Record<string, string>;
  uploadHint?: string;
  /** Optional import capability options; does not carry model. */
  config?: RecordIoConfig;
}>();

const emit = defineEmits<{
  (e: 'imported'): void;
}>();

const open = defineModel<boolean>('open', { default: false });

const scope = useRecordImportScope({
  model: () => props.model,
  config: () => props.config,
});

const model = computed(() => scope.model.value);
const companyId = computed(() => props.companyId?.trim() || scope.companyId.value);
const columnMapping = computed(() => props.columnMapping ?? scope.columnMapping.value);
const uploadHint = computed(() => props.uploadHint ?? scope.uploadHint.value);
</script>
