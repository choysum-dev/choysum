<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: stateActions.create, edit: stateActions.edit, copy: stateActions.copy, delete: stateActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('State/Province Information')" class="bfv-card"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Code" /></ChoyCol>
        <ChoyCol :span="4"><ChoyManyToOneField :store="store" prop="CountryId" :search-view="CountryListView" :search-view-title="_t('Select Country')"
        /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyBooleanField :store="store" prop="IsActive" /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type State from '@/base/service/models/state';
import CountryListView from './CountryListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoyVarcharField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'StateFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/StateFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store?: WebModelStore<State>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const stateActions = defineModelActions('base.State', { entityTitle: _lt('State') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'StateFormView');
const { recordId, viewMode, showHeader, createAction } = props;
</script>

<style scoped>
.bfv-card {
  margin-bottom: 14px;
}
</style>
