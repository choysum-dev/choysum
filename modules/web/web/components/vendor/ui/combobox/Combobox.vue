<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ComboboxRoot } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

withDefaults(
  defineProps<{
    class?: ClassValue;
    disabled?: boolean;
    /** Disable built-in item filtering (caller filters the list). */
    ignoreFilter?: boolean;
    /** Open the list when the input receives focus. */
    openOnFocus?: boolean;
    /** Open the list when the input is clicked. */
    openOnClick?: boolean;
  }>(),
  {
    disabled: false,
    ignoreFilter: false,
    openOnFocus: false,
    openOnClick: false,
  },
);

const modelValue = defineModel<string>();
const open = defineModel<boolean>('open', { default: false });
</script>

<template>
  <ComboboxRoot
    v-model="modelValue"
    v-model:open="open"
    data-slot="combobox"
    :disabled="disabled"
    :ignore-filter="ignoreFilter"
    :open-on-focus="openOnFocus"
    :open-on-click="openOnClick"
    :class="cn($props.class)"
  >
    <slot />
  </ComboboxRoot>
</template>
