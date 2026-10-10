<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next'
import { NavigationMenuTrigger } from 'reka-ui'
import { cn, type ClassValue } from '../../../../lib/utils'
import { navigationMenuTriggerStyle } from './navigationMenuTriggerStyle'

defineProps<{ class?: ClassValue; disabled?: boolean }>()
</script>

<template>
  <NavigationMenuTrigger
    data-slot="navigation-menu-trigger"
    :disabled="disabled"
    :class="cn(navigationMenuTriggerStyle(), 'group', $props.class)"
  >
    <slot />
    <ChevronDown
      class="relative top-px ml-1 size-3 transition duration-300 group-data-[state=open]:rotate-180"
      aria-hidden="true"
    />
  </NavigationMenuTrigger>
</template>
