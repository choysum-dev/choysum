<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ComboboxAnchor } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <ComboboxAnchor
    data-slot="combobox-anchor"
    :class="cn('relative flex w-full items-center', $props.class)"
  >
    <slot />
  </ComboboxAnchor>
</template>
