<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: userActions.create, edit: userActions.edit, copy: userActions.copy, delete: userActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="ufv-card">
      <template #header
        ><div class="ufv-card__header"><span>{{ _t('Account Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OImageField :store="store" prop="Avatar" :label="_t('Avatar')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Username" :label="_t('Username')" :rules="requiredRules" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Email" :label="_t('Email')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Phone" :label="_t('Phone')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" widget="switch" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="FirstName" :label="_t('First Name')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="LastName" :label="_t('Last Name')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="FullName" :label="_t('Full Name')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Language" :label="_t('Language')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Timezone" :label="_t('Timezone')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneRefField :store="store" prop="CompanyId" :label="_t('Company')" @value-click="onCompanyValueClick" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OJsonobjectField :store="store" prop="Preferences" :label="_t('Preferences')" :pretty="true" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="LastLogin" :label="_t('Last Login')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="CreatedAt" :label="_t('Created At')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="UpdatedAt" :label="_t('Updated At')" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24">
          <OManyToManyRefTagsField
            :store="store"
            prop="CompanyIds"
            :label="_t('Accessible Companies')"
            :search-list="CompanyListView"
            :search-view-title="_t('Select Company')"
            :tag-label-field="['Name', 'DisplayName', 'Code', 'Id']"
            @tag-click="onCompanyTagClick"
          />
        </el-col>
      </el-row>
    </el-card>

    <el-tabs model-value="roles" type="card" class="ufv-tabs">
      <el-tab-pane :label="_t('Roles')" name="roles">
        <OManyToManyField :store="store" prop="Roles" label="" :search-list="RoleListView" :search-view-title="_t('Select Role')">
          <OCharField :store="store" prop="Roles.Id" :label="_t('ID')" />
          <OVarCharField :store="store" prop="Roles.Name" :label="_t('Name')" />
          <ODateTimeField :store="store" prop="Roles.CreatedAt" :label="_t('Created At')" />
        </OManyToManyField>
      </el-tab-pane>

      <el-tab-pane :label="_t('Sessions')" name="sessions">
        <OOneToManyField :store="store" prop="Sessions" label="">
          <OCharField :store="store" prop="Sessions.Id" :label="_t('ID')" />
          <OVarCharField :store="store" prop="Sessions.Status" :label="_t('Status')" />
          <ODateTimeField :store="store" prop="Sessions.LastActivityAt" :label="_t('Last Activity')" />
          <ODateTimeField :store="store" prop="Sessions.CreatedAt" :label="_t('Created At')" />
        </OOneToManyField>
      </el-tab-pane>

      <el-tab-pane :label="_t('Tokens')" name="tokens">
        <OOneToManyField :store="store" prop="Tokens" label="">
          <OCharField :store="store" prop="Tokens.Id" :label="_t('ID')" />
          <OVarCharField :store="store" prop="Tokens.TokenType" :label="_t('Type')" />
          <ODateTimeField :store="store" prop="Tokens.CreatedAt" :label="_t('Created At')" />
        </OOneToManyField>
      </el-tab-pane>
    </el-tabs>
  </OFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { ClientModel, BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';

import type User from '@/auth/service/models/user';
import type Company from '@/base/service/models/company';
import { ElCard, ElRow, ElCol, ElTabs, ElTabPane } from 'element-plus';

import OFormView from '@/web/web/components/view/OFormView.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OCharField from '@/web/web/components/field/OCharField.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OImageField from '@/web/web/components/field/OImageField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OOneToManyField from '@/web/web/components/field/OOneToManyField.vue';
import OManyToManyField from '@/web/web/components/field/OManyToManyField.vue';
import RoleListView from '@/auth/web/views/RoleListView.vue';
import CompanyListView from '@/base/web/views/CompanyListView.vue';
import OJsonobjectField from '@/web/web/components/field/OJsonobjectField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import type { ValueClickPayload as ManyToOneRefValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import OManyToManyRefTagsField from '@/web/web/components/field/OManyToManyRefTagsField.vue';
import type { TagClickPayload as RefTagClickPayload } from '@/web/web/components/field/manyToManyTagsTypes';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'UserFormView', inheritAttrs: true });
const { _t } = createTranslate('auth', { scope: 'web/views/UserFormView' });
const { _t: _tRef } = createTranslate('auth', { output: 'reference', scope: 'web/views/UserFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store: WebModelStore<User>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  {
    showHeader: true,
    createAction: undefined,
  }
);

const { store, recordId, viewMode, showHeader, createAction } = props;
const userActions = defineModelActions('auth.User', { entityTitle: _tRef('User') });
const { hasAction } = usePermission();

const router = useRouter();

/**
 * Open the primary company record from the user form.
 */
function onCompanyValueClick(payload: ManyToOneRefValueClickPayload<Company>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CompanyDetail', params: { id } });
}

/**
 * Open a company record from the accessible-company tag list.
 */
function onCompanyTagClick(payload: RefTagClickPayload<Company>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CompanyDetail', params: { id } });
}
</script>

<style scoped>
.ufv-card {
  margin-bottom: 14px;
}
.ufv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.ufv-tabs {
  --el-tabs-header-height: 42px;
}
</style>
