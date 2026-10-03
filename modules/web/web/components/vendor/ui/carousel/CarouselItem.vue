<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { WithClassAsProps } from "./interface"
import { cn } from "../../../../lib/utils"
import { useCarousel } from "./useCarousel"

const props = defineProps<WithClassAsProps>()

const { orientation } = useCarousel()
</script>

<template>
  <div
    data-slot="carousel-item"
    role="group"
    aria-roledescription="slide"
    :class="cn(
      'min-w-0 shrink-0 grow-0 basis-full',
      orientation === 'horizontal' ? 'ps-4' : 'pt-4',
      props.class,
    )"
  >
    <slot />
  </div>
</template>
