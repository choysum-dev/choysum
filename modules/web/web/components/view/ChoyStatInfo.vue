<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <StatInfo v-bind="($attrs as any)" />
</template>

<script setup lang="ts">
/**
 * Store-aware stat button for form chrome (relation length / explicit value).
 * Hosts StatInfo until icon/chrome are fully on Choy tokens.
 */
import StatInfo from './StatInfo.vue';

defineOptions({ name: 'ChoyStatInfo', inheritAttrs: false });
</script>
