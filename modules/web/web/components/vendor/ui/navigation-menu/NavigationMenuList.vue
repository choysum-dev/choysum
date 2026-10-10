<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { NavigationMenuList } from 'reka-ui'
import { cn, type ClassValue } from '../../../../lib/utils'

defineProps<{ class?: ClassValue }>()
</script>

<template>
  <NavigationMenuList
    data-slot="navigation-menu-list"
    :class="
      cn(
        'group flex flex-1 list-none items-center justify-center gap-1',
        $props.class,
      )
    "
  >
    <slot />
  </NavigationMenuList>
</template>
