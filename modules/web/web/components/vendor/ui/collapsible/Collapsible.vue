<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { CollapsibleRoot } from "reka-ui"

const open = defineModel<boolean>("open")

const props = defineProps<{
  defaultOpen?: boolean
  disabled?: boolean
  unmountOnHide?: boolean
}>()
</script>

<template>
  <CollapsibleRoot
    v-slot="slotProps"
    v-model:open="open"
    data-slot="collapsible"
    :default-open="props.defaultOpen"
    :disabled="props.disabled"
    :unmount-on-hide="props.unmountOnHide"
  >
    <slot v-bind="slotProps" />
  </CollapsibleRoot>
</template>
