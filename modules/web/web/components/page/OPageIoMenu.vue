<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-dropdown
    v-if="visibleItems.length"
    trigger="click"
    placement="bottom-end"
    @command="onCommand"
  >
    <el-button
      text
      class="o-page-io-menu__trigger"
      :aria-label="menuAriaLabel"
      data-test="page-io-menu-trigger"
    >
      <el-icon :size="18"><Setting /></el-icon>
    </el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="item in visibleItems"
          :key="item.key"
          :command="item.key"
          :disabled="item.disabled"
          :data-test="`page-io-menu-${item.key}`"
        >
          {{ item.label }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>

  <RecordImportShell
    v-if="importEnabled"
    v-model:open="importOpen"
    :model="resolvedModel"
    :config="importShellConfig"
    :company-id="companyId"
    @imported="onImported"
  />
  <RecordExportShell
    v-if="exportEnabled"
    v-model:open="exportOpen"
    :model="resolvedModel"
    :store="resolvedStore!"
    :list-ref="listRef"
    :company-id="companyId"
  />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { Setting } from '@element-plus/icons-vue';
import { createTranslate } from '@/web/web/i18n';
import { useRecordIoMenu } from '@/web/web/composables/useRecordIoMenu';
import type { PageIoMenuItem, RecordIoConfig } from '@/web/web/composables/recordIoTypes';
import type { RecordExportListRef } from '@/web/web/composables/useRecordExportScope';
import { useResolvedOptionalPageStore } from '@/web/web/composables/usePageContext';
import { RecordImportShell } from '@/web/web/import';
import { RecordExportShell } from '@/web/web/export';

defineOptions({ name: 'OPageIoMenu' });

export type OPageIoMenuListRef = RecordExportListRef & {
  refresh?: () => Promise<void> | void;
};

type PageIoStore = {
  storeId?: string;
  fullModelName?: string;
  state?: { result?: { total?: number } };
};

const props = defineProps<{
  /** Low-level menu entries. When omitted, items are derived from import/export flags. */
  items?: PageIoMenuItem[];
  /** Enable Import panel and menu item. */
  import?: boolean;
  /** Enable Export panel and menu item. */
  export?: boolean;
  /** Optional CSV upload hint for ImportPanel. */
  importUploadHint?: string;
  /** Optional default column mapping for ImportPanel. */
  importColumnMapping?: Record<string, string>;
  /** List/kanban store; falls back to OPage provided store when omitted. */
  store?: PageIoStore;
  /** Current list/kanban view instance (template ref value). */
  listRef?: OPageIoMenuListRef | null;
  /** Optional company override for import/export panels. */
  companyId?: string;
}>();

const emit = defineEmits<{
  (e: 'imported'): void;
}>();

const { _t } = createTranslate('web', { scope: 'web/components/page/OPageIoMenu' });
const menuAriaLabel = _t('Import and export');

const importOpen = ref(false);
const exportOpen = ref(false);

const resolvedStore = useResolvedOptionalPageStore<PageIoStore>(() => props.store);
const resolvedModel = computed(() => String(resolvedStore.value?.fullModelName ?? '').trim());

const requestedConfig = computed<RecordIoConfig>(() => {
  const config: RecordIoConfig = {};
  if (props.import) {
    config.import = {
      enabled: true,
      uploadHint: props.importUploadHint,
      columnMapping: props.importColumnMapping,
    };
  }
  if (props.export) {
    config.export = { enabled: true };
  }
  return config;
});

const importEnabled = computed(() => !!props.import && !!resolvedModel.value);
const exportEnabled = computed(
  () => !!props.export && !!resolvedStore.value && !!resolvedModel.value,
);

const menuConfig = computed<RecordIoConfig>(() => {
  const base = requestedConfig.value;
  let next = base;
  if (base.export?.enabled && (!resolvedStore.value || !resolvedModel.value)) {
    next = { ...next, export: { ...base.export, enabled: false } };
  }
  if (base.import?.enabled && !resolvedModel.value) {
    next = { ...next, import: { ...base.import, enabled: false } };
  }
  return next;
});

const importShellConfig = computed<RecordIoConfig>(() => ({
  import: {
    enabled: true,
    uploadHint: props.importUploadHint,
    columnMapping: props.importColumnMapping,
  },
}));

const { items: configItems } = useRecordIoMenu({
  config: menuConfig,
  openImport: () => {
    importOpen.value = true;
  },
  openExport: () => {
    exportOpen.value = true;
  },
});

const visibleItems = computed(() => {
  const useDerived = props.import || props.export;
  const source = props.items ?? (useDerived ? configItems.value : []);
  return (source ?? []).filter(item => !item.hidden);
});

function onCommand(key: string) {
  const item = visibleItems.value.find(entry => entry.key === key);
  if (item && !item.disabled) {
    item.onClick();
  }
}

function onImported() {
  void props.listRef?.refresh?.();
  emit('imported');
}
</script>

<style scoped>
.o-page-io-menu__trigger {
  padding: 4px 8px;
}
</style>
