<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="OSearchView"
    :show-header="showHeader"
    :action-ids="{ create: tokenActions.create, delete: tokenActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <template #header-right>
      <el-button-group>
        <el-tooltip :content="_t('List View')" placement="top"> <el-button :icon="FormatListBulletedOutlined" @click="toList" type="primary" /></el-tooltip>
        <el-tooltip :content="_t('Kanban View')" placement="top"
          ><el-button v-action="['auth.action.token_edit', 'auth.action.token_copy']" :icon="GridViewSharp" @click="toKanban"
        /></el-tooltip>
        <el-tooltip :content="_t('Icon View')" placement="top"
          ><el-button v-action.disable.and="['auth.action.token_edit', 'auth.action.token_delete']" :icon="BarChartOutlined" @click="toKanban"
        /></el-tooltip>
      </el-button-group>
    </template>

    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />

    <OVarCharField prop="UserId.Username" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="TokenType" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ODateTimeField prop="ExpiresAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <OBooleanField :store="store" prop="Revoked" widget="checkbox" />
    <ODateTimeField prop="RevokedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ODateTimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </OListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Token from '@/auth/service/models/token';
import OListView from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import { ElButton, ElTooltip, ElButtonGroup } from 'element-plus';
import { FormatListBulletedOutlined, GridViewSharp, BarChartOutlined } from '@vicons/material';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

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
function onRowClick(payload: RowEventPayload<Token>) {
  router.push(`/auth/tokens/${payload.row.Id}`);
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
