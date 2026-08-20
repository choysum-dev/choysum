<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-chatter-composer">
    <el-input
      v-model="body"
      type="textarea"
      :rows="3"
      :placeholder="_t('Write a comment...')"
      :disabled="posting || disabled"
      @keydown.ctrl.enter.prevent="submit"
      @keydown.meta.enter.prevent="submit"
    />
    <div class="o-chatter-composer__actions">
      <el-button type="primary" size="small" :loading="posting" :disabled="disabled || !canSubmit" @click="submit">
        {{ _t('Post') }}
      </el-button>
    </div>
    <p v-if="error" class="o-chatter-composer__error" role="alert">{{ error }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { ElButton, ElInput } from 'element-plus';
import { getMessageStore } from '@/web/web/composables/chatter/chatterStores';
import { createTranslate } from '@/web/web/i18n';

const props = defineProps<{
  model: string;
  resId: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  posted: [];
}>();

const { _t } = createTranslate('web', { scope: 'web/components/chatter/OChatterComposer' });
const messageStore = getMessageStore();
const body = ref('');
const posting = ref(false);
const error = ref<string | null>(null);

const canSubmit = computed(() => body.value.trim().length > 0);

async function submit(): Promise<void> {
  const text = body.value.trim();
  if (!text || posting.value || props.disabled) return;
  posting.value = true;
  error.value = null;
  try {
    await messageStore.Post({
      Model: props.model,
      ResId: props.resId,
      Body: text,
    });
    body.value = '';
    emit('posted');
  } catch (err) {
    error.value = err instanceof Error && err.message.trim() ? err.message : _t('Failed to post comment');
  } finally {
    posting.value = false;
  }
}
</script>

<style scoped>
.o-chatter-composer {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.o-chatter-composer__actions {
  display: flex;
  justify-content: flex-end;
}

.o-chatter-composer__error {
  margin: 0;
  font-size: 12px;
  color: var(--el-color-danger);
}
</style>
