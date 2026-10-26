<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { NavigationMenuRoot } from 'reka-ui'
import { cn, type ClassValue } from '../../../../lib/utils'
import NavigationMenuViewport from './NavigationMenuViewport.vue'

withDefaults(
  defineProps<{
    class?: ClassValue
    /** When true, render the floating viewport under the root. */
    viewport?: boolean
  }>(),
  {
    viewport: true,
  },
)
</script>

<template>
  <NavigationMenuRoot
    data-slot="navigation-menu"
    :data-viewport="viewport"
    :class="
      cn(
        'group/navigation-menu relative flex max-w-max flex-1 items-center justify-center',
        $props.class,
      )
    "
  >
    <slot />
    <NavigationMenuViewport v-if="viewport" />
  </NavigationMenuRoot>
</template>
