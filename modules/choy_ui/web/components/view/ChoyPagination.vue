<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import ChoyButton from '../layout/ChoyButton.vue';
import type { ClassValue } from '../../lib/utils';
import {
  clampChoyPage,
  choyTotalPages,
} from './paginationHelpers';

/**
 * List pagination: 1-based page, Prev/Next, and page/total summary.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    total: number;
    disabled?: boolean;
  }>(),
  {
    disabled: false,
  },
);

const page = defineModel<number>('page', { default: 1 });
const pageSize = defineModel<number>('pageSize', { default: 20 });

const totalPages = computed(() => choyTotalPages(props.total, pageSize.value));

const currentPage = computed(() => clampChoyPage(page.value, totalPages.value));

const canPrev = computed(() => currentPage.value > 1 && !props.disabled);
const canNext = computed(
  () => currentPage.value < totalPages.value && !props.disabled,
);

function goPrev(): void {
  if (!canPrev.value) {
    return;
  }
  page.value = currentPage.value - 1;
}

function goNext(): void {
  if (!canNext.value) {
    return;
  }
  page.value = currentPage.value + 1;
}
</script>

<template>
  <div
    data-anchor="choy.pagination"
    :class="[
      'choy-pagination flex flex-wrap items-center gap-3 text-sm text-foreground',
      props.class,
    ]"
  >
    <ChoyButton
      type="button"
      variant="outline"
      size="sm"
      :disabled="!canPrev"
      @click="goPrev"
    >
      Prev
    </ChoyButton>
    <span class="choy-pagination__summary tabular-nums">
      Page {{ currentPage }} of {{ totalPages }}
      <span class="text-foreground/60">· {{ total }} total</span>
    </span>
    <ChoyButton
      type="button"
      variant="outline"
      size="sm"
      :disabled="!canNext"
      @click="goNext"
    >
      Next
    </ChoyButton>
  </div>
</template>
