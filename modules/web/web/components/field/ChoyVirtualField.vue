<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Virtual field registers value only; no DOM. -->
</template>

<script setup lang="ts">
/**
 * Virtual field: holds a model value for forms without rendering DOM.
 * Hosts may still bind v-model for computed/derived values.
 */
defineOptions({ inheritAttrs: false });
defineModel<unknown>({ default: null });
</script>
