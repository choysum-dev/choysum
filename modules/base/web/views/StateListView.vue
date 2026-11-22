<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    :action-ids="{ create: stateActions.create, delete: stateActions.delete }"
    :has-action="hasAction"
    detail-path="/base/states/:id"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Code" />
    <ChoyManyToOneField :store="store" prop="CountryId"><ChoyVarcharField :store="store" prop="CountryId.Name" /></ChoyManyToOneField>
    <ChoyBooleanField :store="store" prop="IsActive" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import type State from '@/base/service/models/state';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoyTableColumn, ChoyVarcharField} from '@/web';

defineOptions({ name: 'StateListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/StateListView' });
const props = defineProps<{ store?: ModelStore<State> }>();
const store = resolvePageStore(props.store, 'StateListView');
const stateActions = defineModelActions('base.State', { entityTitle: _lt('State') });
const { hasAction } = usePermission();
const { listRef, expose } = useListViewExpose<State>();
defineExpose(expose);
</script>
