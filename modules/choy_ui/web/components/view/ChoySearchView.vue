<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import Input from '../vendor/ui/input/Input.vue';
import ChoyButton from '../layout/ChoyButton.vue';
import type { ClassValue } from '../../lib/utils';
import {
  buildChoySearchQuery,
  type ChoySearchQuery,
} from './searchViewHelpers';

/**
 * Keyword search chrome. Emits a normalized ChoySearchQuery on submit.
 * Hosts decide which row fields to filter with filterRowsByKeyword.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    placeholder?: string;
    disabled?: boolean;
  }>(),
  {
    placeholder: 'Search…',
    disabled: false,
  },
);

const keyword = defineModel<string>('keyword', { default: '' });

const emit = defineEmits<{
  'query-update': [query: ChoySearchQuery];
}>();

function submit(): void {
  if (props.disabled) {
    return;
  }
  const query = buildChoySearchQuery(keyword.value);
  // Keep the bound input in sync with the normalized query.
  keyword.value = query.keyword;
  emit('query-update', query);
}

function onKeydown(event: KeyboardEvent): void {
  // Confirming an IME candidate also fires Enter; don't submit half-composed text.
  if (event.key !== 'Enter' || event.isComposing || event.keyCode === 229) {
    return;
  }
  event.preventDefault();
  submit();
}
</script>

<template>
  <div
    data-anchor="choy.search-view"
    :class="['choy-search-view flex flex-wrap items-center gap-2', props.class]"
  >
    <Input
      v-model="keyword"
      class="min-w-[12rem] flex-1"
      :placeholder="placeholder"
      :disabled="disabled"
      :aria-label="placeholder"
      @keydown="onKeydown"
    />
    <ChoyButton
      type="button"
      size="sm"
      :disabled="disabled"
      @click="submit"
    >
      Search
    </ChoyButton>
  </div>
</template>
