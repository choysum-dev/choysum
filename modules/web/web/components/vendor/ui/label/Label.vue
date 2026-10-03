<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { Label } from "reka-ui"
import { cn } from "../../../../lib/utils"

defineProps<{
  class?: HTMLAttributes["class"]
  for?: string
}>()
</script>

<template>
  <Label
    data-slot="label"
    :for="$props.for"
    :class="
      cn(
        'flex items-center gap-2 text-sm leading-none font-medium select-none group-data-[disabled=true]:pointer-events-none group-data-[disabled=true]:opacity-50 peer-disabled:cursor-not-allowed peer-disabled:opacity-50',
        $props.class,
      )
    "
  >
    <slot />
  </Label>
</template>
