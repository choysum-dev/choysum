<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { CollapsibleTrigger } from "reka-ui"

defineProps<{
  asChild?: boolean
}>()
</script>

<template>
  <CollapsibleTrigger
    data-slot="collapsible-trigger"
    :as-child="asChild"
  >
    <slot />
  </CollapsibleTrigger>
</template>
