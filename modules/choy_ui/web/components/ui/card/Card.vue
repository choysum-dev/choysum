<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { cn, type ClassValue } from '../../../lib/utils';

defineProps<{ class?: ClassValue }>();
</script>

<template>
  <div
    data-slot="card"
    :class="cn('rounded-lg border border-border bg-background text-foreground shadow-sm', $props.class)"
  >
    <slot />
  </div>
</template>
