<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.pagination"
    data-testid="choy-pagination"
    :class="[
      'choy-pagination flex h-control flex-wrap items-center gap-3 text-sm text-foreground',
      props.class,
    ]"
  >
    <span class="choy-pagination__total text-muted-foreground tabular-nums" aria-live="polite">
      {{ totalLabel }}
    </span>
    <label class="choy-pagination__size inline-flex items-center gap-2 text-muted-foreground">
      <span class="sr-only">{{ pageSizeLabel }}</span>
      <select
        class="h-control min-w-16 rounded-md border border-border bg-background px-2 text-sm text-foreground"
        :disabled="disabled"
        :aria-label="pageSizeLabel"
        :value="pageSize"
        @change="onPageSizeChange"
      >
        <option v-for="n in pageSizeOptions" :key="n" :value="n">{{ n }}</option>
      </select>
    </label>
    <div class="ms-auto flex items-center gap-2">
      <ChoyButton
        type="button"
        variant="outline"
        size="sm"
        :disabled="!canPrev"
        :aria-label="prevLabel"
        @click="goPrev"
      >
        {{ prevLabel }}
      </ChoyButton>
      <span
        class="choy-pagination__summary tabular-nums"
        aria-live="polite"
        aria-atomic="true"
      >
        {{ pageOfLabel }}
      </span>
      <ChoyButton
        type="button"
        variant="outline"
        size="sm"
        :disabled="!canNext"
        :aria-label="nextLabel"
        @click="goNext"
      >
        {{ nextLabel }}
      </ChoyButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue';
import ChoyButton from '../layout/ChoyButton.vue';
import type { ClassValue } from '../../lib/utils';
import { createTranslate } from '../../i18n';
import { clampChoyPage, choyTotalPages } from './paginationHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/view/Pagination' });

/**
 * Product pagination (Dense Admin §5.6): total + pageSize select + prev/next.
 * Changing pageSize resets to page 1. Kit exports only this pagination API.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    total: number;
    disabled?: boolean;
    pageSizeOptions?: number[];
  }>(),
  {
    disabled: false,
    pageSizeOptions: () => [10, 20, 50, 80],
  },
);

const page = defineModel<number>('page', { default: 1 });
const pageSize = defineModel<number>('pageSize', { default: 20 });

const totalPages = computed(() => choyTotalPages(props.total, pageSize.value));
const currentPage = computed(() => clampChoyPage(page.value, totalPages.value));

watch(pageSize, () => {
  if (page.value !== 1) {
    page.value = 1;
  }
});

watch(
  [page, totalPages],
  () => {
    const clamped = clampChoyPage(page.value, totalPages.value);
    if (page.value !== clamped) {
      page.value = clamped;
    }
  },
  { immediate: true },
);

const canPrev = computed(() => currentPage.value > 1 && !props.disabled);
const canNext = computed(
  () => currentPage.value < totalPages.value && !props.disabled,
);

const totalLabel = computed(() => _t('%s total', props.total));
const pageSizeLabel = computed(() => _t('Rows per page'));
const pageOfLabel = computed(() =>
  _t('Page %s of %s', currentPage.value, totalPages.value),
);
const prevLabel = computed(() => _t('Previous'));
const nextLabel = computed(() => _t('Next'));

function onPageSizeChange(event: Event): void {
  const raw = Number((event.target as HTMLSelectElement).value);
  if (!Number.isFinite(raw) || raw <= 0) return;
  pageSize.value = raw;
}

function goPrev(): void {
  if (!canPrev.value) return;
  page.value = currentPage.value - 1;
}

function goNext(): void {
  if (!canNext.value) return;
  page.value = currentPage.value + 1;
}
</script>
