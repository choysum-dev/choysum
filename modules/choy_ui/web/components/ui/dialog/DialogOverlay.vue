<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DialogOverlay } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <DialogOverlay
    data-slot="dialog-overlay"
    :class="
      cn(
        'fixed inset-0 z-50 bg-foreground/40',
        $props.class,
      )
    "
  />
</template>
