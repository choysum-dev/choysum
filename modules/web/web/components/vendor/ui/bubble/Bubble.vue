<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script lang='ts' setup>
import type { HTMLAttributes } from "vue"
import type { BubbleVariants } from "."
import { Primitive } from "reka-ui"
import { cn } from "../../../../lib/utils"
import { bubbleVariants } from "."

interface Props {
  as?: any
  asChild?: boolean
  variant?: BubbleVariants["variant"]
  align?: "start" | "end"
  class?: HTMLAttributes["class"]
}

const props = withDefaults(defineProps<Props>(), {
  variant: "default",
  align: "start",
  as: "div",
})
</script>

<template>
  <Primitive
    data-slot="bubble"
    :data-variant="variant"
    :data-align="align"
    :as="as"
    :as-child="asChild"
    :class="cn(bubbleVariants({ variant }), props.class)"
  >
    <slot />
  </Primitive>
</template>
