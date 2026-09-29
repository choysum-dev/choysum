<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <DropdownMenu v-if="visibleItems.length">
    <DropdownMenuTrigger
      class="o-page-io-menu__trigger"
      :aria-label="menuAriaLabel"
      data-testid="page-io-menu-trigger"
    >
      <Settings class="size-[18px]" />
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      <DropdownMenuItem
        v-for="item in visibleItems"
        :key="item.key"
        :disabled="item.disabled"
        :data-testid="`page-io-menu-${item.key}`"
        @select="onCommand(item.key)"
      >
        {{ item.label }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <RecordImportShell
    v-if="importEnabled"
    v-model:open="importOpen"
    :model="resolvedModel"
    :config="importShellConfig"
    :company-id="actionCompanyId"
    @imported="onImported"
  />
  <RecordExportShell
    v-if="exportEnabled"
    v-model:open="exportOpen"
    :model="resolvedModel"
    :store="resolvedStore!"
    :list-ref="resolvedListRef"
    :company-id="actionCompanyId"
  />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { Settings } from 'lucide-vue-next';
import DropdownMenu from '../vendor/ui/dropdown-menu/DropdownMenu.vue';
import DropdownMenuContent from '../vendor/ui/dropdown-menu/DropdownMenuContent.vue';
import DropdownMenuItem from '../vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import DropdownMenuTrigger from '../vendor/ui/dropdown-menu/DropdownMenuTrigger.vue';
import { createTranslate } from '@/web/web/i18n';
import { useRecordIoMenu } from '@/web/web/composables/useRecordIoMenu';
import type { PageIoMenuItem, RecordIoConfig } from '@/web/web/composables/recordIoTypes';
import type { RecordExportListRef } from '@/web/web/composables/useRecordExportScope';
import { useResolvedOptionalPageStore, usePageContext } from '@/web/web/composables/usePageContext';
import { RecordImportShell } from '@/web/web/import';
import { RecordExportShell } from '@/web/web/export';

defineOptions({ name: 'ChoyPageIoMenu' });

export type PageIoMenuListRef = RecordExportListRef & {
  refresh?: () => Promise<void> | void;
};

type PageIoStore = {
  storeId?: string;
  fullModelName?: string;
  state?: { result?: { total?: number } };
};

const props = defineProps<{
  /** Low-level menu entries. When omitted, items are derived from action flags. */
  items?: PageIoMenuItem[];
  /** Enable Import panel and menu item. */
  actionImport?: boolean;
  /** Enable Export panel and menu item. */
  actionExport?: boolean;
  /** Optional CSV upload hint for ImportPanel. */
  actionImportUploadHint?: string;
  /** Optional default column mapping for ImportPanel. */
  actionImportColumnMapping?: Record<string, string>;
  /** List/kanban store; falls back to Page provided store when omitted. */
  store?: PageIoStore;
  /** Explicit list/kanban target; falls back to the page-registered main view. */
  actionListRef?: PageIoMenuListRef | null;
  /** Optional company override for import/export panels. */
  actionCompanyId?: string;
}>();

const emit = defineEmits<{
  (e: 'imported'): void;
}>();

const { _t } = createTranslate('web', { scope: 'web/components/layout/ChoyPageIoMenu' });
const menuAriaLabel = _t('Import and export');

const importOpen = ref(false);
const exportOpen = ref(false);

const pageCtx = usePageContext();
const resolvedStore = useResolvedOptionalPageStore<PageIoStore>(() => props.store);
const resolvedModel = computed(() => {
  const store = resolvedStore.value as
    | { fullModelName?: string; application?: string; modelName?: string }
    | null
    | undefined;
  const full = String(store?.fullModelName ?? '').trim();
  if (full) return full;
  const app = String(store?.application ?? '').trim();
  const name = String(store?.modelName ?? '').trim();
  if (app && name) return `${app}.${name}`;
  return '';
});
const resolvedListRef = computed(
  () => props.actionListRef ?? pageCtx?.actionTarget.value ?? null,
);

const requestedConfig = computed<RecordIoConfig>(() => {
  const config: RecordIoConfig = {};
  if (props.actionImport) {
    config.import = {
      enabled: true,
      uploadHint: props.actionImportUploadHint,
      columnMapping: props.actionImportColumnMapping,
    };
  }
  if (props.actionExport) {
    config.export = { enabled: true };
  }
  return config;
});

const importEnabled = computed(() => !!props.actionImport && !!resolvedModel.value);
const exportEnabled = computed(
  () => !!props.actionExport && !!resolvedStore.value && !!resolvedModel.value,
);

// Title-row entries follow the page action flags; panels still require a model.
const menuConfig = computed<RecordIoConfig>(() => requestedConfig.value);

const importShellConfig = computed<RecordIoConfig>(() => ({
  import: {
    enabled: true,
    uploadHint: props.actionImportUploadHint,
    columnMapping: props.actionImportColumnMapping,
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
  const useDerived = props.actionImport || props.actionExport;
  const source = props.items ?? (useDerived ? configItems.value : []);
  return source.filter(item => !item.hidden);
});

function onCommand(key: string) {
  const item = visibleItems.value.find(entry => entry.key === key);
  if (item && !item.disabled) {
    item.onClick();
  }
}

function onImported() {
  void resolvedListRef.value?.refresh?.();
  emit('imported');
}

defineExpose({ onCommand });
</script>

<style scoped>
.o-page-io-menu__trigger {
  padding: 4px 8px;
  border: 0;
  background: transparent;
  cursor: pointer;
}
</style>
