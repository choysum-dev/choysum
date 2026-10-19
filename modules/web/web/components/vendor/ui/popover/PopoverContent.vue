<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { PopoverContent, PopoverPortal } from 'reka-ui'
import { cn, type ClassValue } from '../../../../lib/utils'

defineOptions({ inheritAttrs: false })

withDefaults(
  defineProps<{
    class?: ClassValue
    align?: 'start' | 'center' | 'end'
    side?: 'top' | 'right' | 'bottom' | 'left'
    sideOffset?: number
    alignOffset?: number
  }>(),
  {
    align: 'center',
    side: 'bottom',
    sideOffset: 4,
    alignOffset: 0,
  },
)
</script>

<template>
  <PopoverPortal>
    <PopoverContent
      data-slot="popover-content"
      v-bind="$attrs"
      :align="align"
      :side="side"
      :side-offset="sideOffset"
      :align-offset="alignOffset"
      :class="
        cn(
          'bg-background text-foreground data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 z-50 w-72 origin-(--reka-popover-content-transform-origin) rounded-md border border-border p-4 shadow-md outline-hidden',
          $props.class,
        )
      "
    >
      <slot />
    </PopoverContent>
  </PopoverPortal>
</template>
