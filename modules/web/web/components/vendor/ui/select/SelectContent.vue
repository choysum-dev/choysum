<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import {
  SelectContent,
  SelectPortal,
  SelectViewport,
} from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <SelectPortal>
    <SelectContent
      data-slot="select-content"
      position="popper"
      :class="
        cn(
          'relative z-50 max-h-60 min-w-[8rem] overflow-hidden rounded-md border border-border bg-background text-foreground shadow-md',
          $props.class,
        )
      "
    >
      <SelectViewport class="p-1">
        <slot />
      </SelectViewport>
    </SelectContent>
  </SelectPortal>
</template>
