<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DialogDescription } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <DialogDescription data-slot="dialog-description" :class="cn('text-sm text-foreground/70', $props.class)">
    <slot />
  </DialogDescription>
</template>
