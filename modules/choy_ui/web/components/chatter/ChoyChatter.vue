<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import ChoyCard from '../layout/ChoyCard.vue';
import type { ChatterTimelineEntry } from './chatterTypes';
import { resolveChoyChatterAuthorLabel } from './chatterHelpers';
import ChoyChatterComposer from './ChoyChatterComposer.vue';
import ChoyChatterFollowerBar from './ChoyChatterFollowerBar.vue';
import ChoyChatterTimeline from './ChoyChatterTimeline.vue';

/**
 * Isolation Chatter shell: timeline / composer / followers via props + emits.
 * Hosts wire message/follower stores (or dogfood local state) at the boundary.
 */
const props = withDefaults(
  defineProps<{
    model: string;
    resId?: string;
    disabled?: boolean;
    showComposer?: boolean;
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
    title: 'Activity',
  },
);

const emit = defineEmits<{
  post: [body: string];
  follow: [];
  unfollow: [];
}>();

const composerRef = ref<{ clear: () => void } | null>(null);

const composerVisible = computed(
  () =>
    props.showComposer &&
    !!String(props.resId || '').trim() &&
    !props.disabled,
);

const canToggleFollow = computed(
  () =>
    !!String(props.currentUserId || '').trim() &&
    !!String(props.model || '').trim() &&
    !!String(props.resId || '').trim(),
);

function resolveAuthorLabel(userId: string | null | undefined): string {
  return resolveChoyChatterAuthorLabel(userId, {
    currentUserId: props.currentUserId,
    currentUserName: props.currentUserName,
  });
}

function onPost(body: string): void {
  emit('post', body);
}

watch(
  () => props.posting,
  (next, prev) => {
    // Clear draft when a post finishes successfully (posting true → false, no error).
    if (prev === true && next === false && !props.postError) {
      composerRef.value?.clear();
    }
  },
);

/** Clears the composer draft after a confirmed successful post (host-driven). */
function clear(): void {
  composerRef.value?.clear();
}

defineExpose({ clear });
</script>

<template>
  <div
    class="choy-chatter mt-3.5"
    data-anchor="choy.chatter"
    data-region="chatter"
  >
    <ChoyCard>
      <template #header>
        <div class="flex w-full items-center justify-between gap-3 font-semibold">
          <span>{{ title }}</span>
          <ChoyChatterFollowerBar
            :following="following"
            :follower-count="followerCount"
            :loading="followersLoading"
            :disabled="disabled"
            :can-toggle="canToggleFollow"
            @follow="emit('follow')"
            @unfollow="emit('unfollow')"
          />
        </div>
      </template>

      <ChoyChatterComposer
        v-if="composerVisible"
        ref="composerRef"
        class="mb-3"
        :disabled="disabled"
        :posting="posting"
        :error="postError"
        @post="onPost"
      />

      <ChoyChatterTimeline
        :entries="entries"
        :loading="loading"
        :error="error"
        :resolve-author-label="resolveAuthorLabel"
      />
    </ChoyCard>
  </div>
</template>
