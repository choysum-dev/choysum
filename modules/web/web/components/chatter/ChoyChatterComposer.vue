<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-chatter-composer flex flex-col gap-2" data-anchor="choy.chatter.composer">
    <textarea
      v-model="body"
      :rows="3"
      :placeholder="resolvedPlaceholder"
      :disabled="posting || disabled"
      class="choy-input min-h-[4.5rem] h-auto py-2"
      @keydown.ctrl.enter="!$event.isComposing && ($event.preventDefault(), submit())"
      @keydown.meta.enter="!$event.isComposing && ($event.preventDefault(), submit())"
    ></textarea>
    <div class="flex justify-end">
      <ChoyActionButton
        size="sm"
        :disabled="disabled || !canSubmit"
        :busy="posting"
        @execute="submit"
      >
        {{ posting ? postingLabel : resolvedPostLabel }}
      </ChoyActionButton>
    </div>
    <p
      v-if="error"
      class="m-0 text-xs text-destructive"
      role="alert"
    >
      {{ error }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import ChoyActionButton from '../button/ChoyActionButton.vue';
import { createTranslate } from '@/web/web/i18n';
import { isConnectivityUnreachable } from '@/web/web/stores/connectivityStore';

/**
 * Chatter composer: local draft only; emits post with trimmed body.
 */
const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    posting?: boolean;
    error?: string | null;
    placeholder?: string;
    postLabel?: string;
  }>(),
  {
    disabled: false,
    posting: false,
    error: null,
    placeholder: undefined,
    postLabel: undefined,
  },
);

const emit = defineEmits<{
  post: [body: string];
}>();

const { _t } = createTranslate('web', { scope: 'web/components/chatter/ChoyChatterComposer' });

const body = ref('');
const canSubmit = computed(() => body.value.trim().length > 0);
const resolvedPlaceholder = computed(() => props.placeholder?.trim() || _t('Write a comment...'));
const resolvedPostLabel = computed(() => props.postLabel?.trim() || _t('Post'));
const postingLabel = computed(() => _t('Posting…'));

function submit(): void {
  const text = body.value.trim();
  // Match Post ActionButton: keyboard Ctrl/Cmd+Enter must not bypass offline gate.
  if (!text || props.posting || props.disabled || isConnectivityUnreachable()) return;
  emit('post', text);
}

/** Clears the draft after the host accepts a successful post. */
function clear(): void {
  body.value = '';
}

/** Test/host helper: set the draft body without relying on DOM v-model. */
function setDraft(text: string): void {
  body.value = String(text ?? '');
}

defineExpose({ clear, submit, setDraft });
</script>
