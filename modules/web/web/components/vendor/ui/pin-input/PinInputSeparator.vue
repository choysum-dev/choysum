<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { PrimitiveProps } from "reka-ui"
import { Minus } from 'lucide-vue-next'
import { Primitive, useForwardProps } from "reka-ui"

const props = defineProps<PrimitiveProps>()
const forwardedProps = useForwardProps(props)
</script>

<template>
  <Primitive
    data-slot="pin-input-separator"
    v-bind="forwardedProps"
  >
    <slot>
      <Minus />
    </slot>
  </Primitive>
</template>
