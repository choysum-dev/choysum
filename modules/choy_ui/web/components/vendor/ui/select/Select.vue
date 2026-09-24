<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import { SelectRoot } from 'reka-ui';

const modelValue = defineModel<string | null>();

// Reka treats ''/undefined as "no selection" and rejects '' as an item value, so ''
// is the safe sentinel for the root; a `null` model would render a blank trigger.
const rootValue = computed({
  get: (): string => modelValue.value ?? '',
  set: (value: string | null | undefined) => {
    modelValue.value = value == null || value === '' ? null : String(value);
  },
});
</script>

<template>
  <SelectRoot v-model="rootValue" data-slot="select">
    <slot />
  </SelectRoot>
</template>
