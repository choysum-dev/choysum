<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-chatter-follower-bar">
    <el-button size="small" plain :loading="loading" :disabled="disabled || !canToggle" @click="toggleFollow">
      {{ following ? _t('Unfollow') : _t('Follow') }}
    </el-button>
    <span v-if="followerCount > 0" class="o-chatter-follower-bar__count">
      {{ _t('%d followers', followerCount) }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { ElButton } from 'element-plus';
import { getFollowerStore } from '@/web/web/composables/chatter/chatterStores';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createTranslate } from '@/web/web/i18n';

const props = defineProps<{
  model: string;
  resId: string;
  disabled?: boolean;
}>();

const { _t } = createTranslate('web', { scope: 'web/components/chatter/OChatterFollowerBar' });
const followerStore = getFollowerStore();
const authStore = useAuthStore();
const following = ref(false);
const followerCount = ref(0);
const loading = ref(false);

const currentUserId = computed(() => String((authStore.currentUser as any)?.Id || '').trim());
const canToggle = computed(() => !!currentUserId.value);

async function refresh(): Promise<void> {
  const threadModel = String(props.model || '').trim();
  const threadResId = String(props.resId || '').trim();
  if (!threadModel || !threadResId) {
    following.value = false;
    followerCount.value = 0;
    return;
  }
  loading.value = true;
  try {
    const rows = await followerStore.SearchByRecord(threadModel, threadResId, ['UserId']);
    followerCount.value = rows.length;
    following.value = rows.some(row => String(row?.UserId || '').trim() === currentUserId.value);
  } finally {
    loading.value = false;
  }
}

async function toggleFollow(): Promise<void> {
  if (!canToggle.value || loading.value || props.disabled) return;
  loading.value = true;
  try {
    if (following.value) {
      await followerStore.Unfollow({ Model: props.model, ResId: props.resId });
    } else {
      await followerStore.Follow({ Model: props.model, ResId: props.resId });
    }
    await refresh();
  } finally {
    loading.value = false;
  }
}

watch(
  () => [props.model, props.resId, currentUserId.value],
  () => {
    void refresh();
  },
  { immediate: true }
);
</script>

<style scoped>
.o-chatter-follower-bar {
  display: flex;
  align-items: center;
  gap: 10px;
}

.o-chatter-follower-bar__count {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
