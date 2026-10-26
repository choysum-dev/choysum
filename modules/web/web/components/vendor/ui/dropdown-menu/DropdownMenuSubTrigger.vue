<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ChevronRight } from 'lucide-vue-next';
import { DropdownMenuSubTrigger } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{
  class?: ClassValue;
  disabled?: boolean;
  inset?: boolean;
}>();
</script>

<template>
  <DropdownMenuSubTrigger
    data-slot="dropdown-menu-sub-trigger"
    :disabled="disabled"
    :class="
      cn(
        'flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none',
        'focus:bg-muted data-[state=open]:bg-muted data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
        inset && 'ps-8',
        $props.class,
      )
    "
  >
    <slot />
    <ChevronRight class="ms-auto size-4 shrink-0 opacity-50" aria-hidden="true" />
  </DropdownMenuSubTrigger>
</template>
