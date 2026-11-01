<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{
      create: appSettingActions.create,
      edit: appSettingActions.edit,
      copy: appSettingActions.copy,
      delete: appSettingActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <div data-anchor="choy.form.sheet" class="flex min-h-full flex-col gap-8">
      <ChoySection :title="_t('Parameter')">
        <ChoyGrid :cols="12">
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="Key"
              :label="_t('Key')"
              :rules="requiredRules"
            />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyTextField
              :store="store"
              prop="Value"
              :label="_t('Value')"
              :rules="requiredRules"
            />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>
    </div>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyCol,
  ChoyFormView,
  ChoyGrid,
  ChoySection,
  ChoyTextField,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';
import { appSettingActions } from './app_setting_actions';

defineOptions({ name: 'AppSettingFormView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/AppSettingFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<any>;
    application: string;
    recordId?: string;
    initialValues?: Record<string, unknown>;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  {
    showHeader: true,
    createAction: undefined,
  }
);

const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'AppSettingFormView');
const { recordId, initialValues, viewMode, showHeader, createAction } = props;
</script>
