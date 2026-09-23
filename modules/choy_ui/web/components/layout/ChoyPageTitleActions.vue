<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import ChoyPageIoMenu from './ChoyPageIoMenu.vue';

/**
 * Shared title-actions row for ChoyPage (header and fallback title layouts).
 */
defineProps<{
  hasIoMenu: boolean;
  actionImport: boolean;
  actionExport: boolean;
}>();

const emit = defineEmits<{
  import: [];
  export: [];
}>();
</script>

<template>
  <div
    v-if="hasIoMenu || $slots.default"
    class="choy-page__title-actions flex shrink-0 items-center gap-2"
  >
    <ChoyPageIoMenu
      v-if="hasIoMenu"
      :action-import="actionImport"
      :action-export="actionExport"
      @import="emit('import')"
      @export="emit('export')"
    />
    <slot />
  </div>
</template>
