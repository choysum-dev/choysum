<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{
      create: fieldDefaultActions.create,
      edit: fieldDefaultActions.edit,
      copy: fieldDefaultActions.copy,
      delete: fieldDefaultActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <div data-anchor="choy.form.sheet" class="flex min-h-full flex-col gap-8">
      <ChoySection :title="_t('Target')">
        <ChoyGrid :cols="12">
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="Model"
              :label="_t('Model')"
              :rules="requiredRules"
            />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="Field"
              :label="_t('Field')"
              :rules="requiredRules"
            />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Scope')">
        <ChoyGrid :cols="12">
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="UserId"
              :label="_t('User Id')"
            />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="CompanyId"
              :label="_t('Company Id')"
            />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Value')">
        <ChoyJsonField :store="store" prop="Value" hide-label :rules="requiredRules" />
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
  ChoyJsonField,
  ChoySection,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';
import { fieldDefaultActions } from './field_default_actions';

defineOptions({ name: 'FieldDefaultFormView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/FieldDefaultFormView' });
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
const store = resolvePageStore(props.store, 'FieldDefaultFormView');
const { recordId, initialValues, viewMode, showHeader, createAction } = props;
</script>
