// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, inject, ref, watch, type InjectionKey, type Ref } from 'vue';
import {
  useReactiveCollaborationCaps,
  type CollaborationCaps,
} from '@/web/web/platform/collaborationCaps';
import type { ChatterTimelineEntry } from './chatterTypes';
import { getFieldChangeStore, getMessageStore } from './chatterStores';
import { mergeChatterTimeline } from './mergeChatterTimeline';

const MESSAGE_FIELDS = ['Id', 'Type', 'Body', 'AuthorUid', 'CreatedAt'] as const;
const FIELD_CHANGE_FIELDS = ['Id', 'Field', 'Kind', 'OldValue', 'NewValue', 'ActorUid', 'At'] as const;

export type UseChatterTimelineDeps = {
  getMessageStore?: typeof getMessageStore;
  getFieldChangeStore?: typeof getFieldChangeStore;
  /**
   * Explicit caps (unit harness). Omit to use reactive registry caps.
   * Pass a reactive object when the harness mutates caps after mount;
   * a plain object is a fixed snapshot.
   */
  caps?: CollaborationCaps;
};

/** Optional override for `useChatterTimeline` (unit harness). */
export type UseChatterTimelineFn = typeof useChatterTimeline;
export const UseChatterTimelineKey: InjectionKey<UseChatterTimelineFn> = Symbol('UseChatterTimeline');

/** Resolve injected timeline factory, else the product default. */
export function useInjectedChatterTimeline(
  model: Ref<string>,
  resId: Ref<string | undefined>,
  deps?: UseChatterTimelineDeps
): ReturnType<typeof useChatterTimeline> {
  const override = inject(UseChatterTimelineKey, null);
  return (override ?? useChatterTimeline)(model, resId, deps);
}

export function useChatterTimeline(
  model: Ref<string>,
  resId: Ref<string | undefined>,
  deps?: UseChatterTimelineDeps
) {
  const entries = ref<ChatterTimelineEntry[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  let refreshGeneration = 0;

  const capsRef = deps?.caps
    ? computed(() => deps.caps as CollaborationCaps)
    : useReactiveCollaborationCaps();
  const resolveMessageStore = deps?.getMessageStore ?? getMessageStore;
  const resolveFieldChangeStore = deps?.getFieldChangeStore ?? getFieldChangeStore;
  let messageStore: ReturnType<typeof resolveMessageStore> | null = null;
  let fieldChangeStore: ReturnType<typeof resolveFieldChangeStore> | null = null;

  /** Create stores lazily when the corresponding caps become true. */
  function ensureStores(caps: CollaborationCaps): void {
    if (caps.message && !messageStore) {
      messageStore = resolveMessageStore();
    }
    if (caps.message && caps.audit && !fieldChangeStore) {
      fieldChangeStore = resolveFieldChangeStore();
    }
  }

  async function refresh(): Promise<void> {
    const generation = ++refreshGeneration;
    const caps = capsRef.value;
    const threadModel = String(model.value || '').trim();
    const threadResId = String(resId.value || '').trim();
    if (!caps.message || !threadModel || !threadResId) {
      entries.value = [];
      error.value = null;
      loading.value = false;
      return;
    }
    ensureStores(caps);
    if (!messageStore) {
      entries.value = [];
      error.value = null;
      loading.value = false;
      return;
    }

    loading.value = true;
    error.value = null;
    try {
      const messagesPromise = messageStore.SearchByRecord(threadModel, threadResId, [...MESSAGE_FIELDS]);
      const fieldChangesPromise =
        caps.audit && fieldChangeStore
          ? fieldChangeStore.SearchByRecord(threadModel, threadResId, [...FIELD_CHANGE_FIELDS])
          : Promise.resolve([]);
      const [messages, fieldChanges] = await Promise.all([messagesPromise, fieldChangesPromise]);
      if (generation !== refreshGeneration) return;
      entries.value = mergeChatterTimeline(messages, fieldChanges);
    } catch (err) {
      if (generation !== refreshGeneration) return;
      entries.value = [];
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (generation === refreshGeneration) loading.value = false;
    }
  }

  watch(
    [model, resId, () => capsRef.value.message, () => capsRef.value.audit],
    () => {
      void refresh();
    },
    { immediate: true }
  );

  return { entries, loading, error, refresh };
}
