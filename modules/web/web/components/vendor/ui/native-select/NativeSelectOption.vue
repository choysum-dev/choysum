<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { cn } from "../../../../lib/utils"

const props = defineProps<{ class?: HTMLAttributes["class"] }>()
</script>

<template>
  <option data-slot="native-select-option" :class="cn('bg-popover text-popover-foreground', props.class)">
    <slot />
  </option>
</template>
