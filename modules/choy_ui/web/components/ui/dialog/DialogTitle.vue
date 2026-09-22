<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DialogTitle } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <DialogTitle data-slot="dialog-title" :class="cn('text-lg font-semibold leading-none', $props.class)">
    <slot />
  </DialogTitle>
</template>
