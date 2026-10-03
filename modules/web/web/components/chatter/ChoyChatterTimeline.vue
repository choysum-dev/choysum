<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-chatter-timeline" data-anchor="choy.chatter.timeline">
    <div
      v-if="loading && entries.length === 0"
      class="flex justify-center px-3 py-6"
    >
      <ChoySpinner :label="loadingLabel || 'Loading activity...'" />
    </div>
    <div
      v-else-if="error"
      class="px-3 py-4 text-center text-sm text-destructive"
      role="alert"
    >
      {{ error }}
    </div>
    <ChoyEmpty
      v-else-if="entries.length === 0"
      class="w-full border-none"
      :description="emptyLabel || 'No activity yet'"
    />
    <MessageScrollerProvider v-else default-scroll-position="end">
      <MessageScroller class="max-h-[28rem] min-h-0">
        <MessageScrollerViewport class="px-0.5">
          <MessageScrollerContent class="flex flex-col gap-2.5 py-1">
            <MessageScrollerItem
              v-for="entry in entries"
              :key="`${entry.kind}:${entry.id}`"
              :message-id="`${entry.kind}:${entry.id}`"
            >
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
            </MessageScrollerItem>
          </MessageScrollerContent>
        </MessageScrollerViewport>
      </MessageScroller>
    </MessageScrollerProvider>
  </div>
</template>

<script setup lang="ts">
import type { ChatterTimelineEntry } from './chatterTypes';
import { resolveChoyChatterAuthorLabel } from './chatterHelpers';
import ChoyChatterFieldChangeItem from './ChoyChatterFieldChangeItem.vue';
import ChoyChatterMessageItem from './ChoyChatterMessageItem.vue';
import ChoyEmpty from '../layout/ChoyEmpty.vue';
import ChoySpinner from '../layout/ChoySpinner.vue';
import MessageScroller from '../vendor/ui/message-scroller/MessageScroller.vue';
import MessageScrollerContent from '../vendor/ui/message-scroller/MessageScrollerContent.vue';
import MessageScrollerItem from '../vendor/ui/message-scroller/MessageScrollerItem.vue';
import MessageScrollerProvider from '../vendor/ui/message-scroller/MessageScrollerProvider.vue';
import MessageScrollerViewport from '../vendor/ui/message-scroller/MessageScrollerViewport.vue';

withDefaults(
  defineProps<{
    entries?: ChatterTimelineEntry[];
    loading?: boolean;
    error?: string | null;
    resolveAuthorLabel?: (userId: string | null | undefined) => string;
    loadingLabel?: string;
    emptyLabel?: string;
  }>(),
  {
    entries: () => [],
    loading: false,
    error: null,
    resolveAuthorLabel: (userId: string | null | undefined) =>
      resolveChoyChatterAuthorLabel(userId),
  },
);
</script>
