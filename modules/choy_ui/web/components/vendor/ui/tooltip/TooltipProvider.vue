<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { TooltipProvider } from 'reka-ui';

withDefaults(
  defineProps<{
    delayDuration?: number;
  }>(),
  { delayDuration: 200 },
);
</script>

<template>
  <TooltipProvider :delay-duration="delayDuration">
    <slot />
  </TooltipProvider>
</template>
