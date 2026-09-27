<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { TabsList } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <TabsList
    data-slot="tabs-list"
    :class="cn('inline-flex h-9 items-center justify-center rounded-md bg-muted p-1 text-foreground/70', $props.class)"
  >
    <slot />
  </TabsList>
</template>
