<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { NavigationMenuLink } from 'reka-ui'
import { cn, type ClassValue } from '../../../../lib/utils'

defineProps<{
  class?: ClassValue
  href?: string
  active?: boolean
  asChild?: boolean
}>()
</script>

<template>
  <NavigationMenuLink
    data-slot="navigation-menu-link"
    :href="href"
    :active="active"
    :as-child="asChild"
    :class="
      cn(
        'flex flex-col gap-1 rounded-sm p-2 text-sm transition-[color,box-shadow] outline-none',
        'hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground',
        'focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-1',
        'data-active:bg-accent/50 data-active:text-accent-foreground data-active:hover:bg-accent data-active:focus:bg-accent',
        `[&_svg:not([class*='text-'])]:text-muted-foreground [&_svg:not([class*='size-'])]:size-4`,
        $props.class,
      )
    "
  >
    <slot />
  </NavigationMenuLink>
</template>
