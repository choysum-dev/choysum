<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OSearchView v-if="useStoreEngine" v-bind="(storeBind as any)" v-on="(storeListeners as any)">
    <slot />
  </OSearchView>

  <div
    v-else
    v-bind="($attrs as any)"
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

<script setup lang="ts">
import { computed, useAttrs } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useOptionalPageStore } from '@/web/web/composables/usePageContext';
import { hasChoyStoreEngine } from '@/web/web/composables/choyStoreMode';
import Input from '../vendor/ui/input/Input.vue';
import ChoyButton from '../layout/ChoyButton.vue';
import type { ClassValue } from '../../lib/utils';
import {
  buildChoySearchQuery,
  type ChoySearchQuery,
} from './searchViewHelpers';
import OSearchView from './OSearchView.vue';

defineOptions({ name: 'ChoySearchView', inheritAttrs: false });

/**
 * Search: store-bound mode hosts OSearchView; otherwise keyword chrome.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    placeholder?: string;
    disabled?: boolean;
    store?: WebModelStore<any>;
  }>(),
  {
    placeholder: 'Search…',
    disabled: false,
  },
);

const attrs = useAttrs();
const pageStore = useOptionalPageStore();
const useStoreEngine = computed(() => hasChoyStoreEngine(props.store, pageStore.value));

const storeBind = computed(() => ({
  ...attrs,
  store: props.store ?? pageStore.value,
  class: props.class,
}));

const storeListeners = computed(() => {
  const out: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(attrs)) {
    if (key.startsWith('on') && typeof value === 'function') {
      out[key] = value;
    }
  }
  return out;
});

const keyword = defineModel<string>('keyword', { default: '' });

const emit = defineEmits<{
  'query-update': [query: ChoySearchQuery];
}>();

function submit(): void {
  if (props.disabled) {
    return;
  }
  const query = buildChoySearchQuery(keyword.value);
  keyword.value = query.keyword;
  emit('query-update', query);
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Enter' || event.isComposing || event.keyCode === 229) {
    return;
  }
  event.preventDefault();
  submit();
}
</script>
