<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DropdownMenuPortal, DropdownMenuSubContent } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineOptions({ inheritAttrs: false });

defineProps<{
  class?: ClassValue;
}>();
</script>

<template>
  <DropdownMenuPortal>
    <DropdownMenuSubContent
      data-slot="dropdown-menu-sub-content"
      v-bind="$attrs"
      :class="
        cn(
          'z-50 min-w-[8rem] overflow-hidden rounded-md border border-border bg-background p-1 text-foreground shadow-md',
          $props.class,
        )
      "
    >
      <slot />
    </DropdownMenuSubContent>
  </DropdownMenuPortal>
</template>
