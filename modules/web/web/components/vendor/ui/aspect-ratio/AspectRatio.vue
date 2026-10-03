<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { AspectRatio } from "reka-ui"

// Local props — avoid `defineProps<AspectRatioProps>()` (reka type import breaks QuickJS FE SFC).
const props = defineProps<{
  ratio?: number
}>()
</script>

<template>
  <AspectRatio
    v-slot="slotProps"
    data-slot="aspect-ratio"
    v-bind="props"
  >
    <slot v-bind="slotProps" />
  </AspectRatio>
</template>
