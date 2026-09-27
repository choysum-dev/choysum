<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-chatter-composer flex flex-col gap-2" data-anchor="choy.chatter.composer">
    <Textarea
      v-model="body"
      :rows="3"
      :placeholder="placeholder"
      :disabled="posting || disabled"
      @keydown.ctrl.enter="!$event.isComposing && ($event.preventDefault(), submit())"
      @keydown.meta.enter="!$event.isComposing && ($event.preventDefault(), submit())"
    />
    <div class="flex justify-end">
      <Button
        size="sm"
        :disabled="disabled || posting || !canSubmit"
        @click="submit"
      >
        {{ posting ? 'Posting…' : postLabel }}
      </Button>
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
import Button from '../vendor/ui/button/Button.vue';
import Textarea from '../vendor/ui/textarea/Textarea.vue';

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
    placeholder: 'Write a comment...',
    postLabel: 'Post',
  },
);

const emit = defineEmits<{
  post: [body: string];
}>();

const body = ref('');
const canSubmit = computed(() => body.value.trim().length > 0);

function submit(): void {
  const text = body.value.trim();
  if (!text || props.posting || props.disabled) return;
  emit('post', text);
}

/** Clears the draft after the host accepts a successful post. */
function clear(): void {
  body.value = '';
}

defineExpose({ clear });
</script>
