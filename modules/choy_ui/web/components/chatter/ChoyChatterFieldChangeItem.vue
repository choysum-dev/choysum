<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import type { ChatterFieldChangeEntry } from './chatterTypes';
import { formatChoyUtcIso, formatFieldChangeSummary } from './chatterHelpers';

const props = defineProps<{
  entry: ChatterFieldChangeEntry;
  authorLabel: string;
}>();

const timeLabel = computed(() => formatChoyUtcIso(props.entry.at));

const summary = computed(() =>
  formatFieldChangeSummary(props.entry, {
    created: 'Record created',
    unlinked: 'Record removed',
    changed: (field, oldValue, newValue) => `${field} changed from ${oldValue} to ${newValue}`,
    action: name => `Action: ${name}`,
    fieldFallback: 'Field',
  }),
);
</script>

<template>
  <div
    class="choy-chatter-field-change flex flex-col gap-1.5 rounded-md border-l-[3px] border-l-info bg-muted/40 px-3 py-2.5"
    data-anchor="choy.chatter.field-change"
  >
    <div class="flex justify-between gap-2 text-xs text-muted-foreground">
      <span class="font-semibold text-foreground">{{ authorLabel }}</span>
      <span>{{ timeLabel }}</span>
    </div>
    <div class="text-sm text-foreground/90">{{ summary }}</div>
  </div>
</template>
