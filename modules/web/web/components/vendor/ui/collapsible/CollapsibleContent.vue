<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { CollapsibleContent } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { cn } from "../../../../lib/utils"

const props = defineProps<{
  class?: HTMLAttributes["class"]
  forceMount?: boolean
}>()
</script>

<template>
  <CollapsibleContent
    data-slot="collapsible-content"
    :force-mount="forceMount"
    :class="cn(props.class)"
  >
    <slot />
  </CollapsibleContent>
</template>
