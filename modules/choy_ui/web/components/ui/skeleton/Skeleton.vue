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
    data-slot="skeleton"
    :class="cn('animate-pulse rounded-md bg-muted', $props.class)"
  />
</template>
