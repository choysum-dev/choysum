<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ExportPanel
    v-model="open"
    :model="model"
    :company-id="resolvedCompanyId"
    :ids="ids"
    :domain="domain"
    :default-fields="defaultFields"
    :filtered-count="filteredCount"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ExportPanel from './ExportPanel.vue';
import {
  useRecordExportScope,
  type RecordExportListRef,
} from '@/web/web/composables/useRecordExportScope';

defineOptions({ name: 'RecordExportShell' });

const props = defineProps<{
  model: string;
  store: { storeId?: string; state?: { result?: { total?: number } } };
  /** Current list view instance (template ref value). */
  listRef?: RecordExportListRef;
  companyId?: string;
}>();

const open = defineModel<boolean>('open', { default: false });

const scope = useRecordExportScope({
  store: props.store,
  getListRef: () => props.listRef ?? null,
});

const resolvedCompanyId = computed(() => props.companyId?.trim() || scope.companyId.value);
const ids = scope.ids;
const domain = scope.domain;
const defaultFields = scope.defaultFields;
const filteredCount = scope.filteredCount;
</script>
