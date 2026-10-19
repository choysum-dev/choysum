<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { AlertDialogAction } from "reka-ui"
import { cn } from "../../../../lib/utils"
import { buttonVariants, type ButtonVariants } from "../button"

withDefaults(
  defineProps<{
    class?: HTMLAttributes["class"]
    variant?: ButtonVariants["variant"]
    size?: ButtonVariants["size"]
  }>(),
  {
    variant: "default",
    size: "default",
  },
)
</script>

<template>
  <AlertDialogAction
    :class="cn(buttonVariants({ variant, size }), $props.class)"
  >
    <slot />
  </AlertDialogAction>
</template>
