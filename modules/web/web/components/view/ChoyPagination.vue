<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.pagination"
    data-testid="choy-pagination"
    :class="[
      'choy-pagination flex h-control flex-wrap items-center gap-2 text-sm text-foreground',
      props.class,
    ]"
  >
    <div class="choy-pagination__range inline-flex items-center gap-1 text-sm tabular-nums text-muted-foreground">
      <!-- Shared narrow box so idle ↔ edit does not shift "/ total" or the pager buttons. -->
      <input
        v-if="editing"
        ref="rangeInputRef"
        type="text"
        inputmode="text"
        class="choy-pagination__range-input box-border h-control w-14 shrink-0 rounded-md border border-input bg-background px-1 text-center text-sm tabular-nums text-foreground outline-none focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 disabled:opacity-50"
        :disabled="disabled"
        :aria-label="rangeEditAriaLabel"
        :value="editDraft"
        @input="onEditInput"
        @keydown="onEditKeydown"
        @blur="commitEdit"
      />
      <button
        v-else
        type="button"
        class="choy-pagination__range-value box-border inline-flex h-control w-14 shrink-0 items-center justify-center rounded-md border border-transparent bg-transparent px-1 text-sm tabular-nums text-foreground underline decoration-muted-foreground/60 underline-offset-4 hover:decoration-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50"
        :disabled="disabled || normalizedTotal <= 0"
        :aria-label="rangeEditAriaLabel"
        @click="startEdit"
      >
        {{ rangeValueLabel }}
      </button>
      <span aria-hidden="true">/</span>
      <span class="choy-pagination__total tabular-nums" aria-live="polite">{{ normalizedTotal }}</span>
    </div>
    <ChoyButtonGroup>
      <ChoyButton
        type="button"
        variant="outline"
        size="icon"
        data-testid="choy-pagination-prev"
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
        data-testid="choy-pagination-next"
        :disabled="!canNext"
        :aria-label="nextLabel"
        @click="goNext"
      >
        <ChevronRight class="size-4" aria-hidden="true" />
      </ChoyButton>
    </ChoyButtonGroup>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { ChevronLeft, ChevronRight } from 'lucide-vue-next';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyButtonGroup from '../layout/ChoyButtonGroup.vue';
import type { ClassValue } from '../../lib/utils';
import { createTranslate } from '../../i18n';
import {
  applyChoyPagerRangeEdit,
  clampChoyPage,
  choyTotalPages,
  formatChoyPagerRangeValue,
} from './paginationHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/view/Pagination' });

/**
 * Product pagination (Odoo-style): editable range + total + icon prev/next.
 * Page size stays on the model for the list engine; there is no rows-per-page select.
 * Editing "1-12" can jump by start index, or widen/narrow the window (updates pageSize).
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

const editing = ref(false);
const editDraft = ref('');
const rangeInputRef = ref<HTMLInputElement | null>(null);
const suppressBlurCommit = ref(false);
/** Skip page→1 reset while applying an edited range that sets both models. */
const applyingRangeEdit = ref(false);

const totalPages = computed(() => choyTotalPages(props.total, pageSize.value));
const currentPage = computed(() => clampChoyPage(page.value, totalPages.value));

watch(pageSize, () => {
  if (applyingRangeEdit.value) return;
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

/** Positive integer page size for range math (matches list default / applyChoyPagerRangeEdit). */
const effectivePageSize = computed(() => {
  const n = Number(pageSize.value);
  if (!Number.isFinite(n) || n <= 0) return 20;
  return Math.max(1, Math.floor(n));
});
const normalizedTotal = computed(() => {
  const n = Number(props.total);
  if (!Number.isFinite(n) || n < 0) return 0;
  return Math.floor(n);
});
const rangeStart = computed(() => {
  if (normalizedTotal.value <= 0) return 0;
  return (currentPage.value - 1) * effectivePageSize.value + 1;
});
const rangeEnd = computed(() => {
  if (normalizedTotal.value <= 0) return 0;
  return Math.min(currentPage.value * effectivePageSize.value, normalizedTotal.value);
});
const rangeValueLabel = computed(() =>
  formatChoyPagerRangeValue(rangeStart.value, rangeEnd.value),
);
const rangeEditLabel = computed(() => _t('Edit page range'));
const rangeEditAriaLabel = computed(
  () => `${rangeEditLabel.value}: ${rangeValueLabel.value} / ${normalizedTotal.value}`,
);
const prevLabel = computed(() => _t('Previous'));
const nextLabel = computed(() => _t('Next'));

function onEditInput(event: Event): void {
  editDraft.value = (event.target as HTMLInputElement | null)?.value ?? '';
}

async function startEdit(): Promise<void> {
  if (props.disabled || normalizedTotal.value <= 0) return;
  editing.value = true;
  editDraft.value = rangeValueLabel.value;
  await nextTick();
  const el = rangeInputRef.value;
  if (el && typeof el.focus === 'function') {
    el.focus();
  }
  if (el && typeof el.select === 'function') {
    el.select();
  }
}

/** Test / programmatic: open edit with an explicit draft string. */
async function startEditWith(draft: string): Promise<void> {
  await startEdit();
  editDraft.value = draft;
}

function cancelEdit(): void {
  suppressBlurCommit.value = true;
  editing.value = false;
  editDraft.value = '';
  queueMicrotask(() => {
    suppressBlurCommit.value = false;
  });
}

async function commitEdit(): Promise<void> {
  if (suppressBlurCommit.value || !editing.value) return;
  const draft = editDraft.value;
  // Opening the editor pre-fills the current window; committing it untouched
  // must not rewrite pageSize (a partial last page would shrink the window).
  if (draft.trim() === rangeValueLabel.value) {
    editing.value = false;
    editDraft.value = '';
    return;
  }
  const next = applyChoyPagerRangeEdit({
    draft,
    total: props.total,
    pageSize: pageSize.value,
  });
  editing.value = false;
  editDraft.value = '';
  if (!next) return;
  applyingRangeEdit.value = true;
  try {
    if (next.pageSize !== pageSize.value) {
      pageSize.value = next.pageSize;
    }
    page.value = next.page;
    await nextTick();
  } finally {
    applyingRangeEdit.value = false;
  }
}

function onEditKeydown(event: KeyboardEvent): void {
  if (event.key === 'Enter') {
    event.preventDefault();
    commitEdit();
  } else if (event.key === 'Escape') {
    event.preventDefault();
    cancelEdit();
  }
}

function goPrev(): void {
  if (!canPrev.value) return;
  page.value = currentPage.value - 1;
}

function goNext(): void {
  if (!canNext.value) return;
  page.value = currentPage.value + 1;
}

defineExpose({
  startEdit,
  startEditWith,
  commitEdit,
  cancelEdit,
  goPrev,
  goNext,
  onEditKeydown,
  onEditInput,
});
</script>
