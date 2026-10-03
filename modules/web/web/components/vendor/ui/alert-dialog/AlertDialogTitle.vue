<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { AlertDialogTitle } from "reka-ui"
import { cn } from "../../../../lib/utils"

defineProps<{ class?: HTMLAttributes["class"] }>()
</script>

<template>
  <AlertDialogTitle
    data-slot="alert-dialog-title"
    :class="cn('text-lg font-semibold', $props.class)"
  >
    <slot />
  </AlertDialogTitle>
</template>
