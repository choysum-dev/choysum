<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { SelectRoot } from 'reka-ui';

const modelValue = defineModel<string | null>();
</script>

<template>
  <SelectRoot v-model="modelValue" data-slot="select">
    <slot />
  </SelectRoot>
</template>
