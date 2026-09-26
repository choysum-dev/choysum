<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    v-if="props.actionImport || props.actionExport"
    data-anchor="choy.page-io-menu"
    class="choy-page-io-menu flex items-center gap-2"
  >
    <ChoyButton
      v-if="props.actionImport"
      variant="outline"
      size="sm"
      data-testid="page-io-menu-import"
      @click="emit('import')"
    >
      Import
    </ChoyButton>
    <ChoyButton
      v-if="props.actionExport"
      variant="outline"
      size="sm"
      data-testid="page-io-menu-export"
      @click="emit('export')"
    >
      Export
    </ChoyButton>
  </div>
</template>

<script setup lang="ts">
import ChoyButton from './ChoyButton.vue';

/**
 * Simplified page IO menu. PR3 exposes Import/Export triggers only; real
 * upload/export pipelines land with product cutover / later PRs.
 */
const props = withDefaults(
  defineProps<{
    actionImport?: boolean;
    actionExport?: boolean;
  }>(),
  {
    actionImport: false,
    actionExport: false,
  },
);

const emit = defineEmits<{
  import: [];
  export: [];
}>();
</script>
