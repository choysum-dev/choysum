<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{
      create: propertyDefinitionActions.create,
      edit: propertyDefinitionActions.edit,
      copy: propertyDefinitionActions.copy,
      delete: propertyDefinitionActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <div data-anchor="choy.form.sheet" class="min-h-full">
      <ChoySection :title="_t('Scope')">
        <ChoyGrid :cols="12">
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="TargetModel"
              :label="_t('Target Model')"
              :rules="requiredRules"
            />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="PropertiesField"
              :label="_t('Properties Field')"
              :rules="requiredRules"
            />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="ContainerModel"
              :label="_t('Container Model')"
            />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="ContainerId"
              :label="_t('Container Id')"
            />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Definition')" class="mt-6">
        <ChoyPropertiesDefinitionField :store="store" prop="Definition" hide-label />
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
  ChoyPropertiesDefinitionField,
  ChoySection,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';
import { propertyDefinitionActions } from './property_definition_actions';

defineOptions({ name: 'PropertyDefinitionFormView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/PropertyDefinitionFormView' });
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
const store = resolvePageStore(props.store, 'PropertyDefinitionFormView');
const { recordId, initialValues, viewMode, showHeader, createAction } = props;
</script>
