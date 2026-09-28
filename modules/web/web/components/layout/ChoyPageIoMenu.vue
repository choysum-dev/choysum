<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <PageIoMenu
    v-if="props.actionImport || props.actionExport"
    data-anchor="choy.page-io-menu"
    :action-import="props.actionImport"
    :action-export="props.actionExport"
    :action-import-upload-hint="props.actionImportUploadHint"
    :action-import-column-mapping="props.actionImportColumnMapping"
    :action-list-ref="props.actionListRef"
    :action-company-id="props.actionCompanyId"
    :store="props.store"
  />
</template>

<script setup lang="ts">
import PageIoMenu, { type PageIoMenuListRef } from '../page/PageIoMenu.vue';

/**
 * Public kit alias for the title-row Import/Export menu.
 * Delegates to PageIoMenu so product pages get the real import/export shells.
 */
const props = withDefaults(
  defineProps<{
    actionImport?: boolean;
    actionExport?: boolean;
    actionImportUploadHint?: string;
    actionImportColumnMapping?: Record<string, string>;
    actionListRef?: PageIoMenuListRef | null;
    actionCompanyId?: string;
    store?: { storeId?: string; fullModelName?: string; state?: { result?: { total?: number } } };
  }>(),
  {
    actionImport: false,
    actionExport: false,
  },
);
</script>
