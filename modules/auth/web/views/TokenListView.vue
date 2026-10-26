<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyListView
    ref="listRef"
    class="token-list-view"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    :show-header="showHeader"
    :action-ids="{ create: tokenActions.create, delete: tokenActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <!-- View-mode toggles only; ListView keeps pagination at the trailing edge. -->
    <template v-if="showHeader" #header-right>
      <ChoyViewSwitcher
        :aria-label="_t('View switcher')"
        :active="activeView"
        :options="viewSwitcherOptions"
        @select="onViewSelect"
      />
    </template>

    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />

    <ChoyVarcharField prop="UserId.Username" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="TokenType" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyDatetimeField prop="ExpiresAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyBooleanField :store="store" prop="Revoked" widget="checkbox" />
    <ChoyDatetimeField prop="RevokedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Token from '@/auth/service/models/token';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import {
  ChoyBooleanField,
  ChoyDatetimeField,
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
  ChoyViewSwitcher,
  useChoyViewRouteSwitch,
  type ChoyViewSwitcherOption,
  type RowEventPayload,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'TokenListView', inheritAttrs: false });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/TokenListView' });

const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Token>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  },
);

const store = resolvePageStore(props.store, 'TokenListView');
const { active: activeView, selectView: onViewSelect } = useChoyViewRouteSwitch({ store });
const { showHeader } = props;
const tokenActions = defineModelActions('auth.Token', { entityTitle: _lt('Token') });
const { hasAction } = usePermission();

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'list', label: _t('List View') },
  { value: 'kanban', label: _t('Kanban View') },
  { value: 'chart', label: _t('Chart View') },
]);

function onRowClick(payload: RowEventPayload) {
  const id = String(payload.row?.Id ?? '').trim();
  if (id) router.push(`/auth/tokens/${id}`);
}

const { listRef, expose } = useListViewExpose<Token>();
defineExpose(expose);
</script>
