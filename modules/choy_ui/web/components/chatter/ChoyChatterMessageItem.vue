<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import type { ChatterMessageEntry } from './chatterTypes';
import { formatChoyUtcIso } from './chatterHelpers';

const props = defineProps<{
  entry: ChatterMessageEntry;
  authorLabel: string;
}>();

const timeLabel = computed(() => formatChoyUtcIso(props.entry.at));
</script>

<template>
  <div
    class="choy-chatter-message flex flex-col gap-1.5 rounded-md border border-border bg-background px-3 py-2.5"
    data-anchor="choy.chatter.message"
  >
    <div class="flex justify-between gap-2 text-xs text-muted-foreground">
      <span class="font-semibold text-foreground">{{ authorLabel }}</span>
      <span>{{ timeLabel }}</span>
    </div>
    <div class="whitespace-pre-wrap break-words text-sm text-foreground">{{ entry.body }}</div>
  </div>
</template>
