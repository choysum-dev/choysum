<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-view-container">
    <div v-if="$slots.header || showHeader" class="choy-view-container__header">
      <slot name="header" />
    </div>
    <div class="choy-view-container__body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="choy-view-container__footer">
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

<style scoped>
/* Primary new class names */
.choy-view-container {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-width: 0;
}
.choy-view-container__header {
  flex: 0 0 auto;
}
.choy-view-container__body {
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
}
.choy-view-container__footer {
  flex: 0 0 auto;
}

/* Alternate selectors kept for hosts that still target the old shell class names. */
.choy-view-shell {
  display: contents; /* neutralize legacy wrapper if still referenced */
}
.choy-view-shell__header,
.choy-view-shell__body,
.choy-view-shell__footer {
  display: contents;
}
</style>
