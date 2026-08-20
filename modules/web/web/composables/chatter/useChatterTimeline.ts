// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, watch, type Ref } from 'vue';
import type { ChatterTimelineEntry } from './chatterTypes';
import { getFieldChangeStore, getMessageStore } from './chatterStores';
import { mergeChatterTimeline } from './mergeChatterTimeline';

const MESSAGE_FIELDS = ['Id', 'Type', 'Body', 'AuthorUid', 'CreatedAt'] as const;
const FIELD_CHANGE_FIELDS = ['Id', 'Field', 'Kind', 'OldValue', 'NewValue', 'ActorUid', 'At'] as const;

export function useChatterTimeline(model: Ref<string>, resId: Ref<string | undefined>) {
  const entries = ref<ChatterTimelineEntry[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  const messageStore = getMessageStore();
  const fieldChangeStore = getFieldChangeStore();

  async function refresh(): Promise<void> {
    const threadModel = String(model.value || '').trim();
    const threadResId = String(resId.value || '').trim();
    if (!threadModel || !threadResId) {
      entries.value = [];
      error.value = null;
      return;
    }

    loading.value = true;
    error.value = null;
    try {
      const [messages, fieldChanges] = await Promise.all([
        messageStore.SearchByRecord(threadModel, threadResId, [...MESSAGE_FIELDS]),
        fieldChangeStore.SearchByRecord(threadModel, threadResId, [...FIELD_CHANGE_FIELDS]),
      ]);
      entries.value = mergeChatterTimeline(messages, fieldChanges);
    } catch (err) {
      entries.value = [];
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  }

  watch([model, resId], () => {
    void refresh();
  }, { immediate: true });

  return { entries, loading, error, refresh };
}
