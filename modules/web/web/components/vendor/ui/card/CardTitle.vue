<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <h3 data-slot="card-title" :class="cn('font-semibold leading-none tracking-tight', $props.class)">
    <slot />
  </h3>
</template>
