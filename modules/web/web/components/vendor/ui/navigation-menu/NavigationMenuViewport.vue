<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { NavigationMenuViewport } from 'reka-ui'
import { cn, type ClassValue } from '../../../../lib/utils'

defineProps<{ class?: ClassValue }>()
</script>

<template>
  <div class="absolute top-full left-0 isolate z-50 flex justify-center">
    <NavigationMenuViewport
      data-slot="navigation-menu-viewport"
      :class="
        cn(
          'origin-top-center bg-popover text-popover-foreground data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-90 relative mt-1.5 h-(--reka-navigation-menu-viewport-height) w-full overflow-hidden rounded-md border shadow md:w-(--reka-navigation-menu-viewport-width) left-(--reka-navigation-menu-viewport-left)',
          $props.class,
        )
      "
    />
  </div>
</template>
