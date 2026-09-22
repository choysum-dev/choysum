<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DropdownMenuTrigger } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';

defineProps<{ class?: ClassValue; asChild?: boolean }>();
</script>

<template>
  <DropdownMenuTrigger data-slot="dropdown-menu-trigger" :as-child="asChild" :class="cn($props.class)">
    <slot />
  </DropdownMenuTrigger>
</template>
