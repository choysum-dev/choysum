// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { inject, ref, watch, type InjectionKey, type Ref } from 'vue';
import {
  useCollaborationCaps,
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

  const caps = resolveTimelineCaps(deps);
  const resolveMessageStore = deps?.getMessageStore ?? getMessageStore;
  const resolveFieldChangeStore = deps?.getFieldChangeStore ?? getFieldChangeStore;
  // Do not createStore when the module is absent — caps gate the dial.
  const messageStore = caps.message ? resolveMessageStore() : null;
  const fieldChangeStore = caps.message && caps.audit ? resolveFieldChangeStore() : null;

  async function refresh(): Promise<void> {
    const generation = ++refreshGeneration;
    const threadModel = String(model.value || '').trim();
    const threadResId = String(resId.value || '').trim();
    if (!caps.message || !messageStore || !threadModel || !threadResId) {
      entries.value = [];
      error.value = null;
      loading.value = false;
      return;
    }

    loading.value = true;
    error.value = null;
    try {
      const messagesPromise = messageStore.SearchByRecord(threadModel, threadResId, [...MESSAGE_FIELDS]);
      const fieldChangesPromise = fieldChangeStore
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

  watch([model, resId], () => {
    void refresh();
  }, { immediate: true });

  return { entries, loading, error, refresh };
}

/** Resolve caps from explicit deps, harness store getters, or the live registry. */
function resolveTimelineCaps(deps?: UseChatterTimelineDeps): CollaborationCaps {
  if (deps?.caps) return deps.caps;
  // Unit harnesses inject store getters without a registry — treat injected paths as available.
  if (deps?.getMessageStore || deps?.getFieldChangeStore) {
    return {
      message: deps.getMessageStore != null,
      audit: deps.getFieldChangeStore != null,
      document: true,
    };
  }
  return useCollaborationCaps();
}
