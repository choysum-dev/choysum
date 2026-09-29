<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    v-if="hasIoMenu || $slots.default"
    class="choy-page__title-actions flex shrink-0 items-center gap-2"
  >
    <ChoyPageIoMenu
      v-if="hasIoMenu"
      :action-import="actionImport"
      :action-export="actionExport"
      :action-import-upload-hint="actionImportUploadHint"
      :action-import-column-mapping="actionImportColumnMapping"
      :action-list-ref="actionListRef"
      :action-company-id="actionCompanyId"
      :store="store"
    />
    <slot />
  </div>
</template>

<script setup lang="ts">
import ChoyPageIoMenu from './ChoyPageIoMenu.vue';
import type { PageIoMenuListRef } from './ChoyPageIoMenu.vue';

/**
 * Shared title-actions row for ChoyPage (header and fallback title layouts).
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
