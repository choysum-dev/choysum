<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-chatter-timeline">
    <div v-if="loading" class="o-chatter-timeline__state">{{ _t('Loading activity...') }}</div>
    <div v-else-if="error" class="o-chatter-timeline__state o-chatter-timeline__state--error">{{ error }}</div>
    <div v-else-if="entries.length === 0" class="o-chatter-timeline__state">{{ _t('No activity yet') }}</div>
    <div v-else class="o-chatter-timeline__list">
      <template v-for="entry in entries" :key="`${entry.kind}:${entry.id}`">
        <OChatterMessageItem
          v-if="entry.kind === 'message'"
          :entry="entry"
          :author-label="resolveAuthorLabel(entry.authorUid)"
        />
        <OChatterFieldChangeItem
          v-else
          :entry="entry"
          :author-label="resolveAuthorLabel(entry.actorUid)"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ChatterTimelineEntry } from '@/web/web/composables/chatter/chatterTypes';
import { createTranslate } from '@/web/web/i18n';
import OChatterMessageItem from './OChatterMessageItem.vue';
import OChatterFieldChangeItem from './OChatterFieldChangeItem.vue';

defineProps<{
  entries: ChatterTimelineEntry[];
  loading: boolean;
  error: string | null;
  resolveAuthorLabel: (userId: string | null | undefined) => string;
}>();

const { _t } = createTranslate('web', { scope: 'web/components/chatter/OChatterTimeline' });
</script>

<style scoped>
.o-chatter-timeline__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.o-chatter-timeline__state {
  padding: 16px 12px;
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.o-chatter-timeline__state--error {
  color: var(--el-color-danger);
}
</style>
