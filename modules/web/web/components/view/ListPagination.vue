<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<!--
  List-store pagination adapter (editable range + prev/next).
  Not kit-exported — product API is ChoyPagination only (Dense Admin W2).
  Delete in W6 once ListView consumes ChoyPagination directly.
-->

<template>
  <div class="flex items-center justify-end gap-3 max-md:flex-col max-md:items-center max-md:gap-2">
    <span class="flex items-center gap-1 whitespace-nowrap text-sm text-foreground max-md:text-xs">
      <span class="relative inline-flex items-center">
        <input
          v-if="editingStart"
          ref="startInputRef"
          v-model.number="tempStart"
          type="number"
          min="1"
          :max="paginationRange.total"
          step="1"
          class="h-auto min-h-6 w-[60px] rounded-sm border border-border bg-background px-1.5 py-0.5 text-end text-sm max-md:w-[50px]"
          @blur="finishEditStart"
          @keydown="handleStartKeydown"
        />
        <span v-else class="choy-pagination__editable inline-block min-w-5 cursor-pointer rounded border border-transparent px-1 text-center transition-colors hover:border-border hover:bg-muted hover:text-primary" @click="startEditStart" @mouseenter="handleStartMouseEnter" @mouseleave="handleStartMouseLeave">
          {{ paginationRange.start }}
        </span>
      </span>
      -
      <span class="relative inline-flex items-center">
        <input
          v-if="editingEnd"
          ref="endInputRef"
          v-model.number="tempEnd"
          type="number"
          min="1"
          :max="paginationRange.total"
          step="1"
          class="h-auto min-h-6 w-[60px] rounded-sm border border-border bg-background px-1.5 py-0.5 text-end text-sm max-md:w-[50px]"
          @blur="finishEditEnd"
          @keydown="handleEndKeydown"
        />
        <span v-else class="choy-pagination__editable inline-block min-w-5 cursor-pointer rounded border border-transparent px-1 text-center transition-colors hover:border-border hover:bg-muted hover:text-primary" @click="startEditEnd" @mouseenter="handleEndMouseEnter" @mouseleave="handleEndMouseLeave">
          {{ paginationRange.end }}
        </span>
      </span>
      {{ _t('of %s', paginationRange.total) }}
    </span>
    <div class="flex gap-1">
      <ChoyButton size="sm" :disabled="!paginationRange.canGoPrev" @click="goToPrevPage">
        <span class="inline-flex"><ArrowLeft /></span>
      </ChoyButton>
      <ChoyButton size="sm" :disabled="!paginationRange.canGoNext" @click="goToNextPage">
        <span class="inline-flex"><ArrowRight /></span>
      </ChoyButton>
    </div>
  </div>
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { computed, ref, nextTick } from 'vue';
import type { BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/view/Pagination' });

const props = defineProps<{
  store: WebModelStore<T>;
  // Controlled mode takes precedence over store-backed pagination.
  total?: number;
  limit?: number; // pageSize alias; page/pageSize still takes precedence.
  offset?: number; // (page-1)*pageSize
  page?: number; // Controlled current page, starting from 1.
  pageSize?: number; // Controlled page size.
}>();

// Keep paginateState as the single source of truth for limit and offset.
const emit = defineEmits<{
  (e: 'paginateState', payload: { limit: number; offset: number }): void;
}>();

const editingStart = ref(false);
const editingEnd = ref(false);
const tempStart = ref<number>(0);
const tempEnd = ref<number>(0);

const startInputRef = ref<HTMLInputElement | null>(null);
const endInputRef = ref<HTMLInputElement | null>(null);

// Controlled values override store-backed pagination state.
const effective = computed(() => {
  const total = Number(props.total ?? 0);
  // Priority: page/pageSize over limit/offset.
  const pageSizeControlled = Number(props.pageSize ?? 0) > 0 ? Number(props.pageSize) : undefined;
  const pageControlled = Number(props.page ?? 0) > 0 ? Number(props.page) : undefined;
  const limitRaw = pageSizeControlled ?? (Number(props.limit ?? 20) || 20);
  const offsetRaw =
    pageControlled != null ? Math.max(0, (pageControlled - 1) * limitRaw) : Number.isFinite(Number(props.offset)) ? Math.max(0, Number(props.offset)) : 0;
  const currentPage = limitRaw > 0 ? Math.floor(offsetRaw / limitRaw) + 1 : 1;
  return { total, limit: limitRaw, offset: offsetRaw, currentPage, pageSize: limitRaw };
});

const paginationRange = computed(() => {
  const { currentPage, pageSize } = effective.value;
  const total = Math.max(0, Number(effective.value.total || 0));
  const start = total === 0 ? 0 : Math.max((currentPage - 1) * pageSize + 1, 1);
  const end = total === 0 ? 0 : Math.min(currentPage * pageSize, total);
  const totalPages = pageSize > 0 ? Math.ceil(total / pageSize) || 1 : 1;
  return {
    start,
    end,
    total,
    currentPage,
    pageSize,
    totalPages,
    text: _t('%s - %s of %s', start, end, total),
    canGoPrev: currentPage > 1,
    canGoNext: currentPage < totalPages,
  };
});

async function goToPage(page: number) {
  const totalPages = paginationRange.value.totalPages;
  const pageSize = effective.value.pageSize;
  const validPage = Math.max(1, Math.min(page, totalPages));
  const nextOffset = (validPage - 1) * pageSize;
  emit('paginateState', { limit: pageSize, offset: nextOffset });
}

async function goToPrevPage() {
  if (paginationRange.value.canGoPrev) {
    await goToPage(effective.value.currentPage - 1);
  }
}

async function goToNextPage() {
  if (paginationRange.value.canGoNext) {
    await goToPage(effective.value.currentPage + 1);
  }
}

async function startEditStart() {
  editingStart.value = true;
  tempStart.value = paginationRange.value.start;
  await nextTick();
  startInputRef.value?.focus();
}

async function startEditEnd() {
  editingEnd.value = true;
  tempEnd.value = paginationRange.value.end;
  await nextTick();
  endInputRef.value?.focus();
}

async function finishEditStart() {
  if (tempStart.value > 0) {
    const total = Math.max(0, Number(effective.value.total || 0));
    const currentEnd = paginationRange.value.end || 1;
    const newStart = Math.max(1, Math.min(Math.floor(tempStart.value), total || 1));
    const newPageSize = Math.max(1, currentEnd - newStart + 1);
    const newPage = Math.ceil(newStart / newPageSize);
    const nextOffset = (newPage - 1) * newPageSize;
    emit('paginateState', { limit: newPageSize, offset: nextOffset });
  }
  editingStart.value = false;
  tempStart.value = 0;
}

async function finishEditEnd() {
  if (tempEnd.value > 0) {
    const total = Math.max(0, Number(effective.value.total || 0));
    const currentStart = paginationRange.value.start || 1;
    const newEnd = Math.max(1, Math.min(Math.floor(tempEnd.value), total || 1));
    const newPageSize = Math.max(1, newEnd - currentStart + 1);
    const newPage = Math.ceil(currentStart / newPageSize);
    const nextOffset = (newPage - 1) * newPageSize;
    emit('paginateState', { limit: newPageSize, offset: nextOffset });
  }
  editingEnd.value = false;
  tempEnd.value = 0;
}

function cancelEditStart() {
  editingStart.value = false;
  tempStart.value = 0;
}

function cancelEditEnd() {
  editingEnd.value = false;
  tempEnd.value = 0;
}

function handleStartKeydown(event: KeyboardEvent) {
  if (event.key === '.' || event.key === ',') {
    event.preventDefault();
    return;
  }
  if (event.key === 'Enter') {
    finishEditStart();
  } else if (event.key === 'Escape') {
    cancelEditStart();
  }
}

function handleEndKeydown(event: KeyboardEvent) {
  if (event.key === '.' || event.key === ',') {
    event.preventDefault();
    return;
  }
  if (event.key === 'Enter') {
    finishEditEnd();
  } else if (event.key === 'Escape') {
    cancelEditEnd();
  }
}

function handleStartMouseEnter(event: MouseEvent) {
  const target = event.target as HTMLElement;
  target.classList.add('hover');
}

function handleStartMouseLeave(event: MouseEvent) {
  const target = event.target as HTMLElement;
  target.classList.remove('hover');
}

function handleEndMouseEnter(event: MouseEvent) {
  const target = event.target as HTMLElement;
  target.classList.add('hover');
}

function handleEndMouseLeave(event: MouseEvent) {
  const target = event.target as HTMLElement;
  target.classList.remove('hover');
}
</script>

