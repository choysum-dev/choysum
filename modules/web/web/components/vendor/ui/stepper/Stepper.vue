<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script lang="ts" setup>
import type { HTMLAttributes } from "vue"
import { reactiveOmit } from "@vueuse/core"
import { StepperRoot, useForwardPropsEmits } from "reka-ui"
import { cn } from "../../../../lib/utils"

const props = withDefaults(defineProps<{
  defaultValue?: number
  orientation?: "vertical" | "horizontal"
  dir?: "ltr" | "rtl"
  modelValue?: number
  linear?: boolean
  asChild?: boolean
  as?: string
  class?: HTMLAttributes["class"]
}>(), { linear: true })
const emits = defineEmits<{ "update:modelValue": [payload: number | undefined] }>()

const delegatedProps = reactiveOmit(props, "class")

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <StepperRoot
    v-slot="slotProps"
    :class="cn(
      'flex gap-2',
      props.class,
    )"
    v-bind="forwarded"
  >
    <slot v-bind="slotProps" />
  </StepperRoot>
</template>
