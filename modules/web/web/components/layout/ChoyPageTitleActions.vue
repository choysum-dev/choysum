<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    v-if="hasIoMenu || $slots.overflow || $slots.default"
    class="choy-page__title-actions flex shrink-0 items-center gap-2"
  >
    <ChoyPageIoMenu
      v-if="hasIoMenu || $slots.overflow"
      :action-import="hasIoMenu ? actionImport : false"
      :action-export="hasIoMenu ? actionExport : false"
      :action-import-upload-hint="hasIoMenu ? actionImportUploadHint : undefined"
      :action-import-column-mapping="hasIoMenu ? actionImportColumnMapping : undefined"
      :action-list-ref="hasIoMenu ? actionListRef : undefined"
      :action-company-id="hasIoMenu ? actionCompanyId : undefined"
      :store="store"
    >
      <slot name="overflow" />
    </ChoyPageIoMenu>
    <slot />
  </div>
</template>

<script setup lang="ts">
import ChoyPageIoMenu from './ChoyPageIoMenu.vue';
import type { PageIoMenuListRef } from './ChoyPageIoMenu.vue';

/**
 * Shared title-actions row for ChoyPage and list chrome.
 * `#overflow` items land inside the ⋮ menu; default slot stays beside the trigger.
 */
defineProps<{
  hasIoMenu: boolean;
  actionImport: boolean;
  actionExport: boolean;
  actionImportUploadHint?: string;
  actionImportColumnMapping?: Record<string, string>;
  actionListRef?: PageIoMenuListRef | null;
  actionCompanyId?: string;
  store?: { storeId?: string; fullModelName?: string; state?: { result?: { total?: number } } };
}>();
</script>
