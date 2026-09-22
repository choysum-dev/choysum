<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { PopoverContent, PopoverPortal } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <PopoverPortal>
    <PopoverContent
      data-slot="popover-content"
      :side-offset="4"
      :class="
        cn(
          'z-50 w-72 rounded-md border border-border bg-background p-4 text-foreground shadow-md outline-none',
          $props.class,
        )
      "
    >
      <slot />
    </PopoverContent>
  </PopoverPortal>
</template>
