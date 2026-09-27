<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OVColumn v-bind="($attrs as any)">
    <slot />
  </OVColumn>
</template>

<script setup lang="ts">
/**
 * Public list column registrar for store-bound ChoyListView.
 * Hosts the OVColumn registry until the DataTable column API absorbs slot fields.
 */
import OVColumn from './OVColumn.vue';

defineOptions({ name: 'ChoyVColumn', inheritAttrs: false });
</script>
