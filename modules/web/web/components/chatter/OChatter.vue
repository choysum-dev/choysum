<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-card shadow="never" class="o-chatter" data-region="chatter">
    <template #header>
      <div class="o-chatter__header">
        <span>{{ _t('Activity') }}</span>
        <OChatterFollowerBar :model="model" :res-id="resId" :disabled="disabled" />
      </div>
    </template>

    <OChatterComposer
      v-if="showComposer"
      :model="model"
      :res-id="resId"
      :disabled="disabled"
      @posted="handlePosted"
    />

    <OChatterTimeline
      :entries="entries"
      :loading="loading"
      :error="error"
      :resolve-author-label="resolveAuthorLabel"
    />
  </el-card>
</template>

<script setup lang="ts">
import { computed, toRef } from 'vue';
import { ElCard } from 'element-plus';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createTranslate } from '@/web/web/i18n';
import { useChatterTimeline } from '@/web/web/composables/chatter/useChatterTimeline';
import { useChatterThreadTips } from '@/web/web/composables/chatter/useChatterThreadTips';
import OChatterComposer from './OChatterComposer.vue';
import OChatterTimeline from './OChatterTimeline.vue';
import OChatterFollowerBar from './OChatterFollowerBar.vue';

defineOptions({ name: 'OChatter' });

const props = withDefaults(
  defineProps<{
    model: string;
    resId?: string;
    disabled?: boolean;
    showComposer?: boolean;
  }>(),
  {
    resId: '',
    disabled: false,
    showComposer: true,
  }
);

const { _t } = createTranslate('web', { scope: 'web/components/chatter/OChatter' });
const authStore = useAuthStore();
const modelRef = toRef(props, 'model');
const resIdRef = toRef(props, 'resId');
const { entries, loading, error, refresh } = useChatterTimeline(modelRef, resIdRef);

useChatterThreadTips(modelRef, resIdRef, refresh);

const showComposer = computed(() => props.showComposer && !!String(props.resId || '').trim() && !props.disabled);

function resolveAuthorLabel(userId: string | null | undefined): string {
  const normalized = String(userId || '').trim();
  if (!normalized) return _t('System');
  const currentId = String((authStore.currentUser as any)?.Id || '').trim();
  if (currentId && normalized === currentId) {
    const name = String((authStore.currentUser as any)?.Name || '').trim();
    return name || _t('You');
  }
  return normalized;
}

async function handlePosted(): Promise<void> {
  await refresh();
}
</script>

<style scoped>
.o-chatter {
  margin-top: 14px;
}

.o-chatter__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-weight: 600;
}

.o-chatter :deep(.o-chatter-composer) {
  margin-bottom: 12px;
}
</style>
