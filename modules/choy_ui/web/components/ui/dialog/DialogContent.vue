<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { X } from 'lucide-vue-next';
import { DialogClose, DialogContent, DialogPortal } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';
import DialogOverlay from './DialogOverlay.vue';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <DialogPortal>
    <DialogOverlay />
    <DialogContent
      data-slot="dialog-content"
      :class="
        cn(
          'fixed left-1/2 top-1/2 z-50 grid w-full max-w-lg -translate-x-1/2 -translate-y-1/2 gap-4',
          'border border-border bg-background p-6 text-foreground shadow-lg rounded-lg',
          'focus:outline-none',
          $props.class,
        )
      "
    >
      <slot />
      <DialogClose
        class="absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring"
        aria-label="Close"
      >
        <X class="size-4" />
      </DialogClose>
    </DialogContent>
  </DialogPortal>
</template>
