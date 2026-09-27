<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="choy-chatter-follower-bar flex items-center gap-2"
    data-anchor="choy.chatter.follower-bar"
  >
    <Button
      size="sm"
      variant="outline"
      :disabled="!enabled"
      @click="toggle"
    >
      {{ loading ? '…' : following ? unfollowLabel : followLabel }}
    </Button>
    <span
      v-if="followerCount > 0"
      class="text-xs text-muted-foreground"
    >
      {{ followerCount }} {{ followerCount === 1 ? 'follower' : 'followers' }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import Button from '../vendor/ui/button/Button.vue';

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
    followLabel: 'Follow',
    unfollowLabel: 'Unfollow',
  },
);

const emit = defineEmits<{
  follow: [];
  unfollow: [];
}>();

const enabled = computed(
  () => props.canToggle && !props.disabled && !props.loading,
);

function toggle(): void {
  if (!enabled.value) return;
  if (props.following) emit('unfollow');
  else emit('follow');
}
</script>
