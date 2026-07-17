<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: languageActions.create, edit: languageActions.edit, copy: languageActions.copy, delete: languageActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>{{ _t('Language Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Name" :label="_t('Name')" :rules="requiredRules" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Code" :label="_t('Code')" :rules="requiredRules" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OSelectionField :store="store" prop="Direction" :label="_t('Direction')" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField :store="store" prop="DefaultLocaleId" :label="_t('Default Locale')" :search-view="LocaleListView" :search-view-title="_t('Select Locale')"
        /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OBooleanField :store="store" prop="IsActive" :label="_t('Active')" /></el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Language from '@/base/service/models/language';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import LocaleListView from './LocaleListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'LanguageFormView', inheritAttrs: true });
const { _t } = createTranslate('base', { scope: 'web/views/LanguageFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store: WebModelStore<Language>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const languageActions = defineModelActions('base.Language', { entityTitle: 'Language' });
const { hasAction } = usePermission();
const { store, recordId, viewMode, showHeader, createAction } = props;
</script>

<style scoped>
.bfv-card {
  margin-bottom: 14px;
}
.bfv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
</style>
