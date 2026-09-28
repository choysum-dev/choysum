<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="token-list-view">
    <div v-if="showHeader" class="token-list-view__view-switch mb-2 flex justify-end">
      <div class="flex items-center gap-1">
        <ChoyButton size="sm" :title="_t('List View')" @click="toList">
          <List class="size-4" aria-hidden="true" />
        </ChoyButton>
        <ChoyButton
          v-action="['auth.action.token_edit', 'auth.action.token_copy']"
          variant="outline"
          size="sm"
          :title="_t('Kanban View')"
          @click="toKanban"
        >
          <LayoutGrid class="size-4" aria-hidden="true" />
        </ChoyButton>
        <ChoyButton
          v-action.disable.and="['auth.action.token_edit', 'auth.action.token_delete']"
          variant="outline"
          size="sm"
          :title="_t('Icon View')"
          @click="toKanban"
        >
          <BarChart3 class="size-4" aria-hidden="true" />
        </ChoyButton>
      </div>
    </div>
    <ChoyListView
      ref="listRef"
      v-bind="$attrs"
      :store="store"
      :searchView="ChoySearchView"
      :show-header="showHeader"
      :action-ids="{ create: tokenActions.create, delete: tokenActions.delete }"
      :has-action="hasAction"
      @row-click="onRowClick"
    >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />

    <ChoyVarcharField prop="UserId.Username" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="TokenType" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyDatetimeField prop="ExpiresAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyBooleanField :store="store" prop="Revoked" widget="checkbox" />
    <ChoyDatetimeField prop="RevokedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    </ChoyListView>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Token from '@/auth/service/models/token';
import { BarChart3, LayoutGrid, List } from 'lucide-vue-next';
import { ChoyButton } from '@/web';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyBooleanField, ChoyDatetimeField, ChoyListView, ChoySearchView, ChoyVColumn, ChoyVarcharField } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';

defineOptions({ name: 'TokenListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/TokenListView' });


const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Token>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'TokenListView');
const { showHeader } = props;
const tokenActions = defineModelActions('auth.Token', { entityTitle: _lt('Token') });
const { hasAction } = usePermission();

/**
 * Open the clicked token record in detail view.
 */
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/auth/tokens/${id}`);
}

/**
 * Switch from the token list to the list route.
 */
function toList() {
  router.push('/auth/tokens');
}

/**
 * Switch from the token list to the kanban route.
 */
function toKanban() {
  router.push('/auth/tokens/kanban');
}

const { listRef, expose } = useListViewExpose<Token>();
defineExpose(expose);
</script>
