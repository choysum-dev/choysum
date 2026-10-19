<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="choy-chatter mt-3.5"
    data-anchor="choy.chatter"
    data-region="chatter"
  >
    <ChoyCard>
      <template #header>
        <div class="flex w-full items-center justify-between gap-3 font-semibold">
          <span>{{ resolvedTitle }}</span>
          <ChoyChatterFollowerBar
            :following="resolvedFollowing"
            :follower-count="resolvedFollowerCount"
            :loading="resolvedFollowersLoading"
            :disabled="disabled"
            :can-toggle="canToggleFollow"
            @follow="onFollow"
            @unfollow="onUnfollow"
          />
        </div>
      </template>

      <ChoyChatterComposer
        v-if="composerVisible"
        ref="composerRef"
        class="mb-3"
        :disabled="disabled"
        :posting="resolvedPosting"
        :error="resolvedPostError"
        @post="onPost"
      />

      <ChoyChatterTimeline
        :entries="resolvedEntries"
        :loading="resolvedLoading"
        :error="resolvedError"
        :resolve-author-label="resolveAuthorLabel"
        :loading-label="_t('Loading activity...')"
        :empty-label="_t('No activity yet')"
      />
    </ChoyCard>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useAuthStore } from '@/auth/web/stores/auth';
import {
  useInjectedFollowerStore,
  useInjectedMessageStore
} from '@/web/web/composables/chatter/chatterStores';
import { useInjectedChatterTimeline } from '@/web/web/composables/chatter/useChatterTimeline';
import { useInjectedChatterThreadTips } from '@/web/web/composables/chatter/useChatterThreadTips';
import { createTranslate } from '@/web/web/i18n';
import ChoyCard from '../layout/ChoyCard.vue';
import type { ChatterTimelineEntry } from './chatterTypes';
import { resolveChoyChatterAuthorLabel } from './chatterHelpers';
import ChoyChatterComposer from './ChoyChatterComposer.vue';
import ChoyChatterFollowerBar from './ChoyChatterFollowerBar.vue';
import ChoyChatterTimeline from './ChoyChatterTimeline.vue';

const { _t } = createTranslate('web', { scope: 'web/components/chatter/ChoyChatter' });

/**
 * Chatter shell. Chrome mode: timeline / composer / followers via props + emits.
 * Store mode (`bindStore`): loads timeline, posts, and follow state from message stores.
 */
const props = withDefaults(
  defineProps<{
    model: string;
    resId?: string;
    disabled?: boolean;
    showComposer?: boolean;
    /** When true, bind message/follower stores (product forms). */
    bindStore?: boolean;
    entries?: ChatterTimelineEntry[];
    loading?: boolean;
    error?: string | null;
    following?: boolean;
    followerCount?: number;
    followersLoading?: boolean;
    posting?: boolean;
    postError?: string | null;
    currentUserId?: string | null;
    currentUserName?: string | null;
    title?: string;
  }>(),
  {
    resId: '',
    disabled: false,
    showComposer: true,
    bindStore: false,
    entries: () => [],
    loading: false,
    error: null,
    following: false,
    followerCount: 0,
    followersLoading: false,
    posting: false,
    postError: null,
    currentUserId: null,
    currentUserName: null,
    title: undefined,
  },
);

const resolvedTitle = computed(() => props.title?.trim() || _t('Activity'));

const emit = defineEmits<{
  post: [body: string];
  follow: [];
  unfollow: [];
}>();

const composerRef = ref<{ clear: () => void } | null>(null);
const postingResId = ref<string | null>(null);
const postingModel = ref<string | null>(null);

const authStore = useAuthStore();
const messageStore = useInjectedMessageStore();
const followerStore = useInjectedFollowerStore();

/** Empty model/resId when chrome-only so store timeline stays idle. */
const boundModel = computed(() => (props.bindStore ? String(props.model || '') : ''));
const boundResId = computed(() => (props.bindStore ? props.resId : undefined));

const {
  entries: storeEntries,
  loading: storeLoading,
  error: storeError,
  refresh: refreshTimeline,
} = useInjectedChatterTimeline(boundModel, boundResId);

useInjectedChatterThreadTips(boundModel, boundResId, refreshTimeline);

const storeFollowing = ref(false);
const storeFollowerCount = ref(0);
const storeFollowersLoading = ref(false);
const storePosting = ref(false);
const storePostError = ref<string | null>(null);
let followersGeneration = 0;

const currentUserIdResolved = computed(() => {
  if (props.currentUserId != null && String(props.currentUserId).trim() !== '') {
    return String(props.currentUserId).trim();
  }
  if (!props.bindStore) return '';
  return String((authStore.currentUser as { Id?: string } | null)?.Id || '').trim();
});

const currentUserNameResolved = computed(() => {
  if (props.currentUserName != null && String(props.currentUserName).trim() !== '') {
    return String(props.currentUserName).trim();
  }
  if (!props.bindStore) return '';
  return String((authStore.currentUser as { Name?: string } | null)?.Name || '').trim();
});

function isPostingContextCurrent(targetModel: string, targetResId: string): boolean {
  return (
    postingModel.value === targetModel &&
    postingResId.value === targetResId &&
    String(props.model || '') === targetModel &&
    String(props.resId || '') === targetResId
  );
}

async function refreshFollowers(): Promise<void> {
  if (!props.bindStore) return;
  const generation = ++followersGeneration;
  const threadModel = String(props.model || '').trim();
  const threadResId = String(props.resId || '').trim();
  const userId = currentUserIdResolved.value;
  if (!threadModel || !threadResId) {
    if (generation !== followersGeneration) return;
    storeFollowing.value = false;
    storeFollowerCount.value = 0;
    storeFollowersLoading.value = false;
    return;
  }
  storeFollowersLoading.value = true;
  try {
    const rows = await followerStore.SearchByRecord(threadModel, threadResId, ['UserId']);
    if (generation !== followersGeneration) return;
    storeFollowerCount.value = rows.length;
    storeFollowing.value = rows.some(
      (row) => String(row?.UserId || '').trim() === userId,
    );
  } catch {
    if (generation !== followersGeneration) return;
    // Defined failure state: clear follow snapshot so the bar does not keep stale data.
    storeFollowing.value = false;
    storeFollowerCount.value = 0;
  } finally {
    if (generation === followersGeneration) {
      storeFollowersLoading.value = false;
    }
  }
}

watch(
  () => [props.bindStore, props.model, props.resId, currentUserIdResolved.value] as const,
  () => {
    void refreshFollowers();
  },
  { immediate: true },
);

watch(
  () => [props.model, props.resId] as const,
  () => {
    // Invalidate in-flight post so switching model/resId (incl. A→B→A) cannot
    // let a stale completion clear a draft typed for the new context.
    postingModel.value = null;
    postingResId.value = null;
    storePostError.value = null;
    composerRef.value?.clear();
  },
);

const resolvedEntries = computed(() =>
  props.bindStore ? storeEntries.value : (props.entries ?? []),
);
const resolvedLoading = computed(() =>
  props.bindStore ? storeLoading.value : !!props.loading,
);
const resolvedError = computed(() =>
  props.bindStore ? storeError.value : (props.error ?? null),
);
const resolvedFollowing = computed(() =>
  props.bindStore ? storeFollowing.value : !!props.following,
);
const resolvedFollowerCount = computed(() =>
  props.bindStore ? storeFollowerCount.value : (props.followerCount ?? 0),
);
const resolvedFollowersLoading = computed(() =>
  props.bindStore ? storeFollowersLoading.value : !!props.followersLoading,
);
const resolvedPosting = computed(() =>
  props.bindStore ? storePosting.value : !!props.posting,
);
const resolvedPostError = computed(() =>
  props.bindStore ? storePostError.value : (props.postError ?? null),
);

const composerVisible = computed(
  () => props.showComposer && !!String(props.resId || '').trim() && !props.disabled,
);

const canToggleFollow = computed(
  () =>
    !!currentUserIdResolved.value &&
    !!String(props.model || '').trim() &&
    !!String(props.resId || '').trim(),
);

function resolveAuthorLabel(userId: string | null | undefined): string {
  return resolveChoyChatterAuthorLabel(userId, {
    currentUserId: currentUserIdResolved.value || null,
    currentUserName: currentUserNameResolved.value || null,
  });
}

async function onPost(body: string): Promise<void> {
  if (props.bindStore) {
    const text = body.trim();
    if (!text || storePosting.value || props.disabled) return;
    const targetModel = String(props.model || '');
    const targetResId = String(props.resId || '');
    storePosting.value = true;
    storePostError.value = null;
    postingModel.value = targetModel;
    postingResId.value = targetResId;
    try {
      await messageStore.Post({
        Model: targetModel,
        ResId: targetResId,
        Body: text,
      });
      // Only clear / refresh when still on the posting context (A→B→A guard).
      if (isPostingContextCurrent(targetModel, targetResId)) {
        composerRef.value?.clear();
        await refreshTimeline();
      }
    } catch (err) {
      if (isPostingContextCurrent(targetModel, targetResId)) {
        storePostError.value =
          err instanceof Error && err.message.trim() ? err.message : 'Failed to post comment';
      }
    } finally {
      if (isPostingContextCurrent(targetModel, targetResId)) {
        storePosting.value = false;
      } else if (postingModel.value === null && postingResId.value === null) {
        // Context switched mid-flight; drop the orphan posting flag.
        storePosting.value = false;
      }
    }
    return;
  }
  emit('post', body);
}

async function onFollow(): Promise<void> {
  if (props.bindStore) {
    if (!canToggleFollow.value || storeFollowersLoading.value || props.disabled) return;
    storeFollowersLoading.value = true;
    try {
      await followerStore.Follow({ Model: props.model, ResId: String(props.resId || '') });
      await refreshFollowers();
    } catch {
      // Leave prior follow snapshot; loading cleared below.
    } finally {
      storeFollowersLoading.value = false;
    }
    return;
  }
  emit('follow');
}

async function onUnfollow(): Promise<void> {
  if (props.bindStore) {
    if (!canToggleFollow.value || storeFollowersLoading.value || props.disabled) return;
    storeFollowersLoading.value = true;
    try {
      await followerStore.Unfollow({ Model: props.model, ResId: String(props.resId || '') });
      await refreshFollowers();
    } catch {
      // Leave prior follow snapshot; loading cleared below.
    } finally {
      storeFollowersLoading.value = false;
    }
    return;
  }
  emit('unfollow');
}

watch(
  () => props.posting,
  (next, prev) => {
    if (props.bindStore) return;
    if (next === true) {
      postingModel.value = String(props.model || '');
      postingResId.value = String(props.resId || '');
      return;
    }
    // Clear only when the finished post still belongs to the current context.
    if (
      prev === true &&
      next === false &&
      postingModel.value === String(props.model || '') &&
      postingResId.value === String(props.resId || '')
    ) {
      // Hosts may flip posting false before assigning postError in the same
      // tick (sync flush sees posting first). Re-check on the next microtask.
      void Promise.resolve().then(() => {
        if (
          !props.posting &&
          !props.postError &&
          postingModel.value === String(props.model || '') &&
          postingResId.value === String(props.resId || '')
        ) {
          composerRef.value?.clear();
        }
      });
    }
  },
  // Sync flush: a host that sets posting true→false within one tick would
  // otherwise coalesce to "no change" and never clear the draft.
  // Immediate: capture posting context when mounting mid-flight.
  { flush: 'sync', immediate: true },
);

/** Clears the composer draft after a confirmed successful post (host-driven). */
function clear(): void {
  composerRef.value?.clear();
}

defineExpose({ clear });
</script>
