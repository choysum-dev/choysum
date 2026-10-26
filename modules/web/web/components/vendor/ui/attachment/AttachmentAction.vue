<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import type { ButtonVariants } from "../button"
import Button from "../button/Button.vue"

interface Props {
  as?: any
  asChild?: boolean
  class?: HTMLAttributes["class"]
  variant?: ButtonVariants["variant"]
  size?: ButtonVariants["size"]
}

const props = withDefaults(defineProps<Props>(), {
  variant: "ghost",
  size: "icon-xs",
})
</script>

<template>
  <Button
    data-slot="attachment-action"
    :variant="variant"
    :size="size"
  >
    <slot />
  </Button>
</template>
