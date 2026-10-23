<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { NumberFieldInput } from "reka-ui"
import { cn, controlFocusRing } from "../../../../lib/utils"

const props = defineProps<{
  class?: HTMLAttributes["class"]
}>()
</script>

<template>
  <NumberFieldInput
    data-slot="input"
    :class="cn('flex h-control w-full rounded-md border border-input bg-transparent py-1 text-sm text-center shadow-xs placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50', controlFocusRing, props.class)"
  />
</template>
