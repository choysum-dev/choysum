<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { AlertDialogDescription } from "reka-ui"
import { cn } from "../../../../lib/utils"

defineProps<{ class?: HTMLAttributes["class"] }>()
</script>

<template>
  <AlertDialogDescription
    data-slot="alert-dialog-description"
    :class="cn('text-muted-foreground text-sm', $props.class)"
  >
    <slot />
  </AlertDialogDescription>
</template>
