<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script lang='ts' setup>
import type { HTMLAttributes } from "vue"
import type { BubbleReactionsVariants } from "."
import { Primitive } from "reka-ui"
import { cn } from "../../../../lib/utils"
import { bubbleReactionsVariants } from "."

interface Props {
  as?: any
  asChild?: boolean
  side?: BubbleReactionsVariants["side"]
  align?: BubbleReactionsVariants["align"]
  class?: HTMLAttributes["class"]
}

const props = withDefaults(defineProps<Props>(), {
  side: "bottom",
  align: "end",
  as: "div",
})
</script>

<template>
  <Primitive
    data-slot="bubble-reactions"
    :data-side="side"
    :data-align="align"
    :as="as"
    :as-child="asChild"
    :class="cn(bubbleReactionsVariants({ side, align }), props.class)"
  >
    <slot />
  </Primitive>
</template>
