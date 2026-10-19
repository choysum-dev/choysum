<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script lang='ts' setup>
import type { HTMLAttributes } from "vue"
import { Primitive } from "reka-ui"
import { cn } from "../../../../lib/utils"

interface Props {
  as?: any
  asChild?: boolean
  class?: HTMLAttributes["class"]
}
const props = withDefaults(defineProps<Props>(), {
  as: "div",
})
</script>

<template>
  <Primitive
    data-slot="message-group"
    :as="as"
    :as-child="asChild"
    :class="cn(
      'gap-2 flex min-w-0 flex-col',
      props.class,
    )"
  >
    <slot />
  </Primitive>
</template>
