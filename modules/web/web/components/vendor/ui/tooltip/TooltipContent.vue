<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { TooltipContent, TooltipPortal } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineOptions({ inheritAttrs: false });

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <TooltipPortal>
    <TooltipContent
      data-slot="tooltip-content"
      v-bind="$attrs"
      :side-offset="4"
      :class="
        cn(
          'bg-background text-foreground animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 z-50 origin-(--reka-tooltip-content-transform-origin) overflow-hidden rounded-md border border-border px-3 py-1.5 text-xs shadow-md',
          $props.class,
        )
      "
    >
      <slot />
    </TooltipContent>
  </TooltipPortal>
</template>
