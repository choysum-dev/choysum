<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<!--
  List-store pagination adapter: maps limit/offset ↔ ChoyPagination page/pageSize.
  Not kit-exported — product API is ChoyPagination only.
  Delete this file once ListView consumes ChoyPagination directly.
-->

<template>
  <ChoyPagination
    :total="effective.total"
    v-model:page="pageModel"
    v-model:page-size="pageSizeModel"
  />
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { computed } from 'vue';
import type { BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import ChoyPagination from './ChoyPagination.vue';

const props = defineProps<{
  store: WebModelStore<T>;
  total?: number;
  limit?: number;
  offset?: number;
  page?: number;
  pageSize?: number;
}>();

const emit = defineEmits<{
  (e: 'paginateState', payload: { limit: number; offset: number }): void;
}>();

const effective = computed(() => {
  const total = Math.max(0, Number(props.total ?? 0));
  const pageSizeControlled = Number(props.pageSize ?? 0) > 0 ? Number(props.pageSize) : undefined;
  const pageControlled = Number(props.page ?? 0) > 0 ? Number(props.page) : undefined;
  const limitRaw = pageSizeControlled ?? (Number(props.limit ?? 20) || 20);
  const offsetRaw =
    pageControlled != null
      ? Math.max(0, (pageControlled - 1) * limitRaw)
      : Number.isFinite(Number(props.offset))
        ? Math.max(0, Number(props.offset))
        : 0;
  const currentPage = limitRaw > 0 ? Math.floor(offsetRaw / limitRaw) + 1 : 1;
  return { total, limit: limitRaw, offset: offsetRaw, currentPage, pageSize: limitRaw };
});

const pageModel = computed({
  get: () => effective.value.currentPage,
  set: (page: number) => {
    const size = effective.value.pageSize;
    const totalPages = size > 0 ? Math.max(1, Math.ceil(effective.value.total / size) || 1) : 1;
    const validPage = Math.max(1, Math.min(Math.floor(Number(page) || 1), totalPages));
    emit('paginateState', { limit: size, offset: (validPage - 1) * size });
  },
});

const pageSizeModel = computed({
  get: () => effective.value.pageSize,
  set: (size: number) => {
    const next = Number.isFinite(size) && size > 0 ? Math.floor(size) : effective.value.pageSize;
    emit('paginateState', { limit: next, offset: 0 });
  },
});
</script>
