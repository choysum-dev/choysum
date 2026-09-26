<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import type { ChatterTimelineEntry } from './chatterTypes';
import ChoyChatterFieldChangeItem from './ChoyChatterFieldChangeItem.vue';
import ChoyChatterMessageItem from './ChoyChatterMessageItem.vue';

defineProps<{
  entries: ChatterTimelineEntry[];
  loading?: boolean;
  error?: string | null;
  resolveAuthorLabel: (userId: string | null | undefined) => string;
  loadingLabel?: string;
  emptyLabel?: string;
}>();
</script>

<template>
  <div class="choy-chatter-timeline" data-anchor="choy.chatter.timeline">
    <div
      v-if="loading"
      class="px-3 py-4 text-center text-sm text-muted-foreground"
    >
      {{ loadingLabel || 'Loading activity...' }}
    </div>
    <div
      v-else-if="error"
      class="px-3 py-4 text-center text-sm text-destructive"
      role="alert"
    >
      {{ error }}
    </div>
    <div
      v-else-if="entries.length === 0"
      class="px-3 py-4 text-center text-sm text-muted-foreground"
    >
      {{ emptyLabel || 'No activity yet' }}
    </div>
    <div v-else class="flex flex-col gap-2.5">
      <template v-for="entry in entries" :key="`${entry.kind}:${entry.id}`">
        <ChoyChatterMessageItem
          v-if="entry.kind === 'message'"
          :entry="entry"
          :author-label="resolveAuthorLabel(entry.authorUid)"
        />
        <ChoyChatterFieldChangeItem
          v-else
          :entry="entry"
          :author-label="resolveAuthorLabel(entry.actorUid)"
        />
      </template>
    </div>
  </div>
</template>
