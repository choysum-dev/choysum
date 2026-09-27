<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OVColumn v-bind="($attrs as any)">
    <template v-if="$slots.default" #default="slotProps">
      <slot v-bind="(slotProps as any) || {}" />
    </template>
  </OVColumn>
</template>

<script setup lang="ts">
/**
 * Public list column registrar for store-bound ChoyListView.
 * Hosts the OVColumn registry until the DataTable column API absorbs slot fields.
 * Default slot is forwarded only when the consumer provides one so OVColumn can
 * keep its built-in cell renderer when no custom cell is declared.
 */
import OVColumn from './OVColumn.vue';

defineOptions({ name: 'ChoyVColumn', inheritAttrs: false });
</script>
