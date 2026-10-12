<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DropdownMenuContent, DropdownMenuPortal } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineOptions({ inheritAttrs: false });

withDefaults(
  defineProps<{
    class?: ClassValue;
    align?: 'start' | 'center' | 'end';
    side?: 'top' | 'right' | 'bottom' | 'left';
    sideOffset?: number;
    alignOffset?: number;
  }>(),
  {
    align: 'start',
    side: 'bottom',
    sideOffset: 4,
    alignOffset: 0,
  },
);
</script>

<template>
  <DropdownMenuPortal>
    <DropdownMenuContent
      data-slot="dropdown-menu-content"
      v-bind="$attrs"
      :align="align"
      :side="side"
      :side-offset="sideOffset"
      :align-offset="alignOffset"
      :class="
        cn(
          'z-50 min-w-[8rem] overflow-hidden rounded-md border border-border bg-background p-1 text-foreground shadow-md',
          $props.class,
        )
      "
    >
      <slot />
    </DropdownMenuContent>
  </DropdownMenuPortal>
</template>
