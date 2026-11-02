<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Technical admin Form: Condition/GroupBy are JSON (not the OSearch builder).
    Prefer OSearch Favorites for end-user create/update of search state.
    Settings chrome uses the Saved Filters label for this admin Form.
  -->
  <ChoyFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{
      create: userFilterActions.create,
      edit: userFilterActions.edit,
      copy: userFilterActions.copy,
      delete: userFilterActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <div data-anchor="choy.form.sheet" class="flex min-h-full flex-col gap-8">
      <ChoySection :title="_t('Identity')">
        <ChoyGrid :cols="12">
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="Name"
              :label="_t('Name')"
              :rules="requiredRules"
            />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="ScopeKey"
              :label="_t('Scope Key')"
              :rules="requiredRules"
            />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="Application"
              :label="_t('Application')"
              :rules="requiredRules"
            />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="ModelName"
              :label="_t('Model Name')"
              :rules="requiredRules"
            />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Sharing')">
        <ChoyGrid :cols="12">
          <ChoyCol :span="6">
            <ChoyVarcharField
              :store="store"
              prop="UserId"
              :label="_t('User')"
              :help="_t('Empty means shared with all logged-in users. Cannot assign another user.')"
            />
          </ChoyCol>
          <ChoyCol :span="6">
            <ChoyBooleanField
              :store="store"
              prop="IsDefault"
              :label="_t('Is Default')"
              widget="switch"
            />
          </ChoyCol>
        </ChoyGrid>
      </ChoySection>

      <ChoySection :title="_t('Payload')">
        <ChoyGrid :cols="12">
          <ChoyCol :span="12">
            <ChoyJsonField
              :store="store"
              prop="Condition"
              :label="_t('Condition')"
              :rules="requiredRules"
              :allow-array="true"
            />
          </ChoyCol>
          <ChoyCol :span="12">
            <ChoyJsonField
              :store="store"
              prop="GroupBy"
              :label="_t('Group By')"
              :allow-array="true"
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
  ChoyBooleanField,
  ChoyCol,
  ChoyFormView,
  ChoyGrid,
  ChoyJsonField,
  ChoySection,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';
import { userFilterActions } from './user_filter_actions';

defineOptions({ name: 'UserFilterFormView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/UserFilterFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<any>;
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
const store = resolvePageStore(props.store, 'UserFilterFormView');
const { recordId, initialValues, viewMode, showHeader, createAction } = props;
</script>
