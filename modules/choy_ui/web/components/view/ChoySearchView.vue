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
 * `keywordFields` documents which row fields a host may filter client-side.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    placeholder?: string;
    /** Documented keyword field names for host-side filtering (isolation). */
    keywordFields?: string[];
    disabled?: boolean;
  }>(),
  {
    placeholder: 'Search…',
    keywordFields: () => [],
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
  emit('query-update', buildChoySearchQuery(keyword.value));
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
      @keydown.enter.prevent="submit"
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
