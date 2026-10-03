<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script lang="ts" setup>
import type { HTMLAttributes } from "vue"
import { Primitive } from "reka-ui"
import { cn } from "../../../../lib/utils"

withDefaults(defineProps<{
  class?: HTMLAttributes["class"]
  as?: string | object
  asChild?: boolean
}>(), {
  as: "a",
})
</script>

<template>
  <Primitive
    data-slot="breadcrumb-link"
    :as="as"
    :as-child="asChild"
    :class="cn('hover:text-foreground transition-colors', $props.class)"
  >
    <slot />
  </Primitive>
</template>
