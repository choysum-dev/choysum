<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import type { ChatterFieldChangeEntry } from './chatterTypes';
import { formatChoyUtcIso, formatFieldChangeSummary } from './chatterHelpers';

const props = withDefaults(
  defineProps<{
    entry: ChatterFieldChangeEntry;
    authorLabel: string;
    labels?: {
      created?: string;
      unlinked?: string;
      changed?: (field: string, oldValue: string, newValue: string) => string;
      action?: (name: string) => string;
      fieldFallback?: string;
    };
  }>(),
  {
    labels: () => ({}),
  },
);

const timeLabel = computed(() => formatChoyUtcIso(props.entry.at));

const summary = computed(() =>
  formatFieldChangeSummary(props.entry, {
    created: props.labels.created || 'Record created',
    unlinked: props.labels.unlinked || 'Record removed',
    changed:
      props.labels.changed ||
      ((field, oldValue, newValue) => `${field} changed from ${oldValue} to ${newValue}`),
    action: props.labels.action || (name => `Action: ${name}`),
    fieldFallback: props.labels.fieldFallback || 'Field',
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
    <div class="whitespace-pre-wrap break-words text-sm text-foreground/90">{{ summary }}</div>
  </div>
</template>
