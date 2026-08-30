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
    @update-success="onLanguageSaved"
    @create-success="onLanguageSaved"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>{{ _t('Language Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Name" :rules="requiredRules" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Code" :rules="requiredRules" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OSelectionField :store="store" prop="Direction" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OBooleanField :store="store" prop="IsActive" /></el-col>
      </el-row>
    </el-card>
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>{{ _t('Format') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="DecimalSeparator" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="ThousandSeparator" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Grouping" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="DateFormat" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="TimeFormat" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OIntField :store="store" prop="FirstDayOfWeek" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OSelectionField :store="store" prop="CurrencySymbolPosition" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OBooleanField :store="store" prop="CurrencySymbolSpacing" /></el-col>
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
import OIntField from '@/web/web/components/field/OIntField.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';

defineOptions({ name: 'LanguageFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/LanguageFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store?: WebModelStore<Language>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const languageActions = defineModelActions('base.Language', { entityTitle: _lt('Language') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'LanguageFormView');
const { recordId, viewMode, showHeader, createAction } = props;

/** Refresh FE format overlays so list/number displays pick up Language field changes. */
async function onLanguageSaved() {
  try {
    await useI18nStore().loadActiveUiKeysFromServer();
  } catch {
    // Best-effort; next full page init will reload formats.
  }
}
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
