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
          'z-50 w-72 rounded-md border border-border bg-background p-4 text-foreground shadow-md outline-none',
          $props.class,
        )
      "
    >
      <slot />
    </PopoverContent>
  </PopoverPortal>
</template>
