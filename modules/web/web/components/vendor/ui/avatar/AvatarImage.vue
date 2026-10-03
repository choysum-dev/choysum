<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { AvatarImage } from "reka-ui"

defineProps<{
  src?: string
  alt?: string
}>()
</script>

<template>
  <AvatarImage
    v-if="src"
    data-slot="avatar-image"
    :src="src"
    :alt="alt || ''"
    class="aspect-square size-full"
  >
    <slot />
  </AvatarImage>
</template>
