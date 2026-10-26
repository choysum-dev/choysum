<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ComboboxViewport } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <ComboboxViewport
    data-slot="combobox-viewport"
    :class="cn('max-h-60 overflow-y-auto p-1', $props.class)"
  >
    <slot />
  </ComboboxViewport>
</template>
