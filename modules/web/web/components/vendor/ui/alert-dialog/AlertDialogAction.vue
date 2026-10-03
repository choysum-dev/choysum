<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { AlertDialogAction } from "reka-ui"
import { cn } from "../../../../lib/utils"
import { buttonVariants } from "../button"

defineProps<{ class?: HTMLAttributes["class"] }>()
</script>

<template>
  <AlertDialogAction :class="cn(buttonVariants(), $props.class)">
    <slot />
  </AlertDialogAction>
</template>
