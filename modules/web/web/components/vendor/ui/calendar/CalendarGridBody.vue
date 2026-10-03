<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script lang="ts" setup>
import type { CalendarGridBodyProps } from "reka-ui"
import { CalendarGridBody } from "reka-ui"

const props = defineProps<CalendarGridBodyProps>()
</script>

<template>
  <CalendarGridBody
    data-slot="calendar-grid-body"
    v-bind="props"
  >
    <slot />
  </CalendarGridBody>
</template>
