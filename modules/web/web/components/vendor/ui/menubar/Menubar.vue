<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { MenubarRootEmits, MenubarRootProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { reactiveOmit } from "@vueuse/core"
import {
  MenubarRoot,
  useForwardPropsEmits,
} from "reka-ui"
import { cn } from "../../../../lib/utils"

const props = defineProps<MenubarRootProps & { class?: HTMLAttributes["class"] }>()
const emits = defineEmits<MenubarRootEmits>()

const delegatedProps = reactiveOmit(props, "class")

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <MenubarRoot
    v-slot="slotProps"
    data-slot="menubar"
    v-bind="forwarded"
    :class="
      cn(
        'bg-background flex h-control items-center gap-1 rounded-md border p-1 shadow-sm',
        props.class,
      )
    "
  >
    <slot v-bind="slotProps" />
  </MenubarRoot>
</template>
