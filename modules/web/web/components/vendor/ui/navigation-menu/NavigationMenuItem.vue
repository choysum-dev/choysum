<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { NavigationMenuItem } from 'reka-ui'
import { cn, type ClassValue } from '../../../../lib/utils'

defineProps<{ class?: ClassValue; value?: string }>()
</script>

<template>
  <NavigationMenuItem
    data-slot="navigation-menu-item"
    :value="value"
    :class="cn('relative', $props.class)"
  >
    <slot />
  </NavigationMenuItem>
</template>
