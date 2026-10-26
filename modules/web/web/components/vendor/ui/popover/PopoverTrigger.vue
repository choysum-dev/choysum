<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { PopoverTrigger } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue; asChild?: boolean }>();
</script>

<template>
  <PopoverTrigger data-slot="popover-trigger" :as-child="asChild" :class="cn($props.class)">
    <slot />
  </PopoverTrigger>
</template>
