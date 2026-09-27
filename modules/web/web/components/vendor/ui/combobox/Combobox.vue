<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ComboboxRoot } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue }>();

const modelValue = defineModel<string>();
const open = defineModel<boolean>('open', { default: false });
</script>

<template>
  <ComboboxRoot
    v-model="modelValue"
    v-model:open="open"
    data-slot="combobox"
    :class="cn($props.class)"
  >
    <slot />
  </ComboboxRoot>
</template>
