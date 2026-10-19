<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { MessageScrollerProviderProps } from "./useMessageScroller"
import { provideMessageScroller } from "./useMessageScroller"

const props = defineProps<MessageScrollerProviderProps>()

provideMessageScroller(props)
</script>

<template>
  <slot />
</template>
