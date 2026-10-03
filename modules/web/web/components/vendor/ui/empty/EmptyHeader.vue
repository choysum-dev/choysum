<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { cn } from "../../../../lib/utils"

const props = defineProps<{
  class?: HTMLAttributes["class"]
}>()
</script>

<template>
  <div
    data-slot="empty-header"
    :class="cn(
      'flex max-w-sm flex-col items-center gap-2 text-center',
      props.class,
    )"
  >
    <slot />
  </div>
</template>
