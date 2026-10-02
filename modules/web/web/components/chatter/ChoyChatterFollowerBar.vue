<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="choy-chatter-follower-bar flex flex-wrap items-center gap-2"
    data-anchor="choy.chatter.follower-bar"
  >
    <ChoyButton
      size="sm"
      variant="outline"
      :disabled="!enabled"
      @click="toggle"
    >
      {{ loading ? '…' : following ? resolvedUnfollowLabel : resolvedFollowLabel }}
    </ChoyButton>
    <span
      v-if="followerCount > 0"
      class="text-xs text-muted-foreground"
    >
      {{ followerCountLabel }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ChoyButton from '../layout/ChoyButton.vue';
import { createTranslate } from '@/web/web/i18n';

/**
 * Follow / unfollow control. Host owns following state and persistence.
 */
const props = withDefaults(
  defineProps<{
    following?: boolean;
    followerCount?: number;
    loading?: boolean;
    disabled?: boolean;
    canToggle?: boolean;
    followLabel?: string;
    unfollowLabel?: string;
  }>(),
  {
    following: false,
    followerCount: 0,
    loading: false,
    disabled: false,
    canToggle: true,
    followLabel: undefined,
    unfollowLabel: undefined,
  },
);

const emit = defineEmits<{
  follow: [];
  unfollow: [];
}>();

const { _t } = createTranslate('web', { scope: 'web/components/chatter/ChoyChatterFollowerBar' });

const enabled = computed(
  () => props.canToggle && !props.disabled && !props.loading,
);
const resolvedFollowLabel = computed(() => props.followLabel ?? _t('Follow'));
const resolvedUnfollowLabel = computed(() => props.unfollowLabel ?? _t('Unfollow'));
const followerCountLabel = computed(() => {
  const count = Number(props.followerCount) || 0;
  return count === 1 ? _t('%d follower', count) : _t('%d followers', count);
});

function toggle(): void {
  if (!enabled.value) return;
  if (props.following) emit('unfollow');
  else emit('follow');
}
</script>
