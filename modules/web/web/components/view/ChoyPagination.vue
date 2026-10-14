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
    <span
      class="choy-pagination__range text-sm text-muted-foreground tabular-nums"
      aria-live="polite"
    >
      {{ rangeLabel }}
    </span>
    <label class="choy-pagination__size inline-flex items-center gap-2 text-muted-foreground">
      <span class="sr-only">{{ pageSizeLabel }}</span>
      <NativeSelect
        class="min-w-16"
        :disabled="disabled"
        :aria-label="pageSizeLabel"
        v-model="pageSizeModel"
      >
        <NativeSelectOption v-for="n in effectivePageSizeOptions" :key="n" :value="String(n)">
          {{ n }}
        </NativeSelectOption>
      </NativeSelect>
    </label>
    <!-- role=group mirrors ButtonGroup; avoid barrel import (sibling SFCs break FE-QJS compile). -->
    <div
      role="group"
      data-slot="button-group"
      class="ms-auto flex w-fit items-stretch [&>*:not(:first-child)]:rounded-l-none [&>*:not(:first-child)]:border-l-0 [&>*:not(:last-child)]:rounded-r-none"
    >
      <ChoyButton
        type="button"
        variant="outline"
        size="icon"
        :disabled="!canPrev"
        :aria-label="prevLabel"
        @click="goPrev"
      >
        <ChevronLeft class="size-4" aria-hidden="true" />
      </ChoyButton>
      <ChoyButton
        type="button"
        variant="outline"
        size="icon"
        :disabled="!canNext"
        :aria-label="nextLabel"
        @click="goNext"
      >
        <ChevronRight class="size-4" aria-hidden="true" />
      </ChoyButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue';
import { ChevronLeft, ChevronRight } from 'lucide-vue-next';
import ChoyButton from '../layout/ChoyButton.vue';
import NativeSelect from '../vendor/ui/native-select/NativeSelect.vue';
import NativeSelectOption from '../vendor/ui/native-select/NativeSelectOption.vue';
import type { ClassValue } from '../../lib/utils';
import { createTranslate } from '../../i18n';
import { clampChoyPage, choyTotalPages } from './paginationHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/view/Pagination' });

/**
 * Product pagination (Dense Admin): range + pageSize NativeSelect + icon prev/next.
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

const pageSizeModel = computed({
  get: () => String(pageSize.value),
  set: (value: unknown) => {
    onPageSizeModel(value);
  },
});

const effectivePageSizeOptions = computed(() => {
  const opts = [...props.pageSizeOptions];
  if (Number.isFinite(pageSize.value) && pageSize.value > 0 && !opts.includes(pageSize.value)) {
    opts.push(pageSize.value);
    opts.sort((a, b) => a - b);
  }
  return opts;
});

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

const rangeStart = computed(() => {
  if (!Number.isFinite(props.total) || props.total <= 0) return 0;
  return (currentPage.value - 1) * pageSize.value + 1;
});
const rangeEnd = computed(() => {
  if (!Number.isFinite(props.total) || props.total <= 0) return 0;
  return Math.min(currentPage.value * pageSize.value, props.total);
});
const rangeLabel = computed(() =>
  _t('%s–%s / %s', rangeStart.value, rangeEnd.value, props.total),
);
const pageSizeLabel = computed(() => _t('Rows per page'));
const prevLabel = computed(() => _t('Previous'));
const nextLabel = computed(() => _t('Next'));

function onPageSizeModel(value: unknown): void {
  const raw = Number(value);
  if (!Number.isFinite(raw) || raw <= 0) return;
  pageSize.value = raw;
}

/** @deprecated Kept for unit tests that call setupState().onPageSizeChange */
function onPageSizeChange(event: Event): void {
  onPageSizeModel((event.target as HTMLSelectElement | null)?.value);
}

function goPrev(): void {
  if (!canPrev.value) return;
  page.value = currentPage.value - 1;
}

function goNext(): void {
  if (!canNext.value) return;
  page.value = currentPage.value + 1;
}

defineExpose({ onPageSizeChange, goPrev, goNext });
</script>
