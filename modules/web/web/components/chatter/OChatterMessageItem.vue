<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-chatter-message">
    <div class="o-chatter-message__meta">
      <span class="o-chatter-message__author">{{ authorLabel }}</span>
      <span class="o-chatter-message__time">{{ timeLabel }}</span>
    </div>
    <div class="o-chatter-message__body">{{ entry.body }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { ChatterMessageEntry } from '@/web/web/composables/chatter/chatterTypes';
import { formatUtcIso } from '@/web/web/utils/datetime';

const props = defineProps<{
  entry: ChatterMessageEntry;
  authorLabel: string;
}>();

const timeLabel = computed(() => formatUtcIso(props.entry.at, 'YYYY-MM-DD HH:mm') || '');
</script>

<style scoped>
.o-chatter-message {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-blank);
}

.o-chatter-message__meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.o-chatter-message__author {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.o-chatter-message__body {
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--el-text-color-primary);
}
</style>
