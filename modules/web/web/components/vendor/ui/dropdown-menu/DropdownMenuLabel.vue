<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DropdownMenuLabel } from 'reka-ui'
import { cn, type ClassValue } from '../../../../lib/utils'

defineProps<{
  class?: ClassValue
  inset?: boolean
}>()
</script>

<template>
  <DropdownMenuLabel
    data-slot="dropdown-menu-label"
    :class="
      cn(
        'px-2 py-1.5 text-xs font-medium text-muted-foreground',
        inset && 'ps-8',
        $props.class,
      )
    "
  >
    <slot />
  </DropdownMenuLabel>
</template>
