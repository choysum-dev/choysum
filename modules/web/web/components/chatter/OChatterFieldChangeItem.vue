<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-chatter-field-change">
    <div class="o-chatter-field-change__meta">
      <span class="o-chatter-field-change__author">{{ authorLabel }}</span>
      <span class="o-chatter-field-change__time">{{ timeLabel }}</span>
    </div>
    <div class="o-chatter-field-change__summary">{{ summary }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { ChatterFieldChangeEntry } from '@/web/web/composables/chatter/chatterTypes';
import { formatFieldChangeSummary } from './chatterHelpers';
import { formatUtcIso } from '@/web/web/utils/datetime';
import { createTranslate } from '@/web/web/i18n';

const props = defineProps<{
  entry: ChatterFieldChangeEntry;
  authorLabel: string;
}>();

const { _t } = createTranslate('web', { scope: 'web/components/chatter/OChatterFieldChangeItem' });

const timeLabel = computed(() => formatUtcIso(props.entry.at, 'YYYY-MM-DD HH:mm') || '');

const summary = computed(() =>
  formatFieldChangeSummary(props.entry, {
    created: _t('Record created'),
    unlinked: _t('Record removed'),
    changed: (field, oldValue, newValue) => _t('%s changed from %s to %s', field, oldValue, newValue),
    action: name => _t('Action: %s', name),
    fieldFallback: _t('Field'),
  })
);
</script>

<style scoped>
.o-chatter-field-change {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border-inline-start: 3px solid var(--el-color-info);
  background: var(--el-fill-color-light);
  border-radius: var(--el-border-radius-base);
}

.o-chatter-field-change__meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.o-chatter-field-change__author {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.o-chatter-field-change__summary {
  color: var(--el-text-color-regular);
}
</style>
