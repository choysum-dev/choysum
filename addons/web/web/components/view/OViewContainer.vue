<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-view-container">
    <div v-if="$slots.header || showHeader" class="o-view-container__header">
      <slot name="header" />
    </div>
    <div class="o-view-container__body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="o-view-container__footer">
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
.o-view-container {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-width: 0;
}
.o-view-container__header {
  flex: 0 0 auto;
}
.o-view-container__body {
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
}
.o-view-container__footer {
  flex: 0 0 auto;
}

/* Backward compatibility: keep legacy .o-view-shell selectors temporarily */
.o-view-shell {
  display: contents; /* neutralize legacy wrapper if still referenced */
}
.o-view-shell__header,
.o-view-shell__body,
.o-view-shell__footer {
  display: contents;
}
</style>
