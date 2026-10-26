<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DropdownMenuSeparator } from 'reka-ui'
import { cn, type ClassValue } from '../../../../lib/utils'

defineProps<{
  class?: ClassValue
}>()
</script>

<template>
  <DropdownMenuSeparator
    data-slot="dropdown-menu-separator"
    :class="cn('-mx-1 my-1 h-px bg-border', $props.class)"
  />
</template>
