<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ComboboxEmpty } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <ComboboxEmpty
    data-slot="combobox-empty"
    :class="cn('px-2 py-4 text-center text-sm text-muted-foreground', $props.class)"
  >
    <slot />
  </ComboboxEmpty>
</template>
