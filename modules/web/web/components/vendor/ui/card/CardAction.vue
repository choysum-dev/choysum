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
    data-slot="card-action"
    :class="cn('col-start-2 row-span-2 row-start-1 self-start justify-self-end', $props.class)"
  >
    <slot />
  </div>
</template>
