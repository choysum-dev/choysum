<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <div data-slot="card-header" :class="cn('flex flex-col gap-1 p-3 pb-2', $props.class)">
    <slot />
  </div>
</template>
