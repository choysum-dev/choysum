<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: languageActions.create, edit: languageActions.edit, copy: languageActions.copy, delete: languageActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
    @update-success="onLanguageSaved"
    @create-success="onLanguageSaved"
  >
    <ChoyCard :title="_t('Language Information')" class="bfv-card"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Code" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoySelectionField :store="store" prop="Direction" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyBooleanField :store="store" prop="IsActive" /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
    <ChoyCard :title="_t('Format')" class="bfv-card"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="DecimalSeparator" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="ThousandSeparator" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Grouping" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="DateFormat" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="TimeFormat" /></ChoyCol>
        <ChoyCol :span="4"><ChoyIntField :store="store" prop="FirstDayOfWeek" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoySelectionField :store="store" prop="CurrencySymbolPosition" /></ChoyCol>
        <ChoyCol :span="4"><ChoyBooleanField :store="store" prop="CurrencySymbolSpacing" /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Language from '@/base/service/models/language';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyFormView, ChoyGrid, ChoyNumberField, ChoySelectionField, ChoyVarcharField, ChoyIntField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

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
</style>
