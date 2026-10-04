<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="flex h-full w-full min-w-0 flex-col">
    <div v-if="showHeader" class="shrink-0">
      <slot name="header" />
    </div>
    <div class="min-h-0 min-w-0 flex-1">
      <slot />
    </div>
    <div v-if="$slots.footer" class="shrink-0">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    showHeader?: boolean;
  }>(),
  { showHeader: true }
);
</script>

