<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <div
    data-slot="card-header"
    :class="
      cn(
        '@container/card-header grid auto-rows-min grid-rows-[auto_auto] items-start gap-1.5 px-6 has-data-[slot=card-action]:grid-cols-[1fr_auto] [.border-b]:pb-6',
        $props.class,
      )
    "
  >
    <slot />
  </div>
</template>
