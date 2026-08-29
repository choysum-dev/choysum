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
    :config="config"
    :company-id="companyId"
    @imported="onImported"
  />
  <RecordExportShell
    v-if="exportEnabled"
    v-model:open="exportOpen"
    :model="config!.model"
    :store="store!"
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
import { RecordImportShell } from '@/web/web/import';
import { RecordExportShell } from '@/web/web/export';

defineOptions({ name: 'OPageIoMenu' });

export type OPageIoMenuListRef = RecordExportListRef & {
  refresh?: () => Promise<void> | void;
};

const props = defineProps<{
  /** Low-level menu entries. When omitted, items are derived from `config`. */
  items?: PageIoMenuItem[];
  /** Page IO capability declaration; enables Import/Export panels. */
  config?: RecordIoConfig;
  /** List/kanban store; required when export is enabled via `config`. */
  store?: { storeId?: string; state?: { result?: { total?: number } } };
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

const importEnabled = computed(() => !!props.config?.import?.enabled);
const exportEnabled = computed(() => !!props.config?.export?.enabled && !!props.store);

const menuConfig = computed<RecordIoConfig>(() => {
  const base = props.config ?? { model: '' };
  if (base.export?.enabled && !props.store) {
    return { ...base, export: { ...base.export, enabled: false } };
  }
  return base;
});

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
  const source = props.items ?? (props.config ? configItems.value : []);
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
