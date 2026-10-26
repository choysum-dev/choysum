<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: uomCategoryActions.create, edit: uomCategoryActions.edit, copy: uomCategoryActions.copy, delete: uomCategoryActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Unit of Measure Category')" class="mb-3.5"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Code" /></ChoyCol>
        <ChoyCol :span="4"><ChoyBooleanField :store="store" prop="IsActive" /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type UoMCategory from '@/base/service/models/uom_category';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyFormView, ChoyGrid, ChoyVarcharField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'UoMCategoryFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/UoMCategoryFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{
    store?: WebModelStore<UoMCategory>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  { showHeader: true, createAction: undefined }
);
const uomCategoryActions = defineModelActions('base.UoMCategory', { entityTitle: _lt('Unit of Measure Category') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'UoMCategoryFormView');
const { recordId, viewMode, showHeader, createAction } = props;
</script>

