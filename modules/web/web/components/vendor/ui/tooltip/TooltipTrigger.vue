<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { TooltipTrigger } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue; asChild?: boolean }>();
</script>

<template>
  <TooltipTrigger data-slot="tooltip-trigger" :as-child="asChild" :class="cn($props.class)">
    <slot />
  </TooltipTrigger>
</template>
