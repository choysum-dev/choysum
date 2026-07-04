<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-pagination">
    <span class="o-pagination__text">
      第
      <span class="o-pagination__editable-wrapper">
        <el-input-number
          v-if="editingStart"
          ref="startInputRef"
          v-model="tempStart"
          :min="1"
          :max="paginationRange.total"
          :step="1"
          :precision="0"
          size="small"
          :controls="false"
          class="o-pagination__input"
          @blur="finishEditStart"
          @keydown="handleStartKeydown"
          @change="handleStartChange"
        />
        <span v-else class="o-pagination__editable" @click="startEditStart" @mouseenter="handleStartMouseEnter" @mouseleave="handleStartMouseLeave">
          {{ paginationRange.start }}
        </span>
      </span>
      -
      <span class="o-pagination__editable-wrapper">
        <el-input-number
          v-if="editingEnd"
          ref="endInputRef"
          v-model="tempEnd"
          :min="1"
          :max="paginationRange.total"
          :step="1"
          :precision="0"
          size="small"
          :controls="false"
          class="o-pagination__input"
          @blur="finishEditEnd"
          @keydown="handleEndKeydown"
          @change="handleEndChange"
        />
        <span v-else class="o-pagination__editable" @click="startEditEnd" @mouseenter="handleEndMouseEnter" @mouseleave="handleEndMouseLeave">
          {{ paginationRange.end }}
        </span>
      </span>
      行，共 {{ paginationRange.total }} 行
    </span>
    <div class="o-pagination__controls">
      <el-button size="small" :disabled="!paginationRange.canGoPrev" @click="goToPrevPage">
        <el-icon><ArrowLeft /></el-icon>
      </el-button>
      <el-button size="small" :disabled="!paginationRange.canGoNext" @click="goToNextPage">
        <el-icon><ArrowRight /></el-icon>
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { computed, ref, nextTick } from 'vue';
import { ArrowLeft, ArrowRight } from '@element-plus/icons-vue';
import { ElInputNumber } from 'element-plus';
import type { BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';

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

const startInputRef = ref<InstanceType<typeof ElInputNumber>>();
const endInputRef = ref<InstanceType<typeof ElInputNumber>>();

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
    text: `第 ${start} - ${end} 行，共 ${total} 行`,
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

function handleStartChange(value: number | undefined) {
  if (value !== undefined && value !== null) {
    tempStart.value = Math.floor(value);
  }
}

function handleEndChange(value: number | undefined) {
  if (value !== undefined && value !== null) {
    tempEnd.value = Math.floor(value);
  }
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

<style lang="scss" scoped>
.o-pagination {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}

.o-pagination__text {
  font-size: 14px;
  color: var(--el-text-color-regular);
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 4px;
}

.o-pagination__editable-wrapper {
  display: inline-flex;
  align-items: center;
  position: relative;
}

.o-pagination__editable {
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 4px;
  transition: all 0.2s ease;
  border: 1px solid transparent;
  min-width: 20px;
  text-align: center;
  display: inline-block;
}

.o-pagination__editable:hover,
.o-pagination__editable.hover {
  background-color: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  color: var(--el-color-primary);
}

.o-pagination__input {
  width: 60px !important;
  :deep(.el-input__inner) {
    padding: 2px 6px;
    height: auto;
    min-height: 24px;
    font-size: 14px;
    text-align: end;
  }
  :deep(.el-input__wrapper) {
    padding: 0 2px !important;
  }
}

.o-pagination__controls {
  display: flex;
  gap: 4px;
}

/* Responsive adjustments. */
@media (max-width: 768px) {
  .o-pagination {
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }

  .o-pagination__text {
    font-size: 12px;
  }

  .o-pagination__input {
    width: 50px !important;
  }
}
</style>
